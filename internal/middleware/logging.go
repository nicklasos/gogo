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

const requestIDHeader = "X-Request-ID"

// RequestID puts a request ID on the request context (picked up by the logger) and the
// response header. An inbound ID is reused only when it is safe to write into logs.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(requestIDHeader)
		if !isSafeRequestID(requestID) {
			requestID = uuid.New().String()
		}

		c.Request = c.Request.WithContext(logger.WithRequestID(c.Request.Context(), requestID))
		c.Header(requestIDHeader, requestID)

		c.Next()
	}
}

func isSafeRequestID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
		default:
			return false
		}
	}
	return true
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
