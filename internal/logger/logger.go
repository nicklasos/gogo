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
	RequestID bool   // add the request ID from the context to every record
	// Wrap, when set, decorates the output handler, for example to also feed a monitoring dashboard.
	Wrap func(slog.Handler) slog.Handler
}

// New creates a new structured logger
func New(cfg Config) (*Logger, error) {
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

	var writer io.Writer
	output := strings.TrimSpace(cfg.Output)

	switch output {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	case "both":
		file, err := createLogFile("logs/app.log")
		if err != nil {
			return nil, err
		}
		writer = io.MultiWriter(file, os.Stdout)
	default:
		if output == "" {
			output = "logs/app.log"
		}
		file, err := createLogFile(output)
		if err != nil {
			return nil, err
		}
		writer = io.MultiWriter(file, os.Stdout) // Always include stdout for K8s
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}

	if cfg.Wrap != nil {
		handler = cfg.Wrap(handler)
	}
	if cfg.RequestID {
		handler = contextHandler{handler}
	}

	return &Logger{
		Logger: slog.New(handler),
	}, nil
}

// createLogFile creates log file with proper permissions
func createLogFile(filename string) (*os.File, error) {
	logDir := filepath.Dir(filename)
	if logDir != "." && logDir != "" {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, err
		}
	}

	return os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

// Helper method to create a logger with error included in args
func (l *Logger) WithError(err error) *slog.Logger {
	return l.With("error", err)
}
