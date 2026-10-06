package server

import (
	"app/config"
	"app/internal"
	"app/internal/auth"
	"app/internal/example"
	"app/internal/health"
	"app/internal/logger"
	"app/internal/middleware"
	"app/internal/uploads"
	"app/internal/users"

	"github.com/gin-gonic/gin"
	pulse "github.com/nicklasos/gopulse"
	"github.com/nicklasos/gopulse/pulsegin"
)

// NewEngine builds the Gin engine with the middleware shared by the API binary and the test server.
// p is the monitoring recorder; nil leaves the dashboard and its middleware out.
func NewEngine(cfg *config.Config, log *logger.Logger, p *pulse.Pulse) *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = false

	// Rate limiting keys on the client IP, so X-Forwarded-For is believed only from these proxies.
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Error("Invalid TRUSTED_PROXIES, trusting no proxy", "error", err)
		_ = r.SetTrustedProxies(nil)
	}

	r.Use(middleware.RequestID())
	r.Use(middleware.Recovery(log))
	if p != nil {
		// After Recovery: a panic is recorded with its stack and re-raised for Recovery to answer
		r.Use(pulsegin.Middleware(p))
		pulsegin.Mount(r, p)
	}
	// r.Use(middleware.RequestLogging(log))
	r.Use(middleware.ErrorHandler(log))
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))

	return r
}

// RegisterRoutes is the single list of modules. auth goes first because it sets app.AuthMiddleware.
func RegisterRoutes(r *gin.Engine, app *internal.App) {
	auth.RegisterRoutes(app)
	users.RegisterRoutes(app)
	example.RegisterRoutes(app)
	uploads.RegisterRoutes(app)
	uploads.RegisterPublicRoutes(r, app)
	health.RegisterRoutes(r, app)
}
