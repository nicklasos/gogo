package internal

import (
	"database/sql"
	"myapp/config"
	_ "myapp/docs"

	"github.com/labstack/echo/v4"
)

type App struct {
	Config *config.Config
	DB     *sql.DB
	Api    *echo.Group
}
