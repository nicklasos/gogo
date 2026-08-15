package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLastIDPaginationParamsFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("defaults when empty", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

		params, err := middleware.GetLastIDPaginationParamsFromContext(c, 20, 1, 100)
		require.NoError(t, err)
		assert.Nil(t, params.LastID)
		assert.Equal(t, int32(20), params.Limit)
	})

	t.Run("parses last_id and limit", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/?last_id=42&limit=10", nil)

		params, err := middleware.GetLastIDPaginationParamsFromContext(c, 20, 1, 100)
		require.NoError(t, err)
		require.NotNil(t, params.LastID)
		assert.Equal(t, int32(42), *params.LastID)
		assert.Equal(t, int32(10), params.Limit)
	})

	t.Run("rejects invalid last_id", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/?last_id=0", nil)

		_, err := middleware.GetLastIDPaginationParamsFromContext(c, 20, 1, 100)
		assert.ErrorIs(t, err, middleware.ErrInvalidLastID)
	})
}

func TestGetCreatedAtPaginationParamsFromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("parses RFC3339 last_created_at", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		ts := "2024-01-15T12:00:00Z"
		c.Request = httptest.NewRequest(http.MethodGet, "/?last_created_at="+ts+"&limit=5", nil)

		params, err := middleware.GetCreatedAtPaginationParamsFromContext(c, 20, 1, 100)
		require.NoError(t, err)
		require.NotNil(t, params.LastCreatedAt)
		assert.True(t, params.LastCreatedAt.Equal(time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)))
		assert.Equal(t, int32(5), params.Limit)
	})

	t.Run("rejects invalid last_created_at", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/?last_created_at=not-a-date", nil)

		_, err := middleware.GetCreatedAtPaginationParamsFromContext(c, 20, 1, 100)
		assert.Error(t, err)
	})
}

func TestParseLastCreatedAt(t *testing.T) {
	t.Run("ISO8601 without timezone treated as UTC", func(t *testing.T) {
		got, err := middleware.ParseLastCreatedAt("2024-06-01T10:30:00")
		require.NoError(t, err)
		assert.True(t, got.Equal(time.Date(2024, 6, 1, 10, 30, 0, 0, time.UTC)))
	})
}
