package integration

import (
	"app/internal/db"
	"app/internal/uploads"
	"app/tests/helpers"
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadAPI_UploadFile(t *testing.T) {
	t.Run("should return 200 when file is uploaded successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			token := getAuthToken(t, server)
			userID := getUserIDFromToken(t, ctx, tx, "test@example.com")

			// Create a test file
			fileContent := []byte("test image content")
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", "test.jpg")
			require.NoError(t, err)
			_, err = part.Write(fileContent)
			require.NoError(t, err)
			err = writer.Close()
			require.NoError(t, err)

			// Test: Upload file
			req := server.NewRequest("POST", "/api/v1/uploads", body)
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			resp := server.Do(req)

			// Assert: Check response status
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Assert: Parse and validate response body
			var response uploads.UploadDataResponse
			err = resp.JSON(&response)
			require.NoError(t, err)

			assert.NotNil(t, response.Data)
			assert.True(t, response.Data.ID > 0)
			assert.Equal(t, userID, response.Data.UserID)
			assert.Equal(t, userID, response.Data.FolderID)
			assert.Equal(t, "image", response.Data.Type)
			assert.Equal(t, "test.jpg", response.Data.OriginalFilename)
			assert.Equal(t, int64(len(fileContent)), response.Data.FileSize)
			assert.NotEmpty(t, response.Data.RelativePath)
			assert.NotEmpty(t, response.Data.FullURL)
		})
	})

	t.Run("should return 401 when not authenticated", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			// Create a test file
			fileContent := []byte("test image content")
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", "test.jpg")
			require.NoError(t, err)
			_, err = part.Write(fileContent)
			require.NoError(t, err)
			err = writer.Close()
			require.NoError(t, err)

			// Test: Upload file without token
			req := server.NewRequest("POST", "/api/v1/uploads", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			resp := server.Do(req)

			// Assert: Check response status
			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		})
	})

	t.Run("should return 400 when no file uploaded", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create test server
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			token := getAuthToken(t, server)

			// Test: Upload without file
			req := server.NewRequest("POST", "/api/v1/uploads", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp := server.Do(req)

			// Assert: Check response status
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	})
}

// Note: GetUpload, ListUploads, and DeleteUpload are service methods only
// They are not exposed as HTTP endpoints but can be used internally by other services
