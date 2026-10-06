package example

import (
	"app/internal/cache"
	"app/internal/db"
	"app/internal/errs"
	"app/internal/middleware"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const listCacheTTL = 30 * time.Second

var (
	ErrExampleNotFound  = errs.NewNotFoundError(errs.ErrKeyExampleNotFound, "Example not found")
	ErrExampleForbidden = errs.NewForbiddenError(errs.ErrKeyForbidden, "You are not allowed to do this to this example")
)

// PaginatedExamplesResult is one page of a user's examples
type PaginatedExamplesResult struct {
	Data  []db.Example `json:"data"`
	Total int64        `json:"total"`
}

type ExampleService struct {
	queries *db.Queries
	cache   cache.Cache
}

// NewExampleService creates a new example service. cache may be nil (caching skipped).
func NewExampleService(queries *db.Queries, c cache.Cache) *ExampleService {
	return &ExampleService{
		queries: queries,
		cache:   c,
	}
}

func (s *ExampleService) CreateExample(ctx context.Context, actor middleware.Actor, title, description string) (*db.Example, error) {
	example, err := s.queries.CreateExample(ctx, db.CreateExampleParams{
		UserID:      actor.ID,
		Title:       title,
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
	if err != nil {
		return nil, errs.WrapDatabaseError(err)
	}

	s.invalidateList(ctx, example.UserID)
	return &example, nil
}

func (s *ExampleService) GetExample(ctx context.Context, actor middleware.Actor, exampleID int32) (*db.Example, error) {
	return s.authorized(ctx, actor, exampleID, CanView)
}

func (s *ExampleService) UpdateExample(ctx context.Context, actor middleware.Actor, exampleID int32, title, description string) (*db.Example, error) {
	if _, err := s.authorized(ctx, actor, exampleID, CanUpdate); err != nil {
		return nil, err
	}

	example, err := s.queries.UpdateExample(ctx, db.UpdateExampleParams{
		ID:          exampleID,
		Title:       title,
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
	if err != nil {
		return nil, notFoundOr(err)
	}

	s.invalidateList(ctx, example.UserID)
	return &example, nil
}

func (s *ExampleService) DeleteExample(ctx context.Context, actor middleware.Actor, exampleID int32) error {
	example, err := s.authorized(ctx, actor, exampleID, CanDelete)
	if err != nil {
		return err
	}

	if err := s.queries.DeleteExample(ctx, exampleID); err != nil {
		return errs.WrapDatabaseError(err)
	}

	s.invalidateList(ctx, example.UserID)
	return nil
}

// authorized loads an example and applies a policy to it: 404 when it does not exist,
// 403 when it does and the actor may not do this.
func (s *ExampleService) authorized(ctx context.Context, actor middleware.Actor, exampleID int32, allowed func(middleware.Actor, db.Example) bool) (*db.Example, error) {
	example, err := s.queries.GetExampleByID(ctx, exampleID)
	if err != nil {
		return nil, notFoundOr(err)
	}
	if !allowed(actor, example) {
		return nil, ErrExampleForbidden
	}
	return &example, nil
}

// ListExamplesPaginated returns one page of a user's examples, cached for a short time.
func (s *ExampleService) ListExamplesPaginated(ctx context.Context, userID, page, pageSize int32) (*PaginatedExamplesResult, error) {
	if s.cache == nil {
		return s.loadPage(ctx, userID, page, pageSize)
	}

	var result PaginatedExamplesResult
	err := s.cache.Remember(ctx, s.listCacheKey(ctx, userID, page, pageSize), listCacheTTL, func() (interface{}, error) {
		return s.loadPage(ctx, userID, page, pageSize)
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *ExampleService) loadPage(ctx context.Context, userID, page, pageSize int32) (*PaginatedExamplesResult, error) {
	examples, err := s.queries.ListExamplesForUserPaginated(ctx, db.ListExamplesForUserPaginatedParams{
		UserID: userID,
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	})
	if err != nil {
		return nil, errs.WrapDatabaseError(err)
	}

	total, err := s.queries.CountExamplesForUser(ctx, userID)
	if err != nil {
		return nil, errs.WrapDatabaseError(err)
	}

	if examples == nil {
		examples = []db.Example{}
	}
	return &PaginatedExamplesResult{Data: examples, Total: total}, nil
}

// The cache pattern for lists: every cached page carries the user's list version in its
// key, and a write changes the version. All pages go stale at once, whatever page sizes
// were requested, and the old entries simply expire.
func (s *ExampleService) listVersionKey(userID int32) string {
	return fmt.Sprintf("examples:user:%d:version", userID)
}

func (s *ExampleService) listCacheKey(ctx context.Context, userID, page, pageSize int32) string {
	var version int64
	_ = s.cache.Get(ctx, s.listVersionKey(userID), &version)
	return fmt.Sprintf("examples:user:%d:v%d:page:%d:size:%d", userID, version, page, pageSize)
}

func (s *ExampleService) invalidateList(ctx context.Context, userID int32) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Set(ctx, s.listVersionKey(userID), time.Now().UnixNano(), 24*time.Hour)
}

func notFoundOr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrExampleNotFound
	}
	return errs.WrapDatabaseError(err)
}
