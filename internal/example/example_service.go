package example

import (
	"app/internal/cache"
	"app/internal/db"
	"app/internal/errs"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrExampleNotFound = errs.NewNotFoundError(errs.ErrKeyExampleNotFound, "Example not found")
	ErrInvalidPage     = errs.NewBadRequestError(errs.ErrKeyBadRequest, "Invalid page parameter")
	ErrInvalidPageSize = errs.NewBadRequestError(errs.ErrKeyBadRequest, "Invalid page size parameter")
)

// PaginatedExamplesResult represents paginated example results from service layer
type PaginatedExamplesResult struct {
	Data     []db.Example `json:"data"`
	Total    int64        `json:"total"`
	Page     int32        `json:"page"`
	PageSize int32        `json:"page_size"`
}

// ExampleService contains business logic for example operations
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

func (s *ExampleService) examplesListCacheKey(userID, page, pageSize int32) string {
	return fmt.Sprintf("examples:user:%d:page:%d:size:%d", userID, page, pageSize)
}

func (s *ExampleService) invalidateUserExamplesCache(ctx context.Context, userID int32) {
	if s.cache == nil {
		return
	}
	for page := int32(1); page <= 5; page++ {
		for _, size := range []int32{10, 20, 50, 100} {
			_ = s.cache.Forget(ctx, s.examplesListCacheKey(userID, page, size))
		}
	}
}

// CreateExample creates a new example
func (s *ExampleService) CreateExample(ctx context.Context, userID int32, title, description string) (*db.Example, error) {
	example, err := s.queries.CreateExample(ctx, db.CreateExampleParams{
		UserID:      userID,
		Title:       title,
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
	if err != nil {
		return nil, errs.WrapDatabaseError(err)
	}

	s.invalidateUserExamplesCache(ctx, userID)
	return &example, nil
}

// GetExample retrieves an example by ID for a specific user
func (s *ExampleService) GetExample(ctx context.Context, exampleID, userID int32) (*db.Example, error) {
	example, err := s.queries.GetExampleByID(ctx, db.GetExampleByIDParams{
		ID:     exampleID,
		UserID: userID,
	})
	if err != nil {
		if wrapped := errs.WrapDatabaseError(err); errs.IsNotFound(wrapped) {
			return nil, ErrExampleNotFound
		}
		return nil, errs.WrapDatabaseError(err)
	}

	return &example, nil
}

// UpdateExample updates an existing example
func (s *ExampleService) UpdateExample(ctx context.Context, exampleID, userID int32, title, description string) (*db.Example, error) {
	example, err := s.queries.UpdateExample(ctx, db.UpdateExampleParams{
		ID:          exampleID,
		UserID:      userID,
		Title:       title,
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
	if err != nil {
		if wrapped := errs.WrapDatabaseError(err); errs.IsNotFound(wrapped) {
			return nil, ErrExampleNotFound
		}
		return nil, errs.WrapDatabaseError(err)
	}

	s.invalidateUserExamplesCache(ctx, userID)
	return &example, nil
}

// DeleteExample deletes an example
func (s *ExampleService) DeleteExample(ctx context.Context, exampleID, userID int32) error {
	_, err := s.queries.GetExampleByID(ctx, db.GetExampleByIDParams{
		ID:     exampleID,
		UserID: userID,
	})
	if err != nil {
		if wrapped := errs.WrapDatabaseError(err); errs.IsNotFound(wrapped) {
			return ErrExampleNotFound
		}
		return errs.WrapDatabaseError(err)
	}

	err = s.queries.DeleteExample(ctx, db.DeleteExampleParams{
		ID:     exampleID,
		UserID: userID,
	})
	if err != nil {
		return errs.WrapDatabaseError(err)
	}

	s.invalidateUserExamplesCache(ctx, userID)
	return nil
}

// ListExamples retrieves all examples for a user
func (s *ExampleService) ListExamples(ctx context.Context, userID int32) ([]db.Example, error) {
	examples, err := s.queries.ListExamplesForUser(ctx, userID)
	if err != nil {
		return nil, errs.WrapDatabaseError(err)
	}

	if examples == nil {
		return []db.Example{}, nil
	}

	return examples, nil
}

// ListExamplesPaginated retrieves paginated examples for a user (with short-lived cache).
func (s *ExampleService) ListExamplesPaginated(ctx context.Context, userID, page, pageSize int32) (*PaginatedExamplesResult, error) {
	if page < 1 {
		return nil, ErrInvalidPage
	}
	if pageSize < 1 || pageSize > 100 {
		return nil, ErrInvalidPageSize
	}

	load := func() (*PaginatedExamplesResult, error) {
		offset := (page - 1) * pageSize

		examples, err := s.queries.ListExamplesForUserPaginated(ctx, db.ListExamplesForUserPaginatedParams{
			UserID: userID,
			Limit:  pageSize,
			Offset: offset,
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

		return &PaginatedExamplesResult{
			Data:     examples,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	if s.cache == nil {
		return load()
	}

	var result PaginatedExamplesResult
	err := s.cache.Remember(ctx, s.examplesListCacheKey(userID, page, pageSize), 30*time.Second, func() (interface{}, error) {
		return load()
	}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
