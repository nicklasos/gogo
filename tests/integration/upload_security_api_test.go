package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"app/internal/db"
	"app/internal/errs"
	"app/internal/factory"
	"app/internal/uploads"
	"app/tests/testkit"

	"github.com/stretchr/testify/assert"
)

func uploadAs(app *testkit.App, user *db.User, filename, clientType string, content []byte) *testkit.Response {
	return app.As(user).POSTFile("/api/v1/uploads", "file", testkit.File{Name: filename, Content: content, ContentType: clientType})
}

func TestUploadAPI_ContentChecks(t *testing.T) {
	t.Run("rejects a file whose content does not match its extension", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()
		html := []byte("<html><body><script>alert(1)</script></body></html>")

		cases := []struct {
			name     string
			filename string
			content  []byte
		}{
			{"HTML named .jpg", "photo.jpg", html},
			{"HTML named .pdf", "report.pdf", html},
			{"HTML named .txt", "notes.txt", html},
			{"text named .png", "pixel.png", []byte("just some text")},
			{"image named .pdf", "scan.pdf", testkit.JPEG},
		}
		for _, tc := range cases {
			t.Log(tc.name)
			uploadAs(app, user, tc.filename, "image/jpeg", tc.content).AssertError(http.StatusBadRequest, errs.ErrKeyUploadTypeNotAllowed)
		}

		app.AssertDatabaseCount("uploads", 0, testkit.Row{"user_id": user.ID})
	})

	t.Run("records the detected type, not the one the client claims", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		var created uploads.UploadDataResponse
		uploadAs(app, user, "../../evil/Photo.JPG", "text/html", testkit.JPEG).AssertOK().JSON(&created)
		assert.Equal(t, "image/jpeg", created.Data.MimeType)
		assert.Equal(t, "Photo.JPG", created.Data.OriginalFilename)
		assert.NotContains(t, created.Data.RelativePath, "..")
		assert.True(t, strings.HasPrefix(created.Data.RelativePath, fmt.Sprintf("%d/", user.ID)))
		assert.True(t, strings.HasSuffix(created.Data.RelativePath, ".jpg"))

		uploadAs(app, user, "doc.pdf", "application/pdf", testkit.PDF).AssertOK().JSON(&created)
		assert.Equal(t, "application/pdf", created.Data.MimeType)
		assert.Equal(t, "document", created.Data.Type)
	})

	t.Run("rejects an unknown extension and an empty file with their own keys", func(t *testing.T) {
		app := testkit.New(t)
		user := app.Create.User()

		uploadAs(app, user, "run.exe", "application/octet-stream", []byte("MZ")).AssertError(http.StatusBadRequest, errs.ErrKeyUploadTypeNotAllowed)
		uploadAs(app, user, "page.html", "text/html", []byte("<html></html>")).AssertError(http.StatusBadRequest, errs.ErrKeyUploadTypeNotAllowed)
		uploadAs(app, user, "empty.jpg", "image/jpeg", nil).AssertError(http.StatusBadRequest, errs.ErrKeyUploadEmpty)
	})
}

func TestUploadAPI_PublicFiles(t *testing.T) {
	app := testkit.New(t)
	user := app.Create.User()

	var created uploads.UploadDataResponse
	uploadAs(app, user, "photo.jpg", "image/jpeg", testkit.JPEG).JSON(&created)

	resp := app.GET("/api/files/" + created.Data.RelativePath).AssertOK()
	assert.Equal(t, testkit.JPEG, resp.Body)
	assert.Equal(t, "image/jpeg", resp.Header.Get("Content-Type"))
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "sandbox")

	for _, path := range []string{
		"/api/files/",
		fmt.Sprintf("/api/files/%d", user.ID),
		fmt.Sprintf("/api/files/%d/", user.ID),
		"/api/files/missing.jpg",
		"/api/files/..%2f..%2f..%2fetc%2fpasswd",
		"/api/files/%2e%2e/%2e%2e/go.mod",
	} {
		resp := app.GET(path)
		assert.Equal(t, http.StatusNotFound, resp.StatusCode, path)
		assert.Empty(t, resp.Body, path)
	}

	app.As(user).DELETE("/api/v1/uploads/%d", created.Data.ID).AssertOK()
	app.GET("/api/files/" + created.Data.RelativePath).AssertNotFound()
}

func TestUploadAPI_ListIsPaginatedAndScopedToTheUser(t *testing.T) {
	app := testkit.New(t)
	user := app.Create.User()
	other := app.Create.User(factory.WithEmail("other-uploader@example.com"))
	for i := 0; i < 3; i++ {
		app.Create.Upload(user.ID)
	}
	foreign := app.Create.Upload(other.ID)

	var page uploads.PaginatedUploadsResponse
	resp := app.As(user).GET("/api/v1/uploads?page=1&page_size=2").AssertOK()
	resp.JSON(&page)
	assert.Len(t, page.Data, 2)
	assert.Equal(t, int64(3), page.Pagination.Total)
	assert.Equal(t, int32(2), page.Pagination.LastPage)

	app.As(user).GET("/api/v1/uploads?page=2&page_size=2").JSON(&page)
	assert.Len(t, page.Data, 1)

	app.As(user).GET("/api/v1/uploads/%d", foreign.ID).AssertNotFound()
	app.As(user).DELETE("/api/v1/uploads/%d", foreign.ID).AssertNotFound()
}
