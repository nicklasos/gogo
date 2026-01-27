package example

import (
	"app/internal"
	"app/internal/middleware"
)

func RegisterRoutes(app *internal.App, authService middleware.UserJWTVerifier) {
	// Create service with only the dependencies it needs
	service := NewExampleService(app.Queries)

	// Create handler with only the service it needs
	handler := NewHandler(service, app.Logger)

	// Protected routes (require user authentication)
	examples := app.Api.Group("/examples")
	examples.Use(middleware.UserAuthMiddleware(authService))
	{
		examples.POST("", handler.CreateExample)
		examples.GET("", handler.ListExamples)
		examples.GET("/:id", handler.GetExample)
		examples.PUT("/:id", handler.UpdateExample)
		examples.DELETE("/:id", handler.DeleteExample)
	}
}
