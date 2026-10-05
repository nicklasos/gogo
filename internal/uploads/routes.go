package uploads

import (
	"net/http"
	"os"

	"app/internal"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers authenticated upload API routes
func RegisterRoutes(app *internal.App) {
	config := DefaultUploadConfig(app.Config.UploadFolder, app.Config.FilesBaseURL)
	service := NewUploadService(app.Queries, config)
	handler := NewHandler(service)

	uploads := app.Api.Group("/uploads")
	uploads.Use(app.AuthMiddleware)
	{
		uploads.POST("", handler.UploadFile)
		uploads.GET("", handler.ListUploads)
		uploads.GET("/:id", handler.GetUpload)
		uploads.DELETE("/:id", handler.DeleteUpload)
	}
}

// RegisterPublicRoutes serves uploaded files at /api/files/*. Only files are served:
// there are no directory listings, and nothing outside the upload folder is reachable.
func RegisterPublicRoutes(r *gin.Engine, app *internal.App) {
	storage := NewLocalStorage(app.Config.UploadFolder, app.Config.FilesBaseURL)

	serve := func(c *gin.Context) {
		full, err := storage.Resolve(c.Param("filepath"))
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		info, err := os.Stat(full)
		if err != nil || info.IsDir() {
			c.Status(http.StatusNotFound)
			return
		}

		// Uploads are user content: the browser must not guess a type for them or let
		// one run scripts in the API's origin.
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Security-Policy", "sandbox; default-src 'none'")
		c.File(full)
	}

	r.GET("/api/files/*filepath", serve)
	r.HEAD("/api/files/*filepath", serve)
}
