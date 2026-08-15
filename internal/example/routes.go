package example

import (
	"app/internal"
	"app/internal/middleware"
)

func RegisterRoutes(app *internal.App, authService middleware.UserJWTVerifier) {
	service := NewExampleService(app.Queries, app.Cache)
	handler := NewHandler(service, app.Logger)

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
