package uploads

import (
	"app/internal"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers authenticated upload API routes
func RegisterRoutes(app *internal.App) {
	config := DefaultUploadConfig(app.Config.UploadFolder, app.Config.FilesBaseURL)
	service := NewUploadService(app.Queries, config)
	handler := NewHandler(service, app.Logger)

	uploads := app.Api.Group("/uploads")
	uploads.Use(app.AuthMiddleware)
	{
		uploads.POST("", handler.UploadFile)
		uploads.GET("", handler.ListUploads)
		uploads.GET("/:id", handler.GetUpload)
		uploads.DELETE("/:id", handler.DeleteUpload)
	}
}

// RegisterPublicRoutes serves uploaded files at /api/files/*
func RegisterPublicRoutes(r *gin.Engine, app *internal.App) {
	r.StaticFS("/api/files", http.Dir(app.Config.UploadFolder))
}
