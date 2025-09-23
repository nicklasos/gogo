package internal

import (
	"app/config"
	"app/internal/db"
	"app/internal/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CLIApp contains shared components for CLI commands
type CLIApp struct {
	Config   *config.Config
	Database *pgxpool.Pool
	Queries  *db.Queries
	Logger   *logger.Logger
}
