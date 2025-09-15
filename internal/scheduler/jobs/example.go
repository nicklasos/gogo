package jobs

import (
	"context"

	"app/config"
	"app/internal/db"
	"app/internal/logger"
)

// ExampleJob demonstrates how to create a cron job
type ExampleJob struct {
	config  *config.Config
	queries *db.Queries
	logger  *logger.Logger
}

// NewExampleJob creates a new example job
func NewExampleJob(config *config.Config, queries *db.Queries, logger *logger.Logger) *ExampleJob {
	return &ExampleJob{
		config:  config,
		queries: queries,
		logger:  logger,
	}
}

// Execute runs the example job logic
func (j *ExampleJob) Execute(ctx context.Context) error {
	j.logger.Info(ctx, "Starting example cron job")
	
	// Example: Log some system information
	j.logger.Info(ctx, "Example job executed successfully", 
		"app_name", j.config.AppName,
		"environment", j.config.Environment,
		"port", j.config.Port)
	
	// Here you could add actual job logic like:
	// - Database cleanup
	// - Report generation  
	// - Data synchronization
	// - Email notifications
	// - File processing
	// - API calls to external services
	
	j.logger.Info(ctx, "Example cron job completed")
	return nil
}

// Name returns the job name
func (j *ExampleJob) Name() string {
	return "example-job"
}

// Description returns the job description
func (j *ExampleJob) Description() string {
	return "Example cron job demonstrating the scheduler pattern for the gogo project"
}