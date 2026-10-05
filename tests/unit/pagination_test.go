package unit

import (
	"net/http/httptest"
	"testing"

	"app/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func paginationContext(query string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/items"+query, nil)
	return c
}

func TestGetPaginationParamsFromContext(t *testing.T) {
	t.Run("defaults when empty", func(t *testing.T) {
		params, err := middleware.GetPaginationParamsFromContext(paginationContext(""), 20, 1, 100)
		require.NoError(t, err)
		assert.Equal(t, middleware.PaginationParams{Page: 1, PageSize: 20}, params)
		assert.Equal(t, int32(0), params.Offset())
	})

	t.Run("parses page and page_size", func(t *testing.T) {
		params, err := middleware.GetPaginationParamsFromContext(paginationContext("?page=3&page_size=50"), 20, 1, 100)
		require.NoError(t, err)
		assert.Equal(t, middleware.PaginationParams{Page: 3, PageSize: 50}, params)
		assert.Equal(t, int32(100), params.Offset())
	})

	t.Run("rejects an invalid page", func(t *testing.T) {
		for _, query := range []string{"?page=0", "?page=-1", "?page=abc"} {
			_, err := middleware.GetPaginationParamsFromContext(paginationContext(query), 20, 1, 100)
			assert.ErrorIs(t, err, middleware.ErrInvalidPageParameter, query)
		}
	})

	t.Run("rejects a page size outside the limits", func(t *testing.T) {
		for _, query := range []string{"?page_size=0", "?page_size=101", "?page_size=abc"} {
			_, err := middleware.GetPaginationParamsFromContext(paginationContext(query), 20, 1, 100)
			assert.ErrorIs(t, err, middleware.ErrInvalidPageSize, query)
		}
	})
}
