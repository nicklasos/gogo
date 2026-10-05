package users

import (
	"app/internal"
	"app/internal/middleware"
)

func RegisterRoutes(app *internal.App) {
	service := NewUserService(app.Queries, app.Tx)
	handler := NewHandler(service)

	users := app.Api.Group("/users")
	users.Use(app.AuthMiddleware)
	users.Use(middleware.RequireRole(middleware.RoleAdmin))
	{
		users.GET("", handler.ListUsers)
		users.POST("", handler.CreateUser)
		users.PUT("/:id", handler.UpdateUser)
		users.POST("/:id/set-password", handler.SetPassword)
		users.DELETE("/:id", handler.DeleteUser)
	}
}
