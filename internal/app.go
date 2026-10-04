package internal

import (
	"app/config"
	_ "app/docs"
	"app/internal/cache"
	"app/internal/db"
	"app/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	Config  *config.Config
	DB      *pgxpool.Pool
	Queries *db.Queries
	Tx      *db.TxRunner
	Cache   cache.Cache
	Logger  *logger.Logger
	Api     *gin.RouterGroup
	Images  *ImageService

	// AuthMiddleware is set by auth.RegisterRoutes; modules registered after it use it to protect routes
	AuthMiddleware gin.HandlerFunc
}
