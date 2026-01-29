package uploads

import (
	"app/internal"
	"app/internal/middleware"
)

// RegisterRoutes registers upload routes
func RegisterRoutes(app *internal.App, authService middleware.UserJWTVerifier) {
	config := DefaultUploadConfig(app.Config.UploadFolder, app.Config.FilesBaseURL)
	service := NewUploadService(app.Queries, config)
	handler := NewHandler(service, app.Logger)

	uploads := app.Api.Group("/uploads")
	uploads.Use(middleware.UserAuthMiddleware(authService))
	{
		uploads.POST("", handler.UploadFile)
	}
}
