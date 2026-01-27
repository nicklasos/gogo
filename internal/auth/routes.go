package auth

import (
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup, handler *AuthHandler, authService *AuthService) {
	// Public routes (no authentication required)
	auth := r.Group("/auth")
	{
		auth.POST("/register", handler.Register)
		auth.POST("/login", handler.Login)
		auth.POST("/refresh", handler.RefreshToken)
	}

	// Protected routes (require user authentication)
	userAuth := r.Group("/auth")
	userAuth.Use(middleware.UserAuthMiddleware(authService))
	{
		userAuth.GET("/me", handler.GetMe)
		userAuth.POST("/logout", handler.Logout)
	}
}
