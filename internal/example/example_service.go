package example

import (
	"app/internal/cache"
	"app/internal/db"
	"app/internal/errs"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const listCacheTTL = 30 * time.Second

var ErrExampleNotFound = errs.NewNotFoundError(errs.ErrKeyExampleNotFound, "Example not found")

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

func (s *ExampleService) CreateExample(ctx context.Context, userID int32, title, description string) (*db.Example, error) {
	example, err := s.queries.CreateExample(ctx, db.CreateExampleParams{
		UserID:      userID,
		Title:       title,
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
	if err != nil {
		return nil, errs.WrapDatabaseError(err)
	}

	s.invalidateList(ctx, userID)
	return &example, nil
}

func (s *ExampleService) GetExample(ctx context.Context, exampleID, userID int32) (*db.Example, error) {
	example, err := s.queries.GetExampleByID(ctx, db.GetExampleByIDParams{
		ID:     exampleID,
		UserID: userID,
	})
	if err != nil {
		return nil, notFoundOr(err)
	}
	return &example, nil
}

func (s *ExampleService) UpdateExample(ctx context.Context, exampleID, userID int32, title, description string) (*db.Example, error) {
	example, err := s.queries.UpdateExample(ctx, db.UpdateExampleParams{
		ID:          exampleID,
		UserID:      userID,
		Title:       title,
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
	if err != nil {
		return nil, notFoundOr(err)
	}

	s.invalidateList(ctx, userID)
	return &example, nil
}

func (s *ExampleService) DeleteExample(ctx context.Context, exampleID, userID int32) error {
	if _, err := s.GetExample(ctx, exampleID, userID); err != nil {
		return err
	}

	if err := s.queries.DeleteExample(ctx, db.DeleteExampleParams{ID: exampleID, UserID: userID}); err != nil {
		return errs.WrapDatabaseError(err)
	}

	s.invalidateList(ctx, userID)
	return nil
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
