package example

import (
	"app/internal"
)

func RegisterRoutes(app *internal.App) {
	service := NewExampleService(app.Queries, app.Cache)
	handler := NewHandler(service)

	examples := app.Api.Group("/examples")
	examples.Use(app.AuthMiddleware)
	{
		examples.POST("", handler.CreateExample)
		examples.GET("", handler.ListExamples)
		examples.GET("/:id", handler.GetExample)
		examples.PUT("/:id", handler.UpdateExample)
		examples.DELETE("/:id", handler.DeleteExample)
	}
}
