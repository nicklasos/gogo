package integration

import (
	"testing"

	"app/internal/uploads"
	"app/tests/testkit"

	"github.com/stretchr/testify/assert"
)

func TestUploadAPI_UploadFile(t *testing.T) {
	photo := testkit.File{Name: "test.jpg", Content: testkit.JPEG, ContentType: "image/jpeg"}

	t.Run("stores the file for the logged-in user", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		var response uploads.UploadDataResponse
		app.As(user).POSTFile("/api/v1/uploads", "file", photo).AssertOK().JSON(&response)

		assert.True(t, response.Data.ID > 0)
		assert.Equal(t, user.ID, response.Data.UserID)
		assert.Equal(t, user.ID, response.Data.FolderID)
		assert.Equal(t, "image", response.Data.Type)
		assert.Equal(t, "test.jpg", response.Data.OriginalFilename)
		assert.Equal(t, int64(len(testkit.JPEG)), response.Data.FileSize)
		assert.NotEmpty(t, response.Data.RelativePath)
		assert.NotEmpty(t, response.Data.FullURL)
	})

	t.Run("requires a login", func(t *testing.T) {
		app := testkit.New(t)

		app.POSTFile("/api/v1/uploads", "file", photo).AssertUnauthorized()
	})

	t.Run("requires a file", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		app.As(user).POST("/api/v1/uploads", nil).AssertBadRequest()
	})
}
