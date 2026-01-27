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

// PaginationParams holds parsed pagination parameters
type PaginationParams struct {
	Page     int32
	PageSize int32
}

// GetPaginationParamsFromContext parses pagination parameters from the gin context query parameters
// It validates page (must be >= 1) and pageSize (must be between minPageSize and maxPageSize)
// Returns parsed params on success, or an error if validation fails
func GetPaginationParamsFromContext(c *gin.Context, defaultPageSize, minPageSize, maxPageSize int32) (PaginationParams, error) {
	var params PaginationParams

	page := int32(1)
	if pageStr := c.Query("page"); pageStr != "" {
		pageInt, err := strconv.ParseInt(pageStr, 10, 32)
		if err != nil || pageInt < 1 {
			return params, ErrInvalidPageParameter
		}
		page = int32(pageInt)
	}

	pageSize := defaultPageSize
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		pageSizeInt, err := strconv.ParseInt(pageSizeStr, 10, 32)
		if err != nil || pageSizeInt < int64(minPageSize) || pageSizeInt > int64(maxPageSize) {
			return params, fmt.Errorf("%w (must be between %d and %d)", ErrInvalidPageSize, minPageSize, maxPageSize)
		}
		pageSize = int32(pageSizeInt)
	}

	params.Page = page
	params.PageSize = pageSize
	return params, nil
}
