package users

import "myapp/internal"

func RegisterRoutes(app *internal.App) {
	handler := NewHandler(app.DB)

	api := app.Api.Group("/users")

	api.GET("/", handler.ListUsers)
	api.POST("/", handler.CreateUser)
	api.GET("/:id", handler.GetUser)
}
