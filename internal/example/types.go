package example

import (
	"app/internal"
)

// CreateExampleRequest represents the request to create an example
type CreateExampleRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

// UpdateExampleRequest represents the request to update an example
type UpdateExampleRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

// ExampleResponse represents example information
type ExampleResponse struct {
	ID          int32  `json:"id"`
	UserID      int32  `json:"user_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ExampleDataResponse wraps example data in response
type ExampleDataResponse struct {
	Data *ExampleResponse `json:"data"`
}

// PaginatedExamplesResponse wraps paginated examples in response
type PaginatedExamplesResponse struct {
	Data       []ExampleResponse      `json:"data"`
	Pagination internal.PaginationMeta `json:"pagination"`
}

// ExamplesListResponse wraps examples list in response
type ExamplesListResponse struct {
	Data []ExampleResponse `json:"data"`
}

// MessageResponse wraps a simple message in response
type MessageResponse struct {
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
}

// ErrorResponse represents error response structure
type ErrorResponse struct {
	Error string `json:"error"`
}
