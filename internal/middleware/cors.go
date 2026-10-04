package middleware

import (
	"slices"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORS allows any origin when the list is empty or contains "*", otherwise only the listed origins.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	cfg := cors.Config{
		AllowMethods:  []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Length", "Content-Type", "Authorization", "Accept", requestIDHeader},
		ExposeHeaders: []string{"Content-Length", requestIDHeader},
		MaxAge:        12 * time.Hour,
	}

	if len(allowedOrigins) == 0 || slices.Contains(allowedOrigins, "*") {
		cfg.AllowAllOrigins = true
	} else {
		cfg.AllowOrigins = allowedOrigins
	}

	return cors.New(cfg)
}
