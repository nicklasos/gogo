package users

import "myapp/internal"

func RegisterRoutes(app *internal.App) {
	handler := NewHandler(app.DB)

	app.Api.GET("/users", handler.ListUsers)
	app.Api.POST("/users", handler.CreateUser)
	app.Api.GET("/users/:id", handler.GetUser)
}
