package example

import (
	"app/internal/db"
	"app/internal/errs"
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// Error variables - define all service errors at the top of the file
// Use error keys from errs package and descriptive messages
var (
	ErrExampleNotFound = errs.NewNotFoundError(errs.ErrKeyExampleNotFound, "Example not found")
	ErrInvalidPage     = errs.NewBadRequestError(errs.ErrKeyBadRequest, "Invalid page parameter")
	ErrInvalidPageSize = errs.NewBadRequestError(errs.ErrKeyBadRequest, "Invalid page size parameter")
)

// PaginatedExamplesResult represents paginated example results from service layer
type PaginatedExamplesResult struct {
	Data     []db.Example
	Total    int64
	Page     int32
	PageSize int32
}

// ExampleService contains business logic for example operations
type ExampleService struct {
	queries *db.Queries
}

// NewExampleService creates a new example service
func NewExampleService(queries *db.Queries) *ExampleService {
	return &ExampleService{
		queries: queries,
	}
}

// CreateExample creates a new example
// Error handling example: Wrap database errors as internal errors
func (s *ExampleService) CreateExample(ctx context.Context, userID int32, title, description string) (*db.Example, error) {
	example, err := s.queries.CreateExample(ctx, db.CreateExampleParams{
		UserID:      userID,
		Title:       title,
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
	if err != nil {
		// Wrap database errors - preserves error chain for logging
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to create example", err)
	}

	return &example, nil
}

// GetExample retrieves an example by ID for a specific user
// Error handling example: Return domain error directly for business logic errors
func (s *ExampleService) GetExample(ctx context.Context, exampleID, userID int32) (*db.Example, error) {
	example, err := s.queries.GetExampleByID(ctx, db.GetExampleByIDParams{
		ID:     exampleID,
		UserID: userID,
	})
	if err != nil {
		// Return domain error - handler will format it automatically
		return nil, ErrExampleNotFound
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
		return nil, ErrExampleNotFound
	}

	return &example, nil
}

// DeleteExample deletes an example
func (s *ExampleService) DeleteExample(ctx context.Context, exampleID, userID int32) error {
	// First check if example exists
	_, err := s.queries.GetExampleByID(ctx, db.GetExampleByIDParams{
		ID:     exampleID,
		UserID: userID,
	})
	if err != nil {
		return ErrExampleNotFound
	}

	// Delete the example
	err = s.queries.DeleteExample(ctx, db.DeleteExampleParams{
		ID:     exampleID,
		UserID: userID,
	})
	if err != nil {
		return errs.WrapInternal(errs.ErrKeyInternalError, "failed to delete example", err)
	}

	return nil
}

// ListExamples retrieves all examples for a user
func (s *ExampleService) ListExamples(ctx context.Context, userID int32) ([]db.Example, error) {
	examples, err := s.queries.ListExamplesForUser(ctx, userID)
	if err != nil {
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to list examples", err)
	}

	if examples == nil {
		return []db.Example{}, nil
	}

	return examples, nil
}

// ListExamplesPaginated retrieves paginated examples for a user
// Error handling example: Validate input and return domain errors
func (s *ExampleService) ListExamplesPaginated(ctx context.Context, userID, page, pageSize int32) (*PaginatedExamplesResult, error) {
	// Input validation - return domain errors for invalid input
	if page < 1 {
		return nil, ErrInvalidPage
	}
	if pageSize < 1 || pageSize > 100 {
		return nil, ErrInvalidPageSize
	}

	offset := (page - 1) * pageSize

	examples, err := s.queries.ListExamplesForUserPaginated(ctx, db.ListExamplesForUserPaginatedParams{
		UserID: userID,
		Limit:  pageSize,
		Offset: offset,
	})
	if err != nil {
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to list examples", err)
	}

	total, err := s.queries.CountExamplesForUser(ctx, userID)
	if err != nil {
		return nil, errs.WrapInternal(errs.ErrKeyInternalError, "failed to count examples", err)
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
