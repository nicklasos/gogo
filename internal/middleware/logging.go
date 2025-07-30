package middleware

import (
	"fmt"
	"time"

	"myapp/internal/logger"

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
		ctxLogger := log.FromContext(c.Request.Context())
		
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		ip := c.ClientIP()
		userAgent := c.Request.UserAgent()
		
		// Get any errors from context
		errors := c.Errors.ByType(gin.ErrorTypeAny)
		
		if len(errors) > 0 {
			ctxLogger.Error("HTTP request failed",
				"status", status,
				"method", method,
				"path", path,
				"latency_ms", latency.Milliseconds(),
				"client_ip", ip,
				"user_agent", userAgent,
				"errors", errors.String(),
			)
		} else {
			ctxLogger.Info("HTTP request completed",
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

		// Add request ID and HTTP context to logger
		ctx := log.WithRequestID(c.Request.Context(), requestID)
		ctx = log.WithHTTPContext(ctx,
			c.Request.Method,
			c.Request.URL.Path,
			c.Request.UserAgent(),
			c.ClientIP(),
		)

		// Update request context
		c.Request = c.Request.WithContext(ctx)

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
			
			// Default to 500 if not set
			code := c.Writer.Status()
			if code == 200 {
				code = 500
			}
			
			message := "Internal Server Error"
			if err.Error() != "" {
				message = err.Error()
			}

			// Log only 5xx errors (server errors)
			if code >= 500 {
				log.Error(ctx, "HTTP server error", err.Err,
					"status_code", code,
					"error_message", message,
					"method", c.Request.Method,
					"uri", c.Request.URL.Path,
				)
			}

			// Don't send error details in production
			if code >= 500 {
				message = "Internal Server Error"
			}

			// Send JSON error response if not already sent
			if !c.Writer.Written() {
				c.JSON(code, gin.H{
					"error":      message,
					"status":     code,
					"request_id": c.GetHeader("X-Request-ID"),
					"timestamp":  time.Now().UTC().Format(time.RFC3339),
				})
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
		
		log.Error(ctx, "Panic recovered", err,
			"method", c.Request.Method,
			"uri", c.Request.URL.Path,
		)
		
		// Send error response
		c.JSON(500, gin.H{
			"error":      "Internal Server Error",
			"status":     500,
			"request_id": c.GetHeader("X-Request-ID"),
			"timestamp":  time.Now().UTC().Format(time.RFC3339),
		})
	})
}
