package jobs

import (
	"context"

	"app/config"
	"app/internal/db"
	"app/internal/logger"
)

// CleanupRefreshTokensJob deletes expired refresh tokens from the database.
type CleanupRefreshTokensJob struct {
	config  *config.Config
	queries *db.Queries
	logger  *logger.Logger
}

// NewCleanupRefreshTokensJob creates a new cleanup job
func NewCleanupRefreshTokensJob(config *config.Config, queries *db.Queries, logger *logger.Logger) *CleanupRefreshTokensJob {
	return &CleanupRefreshTokensJob{
		config:  config,
		queries: queries,
		logger:  logger,
	}
}

// Execute runs the cleanup
func (j *CleanupRefreshTokensJob) Execute(ctx context.Context) error {
	j.logger.InfoContext(ctx, "Starting refresh token cleanup job")

	if err := j.queries.DeleteExpiredRefreshTokens(ctx); err != nil {
		j.logger.ErrorContext(ctx, "Failed to delete expired refresh tokens", "error", err)
		return err
	}

	j.logger.InfoContext(ctx, "Refresh token cleanup job completed")
	return nil
}

// Name returns the job name
func (j *CleanupRefreshTokensJob) Name() string {
	return "cleanup-refresh-tokens"
}

// Description returns the job description
func (j *CleanupRefreshTokensJob) Description() string {
	return "Deletes expired refresh tokens from the database"
}
