package scheduler

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/robfig/cron/v3"
	"app/config"
	"app/internal/db"
	"app/internal/logger"
	"app/internal/scheduler/jobs"
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
	// Create cron with logger
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

	// Register example job
	if err := s.registerExampleJob(); err != nil {
		return fmt.Errorf("failed to register example job: %w", err)
	}

	return nil
}

// Start begins executing all registered cron jobs
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.cron.Start()
	s.deps.Logger.Info(context.Background(), "Scheduler started successfully")
}

// Stop gracefully shuts down the scheduler
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.cron.Stop()
	s.deps.Logger.Info(context.Background(), "Scheduler stopped gracefully")
}

// GetEntries returns all scheduled cron entries
func (s *Scheduler) GetEntries() []cron.Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.cron.Entries()
}

// Private job registration methods

func (s *Scheduler) registerExampleJob() error {
	// Initialize job once
	job := jobs.NewExampleJob(s.deps.Config, s.deps.Queries, s.deps.Logger)
	
	// Run example job every 2 hours
	_, err := s.cron.AddFunc("@every 2h", func() {
		if err := job.Execute(context.Background()); err != nil {
			s.deps.Logger.Error(context.Background(), "Example job failed", err)
		}
	})
	
	if err != nil {
		return fmt.Errorf("failed to add example job: %w", err)
	}
	
	s.deps.Logger.Info(context.Background(), "Registered example job (every 2 hours)")
	return nil
}