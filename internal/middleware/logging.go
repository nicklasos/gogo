package middleware

import (
	"time"

	"myapp/internal/logger"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

// RequestLogging creates a structured request logging middleware
func RequestLogging(log *logger.Logger) echo.MiddlewareFunc {
	return echomiddleware.RequestLoggerWithConfig(echomiddleware.RequestLoggerConfig{
		LogStatus:    true,
		LogURI:       true,
		LogError:     true,
		LogMethod:    true,
		LogLatency:   true,
		LogRemoteIP:  true,
		LogUserAgent: true,
		HandleError:  true, // Continue processing on errors
		LogValuesFunc: func(c echo.Context, v echomiddleware.RequestLoggerValues) error {
			// Get logger from context (should have request ID)
			ctxLogger := log.FromContext(c.Request().Context())

			if v.Error == nil {
				ctxLogger.Info("HTTP request completed",
					"status", v.Status,
					"method", v.Method,
					"uri", v.URI,
					"latency_ms", v.Latency.Milliseconds(),
					"remote_ip", v.RemoteIP,
					"user_agent", v.UserAgent,
				)
			} else {
				ctxLogger.Error("HTTP request failed",
					"status", v.Status,
					"method", v.Method,
					"uri", v.URI,
					"latency_ms", v.Latency.Milliseconds(),
					"remote_ip", v.RemoteIP,
					"user_agent", v.UserAgent,
					"error", v.Error.Error(),
				)
			}
			return nil
		},
	})
}

// RequestID middleware adds request ID to context and response headers
func RequestID(log *logger.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Generate or extract request ID
			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Add to response header
			c.Response().Header().Set("X-Request-ID", requestID)

			// Add request ID and HTTP context to logger
			ctx := log.WithRequestID(c.Request().Context(), requestID)
			ctx = log.WithHTTPContext(ctx,
				c.Request().Method,
				c.Request().URL.Path,
				c.Request().UserAgent(),
				c.RealIP(),
			)

			// Update request context
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// ErrorHandler creates a custom error handler that logs all errors
func ErrorHandler(log *logger.Logger) func(err error, c echo.Context) {
	return func(err error, c echo.Context) {
		// Get logger from context
		ctx := c.Request().Context()

		// Default to 500 if not an Echo HTTP error
		code := 500
		message := "Internal Server Error"

		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			if msg, ok := he.Message.(string); ok {
				message = msg
			}
		}

		// Log all 5xx errors and some 4xx errors
		if code >= 500 {
			log.Error(ctx, "HTTP server error", err,
				"status_code", code,
				"error_message", message,
				"method", c.Request().Method,
				"uri", c.Request().URL.Path,
			)
		} else if code == 404 || code == 401 || code == 403 {
			// Log important 4xx errors at warn level
			ctxLogger := log.FromContext(ctx)
			ctxLogger.Warn("HTTP client error",
				"status_code", code,
				"error_message", message,
				"method", c.Request().Method,
				"uri", c.Request().URL.Path,
			)
		}

		// Don't send error details in production
		if code >= 500 {
			message = "Internal Server Error"
		}

		// Send JSON error response
		if !c.Response().Committed {
			if c.Request().Method == "HEAD" {
				err = c.NoContent(code)
			} else {
				err = c.JSON(code, map[string]interface{}{
					"error":      message,
					"status":     code,
					"request_id": c.Response().Header().Get("X-Request-ID"),
					"timestamp":  time.Now().UTC().Format(time.RFC3339),
				})
			}
			if err != nil {
				log.Error(ctx, "Failed to send error response", err)
			}
		}
	}
}

// Recovery middleware with structured logging
func Recovery(log *logger.Logger) echo.MiddlewareFunc {
	return echomiddleware.RecoverWithConfig(echomiddleware.RecoverConfig{
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			ctx := c.Request().Context()
			log.Error(ctx, "Panic recovered", err,
				"stack_trace", string(stack),
				"method", c.Request().Method,
				"uri", c.Request().URL.Path,
			)
			return nil
		},
	})
}
