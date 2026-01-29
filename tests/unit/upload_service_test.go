package unit

import (
	"bytes"
	"context"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"app/internal/db"
	"app/internal/uploads"
	"app/tests/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadService_UploadFile(t *testing.T) {
	t.Run("should upload file successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and temp directory
			user := helpers.CreateTestUser(t, ctx, tx)
			tempDir := t.TempDir()
			config := uploads.DefaultUploadConfig(tempDir, "http://localhost:8181/api/files")
			service := uploads.NewUploadService(queries, config)

			// Create a test file
			fileContent := []byte("test file content")
			fileHeader := createTestFileHeader(t, "test.jpg", fileContent, "image/jpeg")

			// Test: Upload file
			upload, err := service.UploadFile(ctx, fileHeader, user.ID)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, upload)
			assert.Equal(t, user.ID, upload.UserID)
			assert.Equal(t, user.ID, upload.FolderID)
			assert.Equal(t, "image", upload.Type)
			assert.Equal(t, "test.jpg", upload.OriginalFilename)
			assert.Equal(t, int64(len(fileContent)), upload.FileSize)
			assert.True(t, upload.ID > 0)

			// Verify file exists on disk
			fullPath := filepath.Join(tempDir, upload.RelativePath)
			_, err = os.Stat(fullPath)
			assert.NoError(t, err)
		})
	})

	t.Run("should return error for invalid file type", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and temp directory
			user := helpers.CreateTestUser(t, ctx, tx)
			tempDir := t.TempDir()
			config := uploads.DefaultUploadConfig(tempDir, "http://localhost:8181/api/files")
			service := uploads.NewUploadService(queries, config)

			// Create a test file with invalid extension
			fileContent := []byte("test file content")
			fileHeader := createTestFileHeader(t, "test.exe", fileContent, "application/x-msdownload")

			// Test: Upload file
			upload, err := service.UploadFile(ctx, fileHeader, user.ID)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Nil(t, upload)
		})
	})

	t.Run("should return error for file too large", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and temp directory
			user := helpers.CreateTestUser(t, ctx, tx)
			tempDir := t.TempDir()
			config := uploads.DefaultUploadConfig(tempDir, "http://localhost:8181/api/files")
			config.MaxFileSize = 10 // Very small limit
			service := uploads.NewUploadService(queries, config)

			// Create a test file that's too large
			fileContent := make([]byte, 100)
			fileHeader := createTestFileHeader(t, "test.jpg", fileContent, "image/jpeg")

			// Test: Upload file
			upload, err := service.UploadFile(ctx, fileHeader, user.ID)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Nil(t, upload)
		})
	})

	t.Run("should return error for empty file", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and temp directory
			user := helpers.CreateTestUser(t, ctx, tx)
			tempDir := t.TempDir()
			config := uploads.DefaultUploadConfig(tempDir, "http://localhost:8181/api/files")
			service := uploads.NewUploadService(queries, config)

			// Create an empty test file
			fileContent := []byte{}
			fileHeader := createTestFileHeader(t, "test.jpg", fileContent, "image/jpeg")

			// Test: Upload file
			upload, err := service.UploadFile(ctx, fileHeader, user.ID)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Nil(t, upload)
		})
	})
}

func TestUploadService_GetUpload(t *testing.T) {
	t.Run("should get upload successfully", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and upload
			user := helpers.CreateTestUser(t, ctx, tx)
			testUpload := helpers.CreateTestUpload(t, ctx, tx, user.ID)

			tempDir := t.TempDir()
			config := uploads.DefaultUploadConfig(tempDir, "http://localhost:8181/api/files")
			service := uploads.NewUploadService(queries, config)

			// Test: Get upload
			upload, err := service.GetUpload(ctx, testUpload.ID, user.ID)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, upload)
			assert.Equal(t, testUpload.ID, upload.ID)
			assert.Equal(t, user.ID, upload.UserID)
		})
	})

	t.Run("should return error when upload not found", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user
			user := helpers.CreateTestUser(t, ctx, tx)
			tempDir := t.TempDir()
			config := uploads.DefaultUploadConfig(tempDir, "http://localhost:8181/api/files")
			service := uploads.NewUploadService(queries, config)

			// Test: Get non-existent upload
			upload, err := service.GetUpload(ctx, 99999, user.ID)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Equal(t, uploads.ErrUploadNotFound, err)
			assert.Nil(t, upload)
		})
	})
}

func TestUploadService_ListUploads(t *testing.T) {
	t.Run("should list uploads for user", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and multiple uploads
			user := helpers.CreateTestUser(t, ctx, tx)
			upload1 := helpers.CreateTestUpload(t, ctx, tx, user.ID)
			upload2 := helpers.CreateTestUpload(t, ctx, tx, user.ID)

			tempDir := t.TempDir()
			config := uploads.DefaultUploadConfig(tempDir, "http://localhost:8181/api/files")
			service := uploads.NewUploadService(queries, config)

			// Test: List uploads
			uploads, err := service.ListUploads(ctx, user.ID)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, uploads)
			assert.GreaterOrEqual(t, len(uploads), 2)

			// Verify uploads are in the list
			uploadIDs := make(map[int32]bool)
			for _, u := range uploads {
				uploadIDs[u.ID] = true
			}
			assert.True(t, uploadIDs[upload1.ID])
			assert.True(t, uploadIDs[upload2.ID])
		})
	})

	t.Run("should return empty list when user has no uploads", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user
			user := helpers.CreateTestUser(t, ctx, tx)
			tempDir := t.TempDir()
			config := uploads.DefaultUploadConfig(tempDir, "http://localhost:8181/api/files")
			service := uploads.NewUploadService(queries, config)

			// Test: List uploads
			uploads, err := service.ListUploads(ctx, user.ID)

			// Assert: Verify result
			require.NoError(t, err)
			assert.NotNil(t, uploads)
			assert.Equal(t, 0, len(uploads))
		})
	})
}

func TestUploadService_DeleteUpload(t *testing.T) {
	t.Run("should delete upload successfully and remove file from disk", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user and temp directory
			user := helpers.CreateTestUser(t, ctx, tx)
			tempDir := t.TempDir()
			config := uploads.DefaultUploadConfig(tempDir, "http://localhost:8181/api/files")
			service := uploads.NewUploadService(queries, config)

			// Create a test file and upload it
			fileContent := []byte("test file content")
			fileHeader := createTestFileHeader(t, "test.jpg", fileContent, "image/jpeg")
			upload, err := service.UploadFile(ctx, fileHeader, user.ID)
			require.NoError(t, err)

			// Verify file exists on disk
			filePath := filepath.Join(tempDir, upload.RelativePath)
			_, err = os.Stat(filePath)
			require.NoError(t, err, "File should exist before deletion")

			// Test: Delete upload
			err = service.DeleteUpload(ctx, upload.ID, user.ID)

			// Assert: Verify result
			require.NoError(t, err)

			// Verify upload is deleted from database
			_, err = service.GetUpload(ctx, upload.ID, user.ID)
			assert.Error(t, err)
			assert.Equal(t, uploads.ErrUploadNotFound, err)

			// Verify file is deleted from disk
			_, err = os.Stat(filePath)
			assert.Error(t, err, "File should be deleted from disk")
			assert.True(t, os.IsNotExist(err), "File should not exist after deletion")
		})
	})

	t.Run("should return error when upload not found", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			// Setup: Create a user
			user := helpers.CreateTestUser(t, ctx, tx)
			tempDir := t.TempDir()
			config := uploads.DefaultUploadConfig(tempDir, "http://localhost:8181/api/files")
			service := uploads.NewUploadService(queries, config)

			// Test: Delete non-existent upload
			err := service.DeleteUpload(ctx, 99999, user.ID)

			// Assert: Should return error
			assert.Error(t, err)
			assert.Equal(t, uploads.ErrUploadNotFound, err)
		})
	})
}

func TestUploadService_GetFileType(t *testing.T) {
	t.Run("should return correct file type for images", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			tempDir := t.TempDir()
			config := uploads.DefaultUploadConfig(tempDir, "http://localhost:8181/api/files")
			service := uploads.NewUploadService(queries, config)

			assert.Equal(t, "image", service.GetFileType("test.jpg"))
			assert.Equal(t, "image", service.GetFileType("test.png"))
			assert.Equal(t, "image", service.GetFileType("test.gif"))
		})
	})

	t.Run("should return correct file type for videos", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			tempDir := t.TempDir()
			config := uploads.DefaultUploadConfig(tempDir, "http://localhost:8181/api/files")
			service := uploads.NewUploadService(queries, config)

			assert.Equal(t, "video", service.GetFileType("test.mp4"))
			assert.Equal(t, "video", service.GetFileType("test.avi"))
		})
	})

	t.Run("should return correct file type for documents", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			tempDir := t.TempDir()
			config := uploads.DefaultUploadConfig(tempDir, "http://localhost:8181/api/files")
			service := uploads.NewUploadService(queries, config)

			assert.Equal(t, "document", service.GetFileType("test.pdf"))
			assert.Equal(t, "document", service.GetFileType("test.doc"))
		})
	})

	t.Run("should return other for unknown types", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			tempDir := t.TempDir()
			config := uploads.DefaultUploadConfig(tempDir, "http://localhost:8181/api/files")
			service := uploads.NewUploadService(queries, config)

			assert.Equal(t, "other", service.GetFileType("test.unknown"))
		})
	})
}

// Helper function to create a test file header
func createTestFileHeader(t *testing.T, filename string, content []byte, contentType string) *multipart.FileHeader {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)

	_, err = part.Write(content)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(32 << 20)
	require.NoError(t, err)

	fileHeader := form.File["file"][0]
	fileHeader.Header.Set("Content-Type", contentType)

	return fileHeader
}
