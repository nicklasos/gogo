package logger

import (
	"context"
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
	Level      string // debug, info, warn, error
	Format     string // json, text
	Output     string // file path, "stdout", "stderr", or "both"
	AddSource  bool   // add source code position
	RequestID  bool   // enable request ID tracking
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

// WithRequestID adds request ID to logger context
func (l *Logger) WithRequestID(ctx context.Context, requestID string) context.Context {
	logger := l.With("request_id", requestID)
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves logger from context or returns default
func (l *Logger) FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return l.Logger
}

// Info logs info message with optional context
func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	if ctx != nil {
		logger := l.FromContext(ctx)
		logger.Info(msg, args...)
	} else {
		l.Logger.Info(msg, args...)
	}
}

// Error logs error with context and prepares for Sentry
func (l *Logger) Error(ctx context.Context, msg string, err error, args ...any) {
	var logger *slog.Logger
	if ctx != nil {
		logger = l.FromContext(ctx)
	} else {
		logger = l.Logger
	}
	
	// Add error to args
	allArgs := append([]any{"error", err}, args...)
	
	// Log the error
	logger.Error(msg, allArgs...)
	
	// TODO: Send to Sentry when implemented
	// sentry.CaptureException(err)
}

// HTTP request context helpers
func (l *Logger) WithHTTPContext(ctx context.Context, method, path, userAgent, ip string) context.Context {
	logger := l.With(
		"http_method", method,
		"http_path", path,
		"user_agent", userAgent,
		"client_ip", ip,
	)
	return context.WithValue(ctx, loggerKey, logger)
}

// Database operation helpers
func (l *Logger) WithDBContext(ctx context.Context, operation, table string) context.Context {
	logger := l.FromContext(ctx).With(
		"db_operation", operation,
		"db_table", table,
	)
	return context.WithValue(ctx, loggerKey, logger)
}

// Service operation helpers  
func (l *Logger) WithServiceContext(ctx context.Context, service, operation string) context.Context {
	logger := l.FromContext(ctx).With(
		"service", service,
		"operation", operation,
	)
	return context.WithValue(ctx, loggerKey, logger)
}

// Context key for logger
type contextKey string

const loggerKey contextKey = "logger"