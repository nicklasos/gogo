package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Logger wraps slog.Logger with additional context methods
type Logger struct {
	*slog.Logger
}

// Config holds logger configuration
type Config struct {
	Level     string // debug, info, warn, error
	Format    string // json, text
	Output    string // file path, "stdout", "stderr", or "both"
	AddSource bool   // add source code position
	RequestID bool   // enable request ID tracking
}

// New creates a new structured logger
func New(cfg Config) (*Logger, error) {
	// Parse log level
	var level slog.Level
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Configure output writer
	var writer io.Writer
	// Trim whitespace from config value to handle potential formatting issues
	output := strings.TrimSpace(cfg.Output)

	switch output {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	case "both":
		// Default: write to both file and stdout
		file, err := createLogFile("logs/app.log")
		if err != nil {
			return nil, err
		}
		writer = io.MultiWriter(file, os.Stdout)
	default:
		// File path specified or default to logs/app.log
		if output == "" {
			output = "logs/app.log"
		}
		file, err := createLogFile(output)
		if err != nil {
			return nil, err
		}
		writer = io.MultiWriter(file, os.Stdout) // Always include stdout for K8s
	}

	// Configure handler options
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	// Choose handler based on format
	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}, nil
}

// createLogFile creates log file with proper permissions
func createLogFile(filename string) (*os.File, error) {
	// Create logs directory if it doesn't exist
	logDir := filepath.Dir(filename)
	if logDir != "." && logDir != "" {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, err
		}
	}

	// Open file with append mode
	return os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

// Helper method to create a logger with error included in args
func (l *Logger) WithError(err error) *slog.Logger {
	return l.With("error", err)
}
