package users

import (
	"database/sql"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(api *echo.Group, db *sql.DB) {
	handler := NewHandler(db)
	
	api.GET("/users", handler.ListUsers)
	api.POST("/users", handler.CreateUser)
	api.GET("/users/:id", handler.GetUser)
}