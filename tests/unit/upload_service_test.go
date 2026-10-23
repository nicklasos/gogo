package unit

import (
	"bytes"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"app/internal/uploads"
	"app/tests/testkit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func uploadConfig(app *testkit.App) *uploads.UploadConfig {
	return uploads.DefaultUploadConfig(app.Config.UploadFolder, app.Config.FilesBaseURL)
}

func newUploadService(app *testkit.App) *uploads.UploadService {
	return uploads.NewUploadService(app.Queries, uploadConfig(app))
}

func TestUploadService_UploadFile(t *testing.T) {
	t.Run("should upload file successfully", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		service := newUploadService(app)

		// Create a test file
		fileContent := testkit.JPEG
		fileHeader := createTestFileHeader(t, "test.jpg", fileContent, "image/jpeg")

		upload, err := service.UploadFile(app.Ctx, fileHeader, user.ID)

		require.NoError(t, err)
		assert.NotNil(t, upload)
		assert.Equal(t, user.ID, upload.UserID)
		assert.Equal(t, user.ID, upload.FolderID)
		assert.Equal(t, "image", upload.Type)
		assert.Equal(t, "test.jpg", upload.OriginalFilename)
		assert.Equal(t, int64(len(fileContent)), upload.FileSize)
		assert.True(t, upload.ID > 0)

		// Verify file exists on disk
		fullPath := filepath.Join(app.Config.UploadFolder, upload.RelativePath)
		_, err = os.Stat(fullPath)
		assert.NoError(t, err)
	})

	t.Run("should return error for invalid file type", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		service := newUploadService(app)

		// Create a test file with invalid extension
		fileContent := testkit.JPEG
		fileHeader := createTestFileHeader(t, "test.exe", fileContent, "application/x-msdownload")

		upload, err := service.UploadFile(app.Ctx, fileHeader, user.ID)

		assert.Error(t, err)
		assert.Nil(t, upload)
	})

	t.Run("should return error for file too large", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		config := uploadConfig(app)
		config.MaxFileSize = 10
		service := uploads.NewUploadService(app.Queries, config)

		// Create a test file that's too large
		fileContent := make([]byte, 100)
		fileHeader := createTestFileHeader(t, "test.jpg", fileContent, "image/jpeg")

		upload, err := service.UploadFile(app.Ctx, fileHeader, user.ID)

		assert.Error(t, err)
		assert.Nil(t, upload)
	})

	t.Run("should return error for empty file", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		service := newUploadService(app)

		// Create an empty test file
		fileContent := []byte{}
		fileHeader := createTestFileHeader(t, "test.jpg", fileContent, "image/jpeg")

		upload, err := service.UploadFile(app.Ctx, fileHeader, user.ID)

		assert.Error(t, err)
		assert.Nil(t, upload)
	})
}

func TestUploadService_GetUpload(t *testing.T) {
	t.Run("should get upload successfully", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		testUpload := app.Create.Upload(user.ID)

		service := newUploadService(app)

		upload, err := service.GetUpload(app.Ctx, testUpload.ID, user.ID)

		require.NoError(t, err)
		assert.NotNil(t, upload)
		assert.Equal(t, testUpload.ID, upload.ID)
		assert.Equal(t, user.ID, upload.UserID)
	})

	t.Run("should return error when upload not found", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		service := newUploadService(app)

		upload, err := service.GetUpload(app.Ctx, 99999, user.ID)

		assert.Error(t, err)
		assert.Equal(t, uploads.ErrUploadNotFound, err)
		assert.Nil(t, upload)
	})
}

func TestUploadService_DeleteUpload(t *testing.T) {
	t.Run("should delete upload successfully and remove file from disk", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		service := newUploadService(app)

		// Create a test file and upload it
		fileContent := testkit.JPEG
		fileHeader := createTestFileHeader(t, "test.jpg", fileContent, "image/jpeg")
		upload, err := service.UploadFile(app.Ctx, fileHeader, user.ID)
		require.NoError(t, err)

		// Verify file exists on disk
		filePath := filepath.Join(app.Config.UploadFolder, upload.RelativePath)
		_, err = os.Stat(filePath)
		require.NoError(t, err, "File should exist before deletion")

		err = service.DeleteUpload(app.Ctx, upload.ID, user.ID)

		require.NoError(t, err)

		// Verify upload is deleted from database
		_, err = service.GetUpload(app.Ctx, upload.ID, user.ID)
		assert.Error(t, err)
		assert.Equal(t, uploads.ErrUploadNotFound, err)

		// Verify file is deleted from disk
		_, err = os.Stat(filePath)
		assert.Error(t, err, "File should be deleted from disk")
		assert.True(t, os.IsNotExist(err), "File should not exist after deletion")
	})

	t.Run("should return error when upload not found", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		service := newUploadService(app)

		err := service.DeleteUpload(app.Ctx, 99999, user.ID)

		assert.Error(t, err)
		assert.Equal(t, uploads.ErrUploadNotFound, err)
	})
}

func TestUploadService_GetFileType(t *testing.T) {
	t.Run("should return correct file type for images", func(t *testing.T) {
		app := testkit.New(t)
		service := newUploadService(app)

		assert.Equal(t, "image", service.GetFileType("test.jpg"))
		assert.Equal(t, "image", service.GetFileType("test.png"))
		assert.Equal(t, "image", service.GetFileType("test.gif"))
	})

	t.Run("should return correct file type for videos", func(t *testing.T) {
		app := testkit.New(t)
		service := newUploadService(app)

		assert.Equal(t, "video", service.GetFileType("test.mp4"))
		assert.Equal(t, "video", service.GetFileType("test.avi"))
	})

	t.Run("should return correct file type for documents", func(t *testing.T) {
		app := testkit.New(t)
		service := newUploadService(app)

		assert.Equal(t, "document", service.GetFileType("test.pdf"))
		assert.Equal(t, "document", service.GetFileType("test.doc"))
	})

	t.Run("should return other for unknown types", func(t *testing.T) {
		app := testkit.New(t)
		service := newUploadService(app)

		assert.Equal(t, "other", service.GetFileType("test.unknown"))
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
