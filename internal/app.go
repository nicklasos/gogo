package internal

import (
	"database/sql"
	"myapp/config"
	_ "myapp/docs"
	"myapp/internal/cache"
	"myapp/internal/logger"

	"github.com/labstack/echo/v4"
)

type App struct {
	Config *config.Config
	DB     *sql.DB
	Cache  cache.Cache
	Logger *logger.Logger
	Api    *echo.Group
}
