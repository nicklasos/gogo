package middleware

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"app/internal/errs"

	"github.com/gin-gonic/gin"
)

var (
	ErrInvalidPageParameter = errors.New("invalid page parameter")
	ErrInvalidPageSize      = errors.New("invalid page_size parameter")
	ErrInvalidLastID        = errors.New("invalid last_id parameter")
	ErrInvalidLimit         = errors.New("invalid limit parameter")
	ErrInvalidCreatedAt     = errors.New("Invalid last_created_at format, use RFC3339 or ISO 8601")
)

var iso8601Formats = []string{
	"2006-01-02T15:04:05.000",
	"2006-01-02T15:04:05",
}

func ParseLastCreatedAt(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	for _, layout := range iso8601Formats {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, ErrInvalidCreatedAt
}

// PaginationParams holds parsed page-based pagination parameters
type PaginationParams struct {
	Page     int32
	PageSize int32
}

// LastIDPaginationParams holds cursor-based pagination using last_id
type LastIDPaginationParams struct {
	LastID *int32
	Limit  int32
}

// CreatedAtPaginationParams holds cursor-based pagination using created_at
type CreatedAtPaginationParams struct {
	LastCreatedAt *time.Time
	Limit         int32
}

// GetPaginationParamsFromContext parses page/page_size query parameters.
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

// GetLastIDPaginationParamsFromContext parses last_id/limit cursor pagination.
func GetLastIDPaginationParamsFromContext(c *gin.Context, defaultLimit, minLimit, maxLimit int32) (LastIDPaginationParams, error) {
	var params LastIDPaginationParams

	if lastIDStr := c.Query("last_id"); lastIDStr != "" {
		lastID, err := strconv.ParseInt(lastIDStr, 10, 32)
		if err != nil || lastID <= 0 {
			return LastIDPaginationParams{}, ErrInvalidLastID
		}
		lastIDInt32 := int32(lastID)
		params.LastID = &lastIDInt32
	}

	if limitStr := c.Query("limit"); limitStr == "" {
		params.Limit = defaultLimit
	} else {
		limit, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil {
			return LastIDPaginationParams{}, ErrInvalidLimit
		}
		limitInt32 := int32(limit)
		if limitInt32 < minLimit || limitInt32 > maxLimit {
			return LastIDPaginationParams{}, ErrInvalidLimit
		}
		params.Limit = limitInt32
	}

	return params, nil
}

// GetCreatedAtPaginationParamsFromContext parses last_created_at/limit cursor pagination.
func GetCreatedAtPaginationParamsFromContext(c *gin.Context, defaultLimit, minLimit, maxLimit int32) (CreatedAtPaginationParams, error) {
	var params CreatedAtPaginationParams

	if lastCreatedAtStr := c.Query("last_created_at"); lastCreatedAtStr != "" {
		lastCreatedAt, err := ParseLastCreatedAt(lastCreatedAtStr)
		if err != nil {
			return CreatedAtPaginationParams{}, errs.NewBadRequestError(errs.ErrKeyInvalidFormat, "Invalid last_created_at format, use RFC3339 or ISO 8601")
		}
		params.LastCreatedAt = &lastCreatedAt
	}

	if limitStr := c.Query("limit"); limitStr == "" {
		params.Limit = defaultLimit
	} else {
		limit, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil {
			return CreatedAtPaginationParams{}, errs.NewBadRequestError(errs.ErrKeyBadRequest, "invalid limit parameter")
		}
		limitInt32 := int32(limit)
		if limitInt32 < minLimit || limitInt32 > maxLimit {
			return CreatedAtPaginationParams{}, errs.NewBadRequestError(errs.ErrKeyBadRequest, "invalid limit parameter")
		}
		params.Limit = limitInt32
	}

	return params, nil
}
