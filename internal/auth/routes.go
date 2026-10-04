package auth

import (
	"app/internal"
	"app/internal/middleware"
)

// RegisterRoutes wires auth routes and sets app.AuthMiddleware for the modules registered after it.
func RegisterRoutes(app *internal.App) {
	authService := NewAuthService(app.Queries, app.Tx, []byte(app.Config.JWTSecret), app.Logger)
	handler := NewAuthHandler(authService, app.Logger)

	app.AuthMiddleware = middleware.UserAuthMiddleware(authService)

	auth := app.Api.Group("/auth")
	{
		auth.POST("/register", handler.Register)
		auth.POST("/login", handler.Login)
		auth.POST("/refresh", handler.RefreshToken)
	}

	userAuth := app.Api.Group("/auth")
	userAuth.Use(app.AuthMiddleware)
	{
		userAuth.GET("/me", handler.GetMe)
		userAuth.PUT("/me", handler.UpdateMe)
		userAuth.PUT("/me/password", handler.UpdatePassword)
		userAuth.POST("/logout", handler.Logout)
	}
}
