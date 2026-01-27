package internal

import "math"

// ErrorResponse represents a simple error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// PaginationMeta contains pagination metadata
type PaginationMeta struct {
	Total       int64 `json:"total"`
	CurrentPage int32 `json:"current_page"`
	LastPage    int32 `json:"last_page"`
	PerPage     int32 `json:"per_page"`
}

// NewPaginationMeta creates pagination metadata from pagination parameters
func NewPaginationMeta(total int64, page, pageSize int32) PaginationMeta {
	lastPage := int32(math.Ceil(float64(total) / float64(pageSize)))
	if lastPage == 0 {
		lastPage = 1
	}

	return PaginationMeta{
		Total:       total,
		CurrentPage: page,
		LastPage:    lastPage,
		PerPage:     pageSize,
	}
}
