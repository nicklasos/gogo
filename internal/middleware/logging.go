package middleware

import (
	"fmt"
	"time"

	"app/internal/errs"
	"app/internal/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestLogging creates a structured request logging middleware
func RequestLogging(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Log after request completes
		latency := time.Since(start)

		status := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		ip := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Get any errors from context
		errors := c.Errors.ByType(gin.ErrorTypeAny)

		if len(errors) > 0 {
			log.ErrorContext(c.Request.Context(), "HTTP request failed",
				"status", status,
				"method", method,
				"path", path,
				"latency_ms", latency.Milliseconds(),
				"client_ip", ip,
				"user_agent", userAgent,
				"errors", errors.String(),
			)
		} else {
			log.InfoContext(c.Request.Context(), "HTTP request completed",
				"status", status,
				"method", method,
				"path", path,
				"latency_ms", latency.Milliseconds(),
				"client_ip", ip,
				"user_agent", userAgent,
			)
		}
	}
}

// RequestID middleware adds request ID to context and response headers
func RequestID(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate or extract request ID
		requestID := c.Request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add to response header
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// ErrorHandler creates a middleware that handles errors and sends appropriate responses
func ErrorHandler(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			ctx := c.Request.Context()

			// Log only 5xx errors (server errors)
			if c.Writer.Status() >= 500 {
				log.ErrorContext(ctx, "HTTP server error",
					"error", err.Err,
					"status_code", c.Writer.Status(),
					"error_message", err.Error(),
					"method", c.Request.Method,
					"uri", c.Request.URL.Path,
				)
			}

			// Send JSON error response if not already sent
			if !c.Writer.Written() {
				// Use the new error response system
				errs.RespondWithError(c, err.Err)
			}
		}
	}
}

// Recovery middleware with structured logging
func Recovery(log *logger.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		ctx := c.Request.Context()

		// Convert recovered value to error
		var err error
		if e, ok := recovered.(error); ok {
			err = e
		} else {
			err = fmt.Errorf("panic: %v", recovered)
		}

		log.ErrorContext(ctx, "Panic recovered",
			"error", err,
			"method", c.Request.Method,
			"uri", c.Request.URL.Path,
		)

		// Send error response
		errs.RespondWithInternalError(c, "Internal Server Error")
	})
}
