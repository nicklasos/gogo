package middleware

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

var (
	ErrInvalidPageParameter = errors.New("invalid page parameter")
	ErrInvalidPageSize      = errors.New("invalid page_size parameter")
)

// PaginationParams holds parsed page-based pagination parameters
type PaginationParams struct {
	Page     int32
	PageSize int32
}

// Offset is the number of rows to skip for this page.
func (p PaginationParams) Offset() int32 {
	return (p.Page - 1) * p.PageSize
}

// GetPaginationParamsFromContext parses page/page_size query parameters.
func GetPaginationParamsFromContext(c *gin.Context, defaultPageSize, minPageSize, maxPageSize int32) (PaginationParams, error) {
	params := PaginationParams{Page: 1, PageSize: defaultPageSize}

	if pageStr := c.Query("page"); pageStr != "" {
		page, err := strconv.ParseInt(pageStr, 10, 32)
		if err != nil || page < 1 {
			return params, ErrInvalidPageParameter
		}
		params.Page = int32(page)
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		pageSize, err := strconv.ParseInt(pageSizeStr, 10, 32)
		if err != nil || pageSize < int64(minPageSize) || pageSize > int64(maxPageSize) {
			return params, fmt.Errorf("%w (must be between %d and %d)", ErrInvalidPageSize, minPageSize, maxPageSize)
		}
		params.PageSize = int32(pageSize)
	}

	return params, nil
}
