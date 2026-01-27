package integration

import (
	"app/internal/auth"
	"app/internal/example"
	"app/internal/db"
	"app/tests/helpers"
	"context"
	"net/http"
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getAuthToken(t *testing.T, server *helpers.TestServer) string {
	// Register and login to get token
	registerReq := `{
		"email": "test@example.com",
		"name": "Test User",
		"password": "password123"
	}`

	regResp := server.POST("/api/v1/auth/register", registerReq)
	require.Equal(t, http.StatusOK, regResp.StatusCode)

	var registerResponse auth.RegisterDataResponse
	err := regResp.JSON(&registerResponse)
	require.NoError(t, err)

	return registerResponse.Data.AccessToken
}

func getUserIDFromToken(t *testing.T, ctx context.Context, tx pgx.Tx, email string) int32 {
	var userID int32
	err := tx.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	require.NoError(t, err, "Failed to get user ID from email")
	return userID
}

func TestExampleAPI_CreateExample(t *testing.T) {
	t.Run("should return 200 when example is created successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			token := getAuthToken(t, server)

			// Test: Create example
			reqBody := `{
				"title": "Test Example",
				"description": "Test Description"
			}`

			reqBodyReader := helpers.StringToReadCloser(reqBody)
			req := server.NewRequest("POST", "/api/v1/examples", reqBodyReader)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			resp := server.Do(req)

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response example.ExampleDataResponse
			err := resp.JSON(&response)
			require.NoError(t, err)

			assert.NotNil(t, response.Data)
			assert.Equal(t, "Test Example", response.Data.Title)
			assert.Equal(t, "Test Description", response.Data.Description)
			assert.True(t, response.Data.ID > 0)
		})
	})

	t.Run("should return 401 when not authenticated", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Test: Create example without token
			reqBody := `{
				"title": "Test Example",
				"description": "Test Description"
			}`

			resp := server.POST("/api/v1/examples", reqBody)

			// Assert: Check response status
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	})

	t.Run("should return 400 when title is missing", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			token := getAuthToken(t, server)

			// Test: Create example without title
			reqBody := `{
				"description": "Test Description"
			}`

			reqBodyReader := helpers.StringToReadCloser(reqBody)
			req := server.NewRequest("POST", "/api/v1/examples", reqBodyReader)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			resp := server.Do(req)

			// Assert: Check response status
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	})
}

func TestExampleAPI_GetExample(t *testing.T) {
	t.Run("should return 200 when example is found", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server and example
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			token := getAuthToken(t, server)
			userID := getUserIDFromToken(t, ctx, tx, "test@example.com")
			testExample := helpers.CreateTestExample(t, ctx, tx, userID)

			// Test: Get example
			req := server.NewRequest("GET", "/api/v1/examples/"+strconv.Itoa(int(testExample.ID)), nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp := server.Do(req)

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response example.ExampleDataResponse
			err := resp.JSON(&response)
			require.NoError(t, err)

			assert.NotNil(t, response.Data)
			assert.Equal(t, testExample.ID, response.Data.ID)
			assert.Equal(t, testExample.Title, response.Data.Title)
		})
	})

	t.Run("should return 404 when example not found", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			token := getAuthToken(t, server)

			// Test: Get non-existent example
			req := server.NewRequest("GET", "/api/v1/examples/99999", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp := server.Do(req)

			// Assert: Check response status
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	})
}

func TestExampleAPI_ListExamples(t *testing.T) {
	t.Run("should return 200 with paginated examples", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server and examples
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			token := getAuthToken(t, server)
			userID := getUserIDFromToken(t, ctx, tx, "test@example.com")
			
			// Create multiple examples
			for i := 0; i < 5; i++ {
				helpers.CreateTestExample(t, ctx, tx, userID)
			}

			// Test: List examples
			req := server.NewRequest("GET", "/api/v1/examples?page=1&page_size=3", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp := server.Do(req)

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response example.PaginatedExamplesResponse
			err := resp.JSON(&response)
			require.NoError(t, err)

			assert.Equal(t, 3, len(response.Data))
			assert.Equal(t, int64(5), response.Pagination.Total)
			assert.Equal(t, int32(1), response.Pagination.CurrentPage)
			assert.Equal(t, int32(3), response.Pagination.PerPage)
		})
	})

	t.Run("should return 200 with empty list when no examples", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			token := getAuthToken(t, server)

			// Test: List examples
			req := server.NewRequest("GET", "/api/v1/examples", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp := server.Do(req)

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response example.PaginatedExamplesResponse
			err := resp.JSON(&response)
			require.NoError(t, err)

			assert.Equal(t, 0, len(response.Data))
			assert.Equal(t, int64(0), response.Pagination.Total)
		})
	})
}

func TestExampleAPI_UpdateExample(t *testing.T) {
	t.Run("should return 200 when example is updated successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server and example
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			token := getAuthToken(t, server)
			userID := getUserIDFromToken(t, ctx, tx, "test@example.com")
			testExample := helpers.CreateTestExample(t, ctx, tx, userID)

			// Test: Update example
			reqBody := `{
				"title": "Updated Title",
				"description": "Updated Description"
			}`

			reqBodyReader := helpers.StringToReadCloser(reqBody)
			req := server.NewRequest("PUT", "/api/v1/examples/"+strconv.Itoa(int(testExample.ID)), reqBodyReader)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			resp := server.Do(req)

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response example.ExampleDataResponse
			err := resp.JSON(&response)
			require.NoError(t, err)

			assert.Equal(t, "Updated Title", response.Data.Title)
			assert.Equal(t, "Updated Description", response.Data.Description)
		})
	})

	t.Run("should return 404 when example not found", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			token := getAuthToken(t, server)

			// Test: Update non-existent example
			reqBody := `{
				"title": "Updated Title",
				"description": "Updated Description"
			}`

			reqBodyReader := helpers.StringToReadCloser(reqBody)
			req := server.NewRequest("PUT", "/api/v1/examples/99999", reqBodyReader)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			resp := server.Do(req)

			// Assert: Check response status
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	})
}

func TestExampleAPI_DeleteExample(t *testing.T) {
	t.Run("should return 200 when example is deleted successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server and example
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			token := getAuthToken(t, server)
			userID := getUserIDFromToken(t, ctx, tx, "test@example.com")
			testExample := helpers.CreateTestExample(t, ctx, tx, userID)

			// Test: Delete example
			req := server.NewRequest("DELETE", "/api/v1/examples/"+strconv.Itoa(int(testExample.ID)), nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp := server.Do(req)

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Verify example is deleted
			req2 := server.NewRequest("GET", "/api/v1/examples/"+strconv.Itoa(int(testExample.ID)), nil)
			req2.Header.Set("Authorization", "Bearer "+token)
			resp2 := server.Do(req2)
			assert.Equal(t, http.StatusNotFound, resp2.StatusCode)
		})
	})

	t.Run("should return 404 when example not found", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			token := getAuthToken(t, server)

			// Test: Delete non-existent example
			req := server.NewRequest("DELETE", "/api/v1/examples/99999", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp := server.Do(req)

			// Assert: Check response status
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	})
}
