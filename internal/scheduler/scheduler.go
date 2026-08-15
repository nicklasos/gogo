package scheduler

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"app/config"
	"app/internal/db"
	"app/internal/logger"
	"app/internal/scheduler/jobs"

	"github.com/robfig/cron/v3"
)

// Dependencies contains all services that might be needed by the scheduler
type Dependencies struct {
	Config  *config.Config
	DB      db.DBTX
	Queries *db.Queries
	Logger  *logger.Logger
}

// Job represents a cron job that can be executed
type Job interface {
	Execute(ctx context.Context) error
	Name() string
	Description() string
}

// Scheduler manages all cron jobs for the application
type Scheduler struct {
	cron *cron.Cron
	deps *Dependencies
	mu   sync.RWMutex
}

// NewScheduler creates a new scheduler instance with all dependencies
func NewScheduler(deps *Dependencies) *Scheduler {
	cronLogger := cron.VerbosePrintfLogger(log.New(os.Stdout, "scheduler: ", log.LstdFlags))
	c := cron.New(cron.WithLogger(cronLogger))

	return &Scheduler{
		cron: c,
		deps: deps,
	}
}

// RegisterJobs registers all application cron jobs
func (s *Scheduler) RegisterJobs() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.registerCleanupRefreshTokensJob(); err != nil {
		return fmt.Errorf("failed to register cleanup refresh tokens job: %w", err)
	}

	return nil
}

// Start begins executing all registered cron jobs
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cron.Start()
	s.deps.Logger.Info("Scheduler started successfully")
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cron.Stop()
	s.deps.Logger.Info("Scheduler stopped gracefully")
}

// GetEntries returns all scheduled cron entries
func (s *Scheduler) GetEntries() []cron.Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cron.Entries()
}

func (s *Scheduler) registerCleanupRefreshTokensJob() error {
	job := jobs.NewCleanupRefreshTokensJob(s.deps.Config, s.deps.Queries, s.deps.Logger)

	_, err := s.cron.AddFunc("@daily", func() {
		if err := job.Execute(context.Background()); err != nil {
			s.deps.Logger.Error("Cleanup refresh tokens job failed", "error", err)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to add cleanup refresh tokens job: %w", err)
	}

	s.deps.Logger.Info("Registered cleanup-refresh-tokens job (daily)")
	return nil
}
