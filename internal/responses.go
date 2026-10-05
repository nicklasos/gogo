package internal

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

const timeFormat = "2006-01-02T15:04:05Z07:00"

// FormatTime renders a database timestamp the way every API response does.
func FormatTime(t pgtype.Timestamp) string {
	return t.Time.Format(timeFormat)
}

// MessageResponse is the body of an action that has nothing else to return
type MessageResponse struct {
	Data MessageData `json:"data"`
}

type MessageData struct {
	Message string `json:"message"`
}

// RespondMessage answers 200 with a MessageResponse.
func RespondMessage(c *gin.Context, message string) {
	c.JSON(http.StatusOK, MessageResponse{Data: MessageData{Message: message}})
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
