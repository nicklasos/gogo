package auth

import (
	"time"

	"app/internal"
	"app/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires auth routes and sets app.AuthMiddleware for the modules registered after it.
func RegisterRoutes(app *internal.App) {
	cfg := app.Config
	authService := NewAuthService(app.Queries, app.Tx, app.Mail, app.Logger, Options{
		JWTSecret:            []byte(cfg.JWTSecret),
		AccessTokenTTL:       cfg.AccessTokenTTL,
		RefreshTokenTTL:      cfg.RefreshTokenTTL,
		PasswordResetTTL:     cfg.PasswordResetTTL,
		EmailVerificationTTL: cfg.EmailVerificationTTL,
		FrontendURL:          cfg.FrontendURL,
		AppName:              cfg.AppName,
	})
	handler := NewAuthHandler(authService)

	app.AuthMiddleware = middleware.UserAuthMiddleware(authService)

	loginLimit, refreshLimit, emailLimit := noLimit, noLimit, noLimit
	if cfg.AuthRateLimit {
		loginLimit = middleware.AuthRateLimit(app.Cache, app.Logger)
		refreshLimit = middleware.RateLimit(app.Cache, app.Logger, "refresh", 60, time.Minute)
		// These send mail or check emailed tokens, so they get a much smaller budget
		emailLimit = middleware.RateLimit(app.Cache, app.Logger, "email_link", 10, 10*time.Minute)
	}

	auth := app.Api.Group("/auth")
	{
		if cfg.AllowRegistration {
			auth.POST("/register", emailLimit, handler.Register)
		} else {
			auth.POST("/register", RegistrationDisabled)
		}
		auth.POST("/login", loginLimit, handler.Login)
		auth.POST("/refresh", refreshLimit, handler.RefreshToken)
		auth.POST("/forgot-password", emailLimit, handler.ForgotPassword)
		auth.POST("/reset-password", emailLimit, handler.ResetPassword)
		auth.POST("/verify-email", emailLimit, handler.VerifyEmail)
	}

	userAuth := app.Api.Group("/auth")
	userAuth.Use(app.AuthMiddleware)
	{
		userAuth.GET("/me", handler.GetMe)
		userAuth.PUT("/me", handler.UpdateMe)
		userAuth.PUT("/me/password", handler.UpdatePassword)
		userAuth.POST("/me/verify-email", emailLimit, handler.ResendVerification)
		userAuth.POST("/logout", handler.Logout)
	}
}

func noLimit(c *gin.Context) {
	c.Next()
}
