package internal

import (
	"app/config"
	_ "app/docs"
	"app/internal/cache"
	"app/internal/db"
	"app/internal/logger"
	"app/internal/mail"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	pulse "github.com/nicklasos/gopulse"
)

type App struct {
	Config  *config.Config
	DB      *pgxpool.Pool
	Queries *db.Queries
	Tx      *db.TxRunner
	Cache   cache.Cache
	Logger  *logger.Logger
	Mail    mail.Sender
	// Pulse is the monitoring recorder, or nil when the dashboard is off. Use internal/monitoring
	// helpers rather than calling it directly, so modules do not have to check for nil.
	Pulse *pulse.Pulse
	Api   *gin.RouterGroup

	// AuthMiddleware is set by auth.RegisterRoutes; modules registered after it use it to protect routes
	AuthMiddleware gin.HandlerFunc
}
