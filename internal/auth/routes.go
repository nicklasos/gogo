package auth

import (
	"app/internal"
	"app/internal/middleware"
)

// RegisterRoutes wires auth routes and returns the AuthService for reuse by other modules.
func RegisterRoutes(app *internal.App) *AuthService {
	authService := NewAuthService(app.Queries, []byte(app.Config.JWTSecret), app.Logger)
	handler := NewAuthHandler(authService, app.Logger)

	auth := app.Api.Group("/auth")
	{
		auth.POST("/register", handler.Register)
		auth.POST("/login", handler.Login)
		auth.POST("/refresh", handler.RefreshToken)
	}

	userAuth := app.Api.Group("/auth")
	userAuth.Use(middleware.UserAuthMiddleware(authService))
	{
		userAuth.GET("/me", handler.GetMe)
		userAuth.POST("/logout", handler.Logout)
	}

	return authService
}
