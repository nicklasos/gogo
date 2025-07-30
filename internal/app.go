package internal

import (
	"myapp/config"
	_ "myapp/docs"
	"myapp/internal/cache"
	"myapp/internal/db"
	"myapp/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	Config  *config.Config
	DB      *pgxpool.Pool
	Queries *db.Queries
	Cache   cache.Cache
	Logger  *logger.Logger
	Api     *gin.RouterGroup
}
