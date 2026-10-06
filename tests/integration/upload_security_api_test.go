package integration

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"testing"

	"app/internal/db"
	"app/internal/errs"
	"app/internal/factory"
	"app/internal/uploads"
	"app/tests/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func uploadFile(t *testing.T, server *helpers.TestServer, token, filename, clientType string, content []byte) *helpers.TestResponse {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	header.Set("Content-Type", clientType)
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := server.NewRequest("POST", "/api/v1/uploads", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return server.Do(req)
}

func TestUploadAPI_ContentChecks(t *testing.T) {
	t.Run("rejects a file whose content does not match its extension", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := factory.User(t, tx)
			token := helpers.GenerateTestJWT(user.ID, user.Email)
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
				{"image named .pdf", "scan.pdf", helpers.TestJPEG},
			}
			for _, tc := range cases {
				resp := uploadFile(t, server, token, tc.filename, "image/jpeg", tc.content)
				assert.Equal(t, http.StatusBadRequest, resp.StatusCode, tc.name)
				var body errs.ErrorResponse
				require.NoError(t, resp.JSON(&body), tc.name)
				assert.Equal(t, errs.ErrKeyUploadTypeNotAllowed, body.ErrorKey, tc.name)
			}

			var count int
			require.NoError(t, tx.QueryRow(ctx, "SELECT count(*) FROM uploads WHERE user_id = $1", user.ID).Scan(&count))
			assert.Zero(t, count)
		})
	})

	t.Run("records the detected type, not the one the client claims", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := factory.User(t, tx)
			token := helpers.GenerateTestJWT(user.ID, user.Email)

			resp := uploadFile(t, server, token, "../../evil/Photo.JPG", "text/html", helpers.TestJPEG)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			var created uploads.UploadDataResponse
			require.NoError(t, resp.JSON(&created))
			assert.Equal(t, "image/jpeg", created.Data.MimeType)
			assert.Equal(t, "Photo.JPG", created.Data.OriginalFilename)
			assert.NotContains(t, created.Data.RelativePath, "..")
			assert.True(t, strings.HasPrefix(created.Data.RelativePath, fmt.Sprintf("%d/", user.ID)))
			assert.True(t, strings.HasSuffix(created.Data.RelativePath, ".jpg"))

			resp = uploadFile(t, server, token, "doc.pdf", "application/pdf", helpers.TestPDF)
			require.Equal(t, http.StatusOK, resp.StatusCode)
			require.NoError(t, resp.JSON(&created))
			assert.Equal(t, "application/pdf", created.Data.MimeType)
			assert.Equal(t, "document", created.Data.Type)
		})
	})

	t.Run("rejects an unknown extension and an empty file with their own keys", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			server := helpers.CreateTestServer(t, ctx, tx, queries)
			defer server.Close()

			user := factory.User(t, tx)
			token := helpers.GenerateTestJWT(user.ID, user.Email)

			assertErrorKey(t, uploadFile(t, server, token, "run.exe", "application/octet-stream", []byte("MZ")), http.StatusBadRequest, errs.ErrKeyUploadTypeNotAllowed)
			assertErrorKey(t, uploadFile(t, server, token, "page.html", "text/html", []byte("<html></html>")), http.StatusBadRequest, errs.ErrKeyUploadTypeNotAllowed)
			assertErrorKey(t, uploadFile(t, server, token, "empty.jpg", "image/jpeg", nil), http.StatusBadRequest, errs.ErrKeyUploadEmpty)
		})
	})
}

func TestUploadAPI_PublicFiles(t *testing.T) {
	helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
		server := helpers.CreateTestServer(t, ctx, tx, queries)
		defer server.Close()

		user := factory.User(t, tx)
		token := helpers.GenerateTestJWT(user.ID, user.Email)

		var created uploads.UploadDataResponse
		require.NoError(t, uploadFile(t, server, token, "photo.jpg", "image/jpeg", helpers.TestJPEG).JSON(&created))

		resp := server.GET("/api/files/" + created.Data.RelativePath)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, helpers.TestJPEG, resp.Body)
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
			resp := server.GET(path)
			assert.Equal(t, http.StatusNotFound, resp.StatusCode, path)
			assert.Empty(t, resp.Body, path)
		}

		require.Equal(t, http.StatusOK, server.DELETEAuth(fmt.Sprintf("/api/v1/uploads/%d", created.Data.ID), token).StatusCode)
		assert.Equal(t, http.StatusNotFound, server.GET("/api/files/"+created.Data.RelativePath).StatusCode)
	})
}

func TestUploadAPI_ListIsPaginatedAndScopedToTheUser(t *testing.T) {
	helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
		server := helpers.CreateTestServer(t, ctx, tx, queries)
		defer server.Close()

		user := factory.User(t, tx)
		other := factory.User(t, tx, factory.WithEmail("other-uploader@example.com"))
		token := helpers.GenerateTestJWT(user.ID, user.Email)
		for i := 0; i < 3; i++ {
			factory.Upload(t, tx, user.ID)
		}
		foreign := factory.Upload(t, tx, other.ID)

		var page uploads.PaginatedUploadsResponse
		resp := server.GETAuth("/api/v1/uploads?page=1&page_size=2", token)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.NoError(t, resp.JSON(&page))
		assert.Len(t, page.Data, 2)
		assert.Equal(t, int64(3), page.Pagination.Total)
		assert.Equal(t, int32(2), page.Pagination.LastPage)

		require.NoError(t, server.GETAuth("/api/v1/uploads?page=2&page_size=2", token).JSON(&page))
		assert.Len(t, page.Data, 1)

		assert.Equal(t, http.StatusNotFound, server.GETAuth(fmt.Sprintf("/api/v1/uploads/%d", foreign.ID), token).StatusCode)
		assert.Equal(t, http.StatusNotFound, server.DELETEAuth(fmt.Sprintf("/api/v1/uploads/%d", foreign.ID), token).StatusCode)
	})
}
