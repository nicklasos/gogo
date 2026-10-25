// Package errortracking reports panics and 5xx errors to Sentry.
// Delivery is asynchronous and best effort: when Sentry is down, slow or the account
// is unpaid, events are dropped and requests are not slowed down or failed.
package errortracking

import (
	"net/http"
	"time"

	"app/config"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
)

const (
	sendTimeout  = 5 * time.Second
	flushTimeout = 2 * time.Second
)

// Init starts the Sentry client. An empty SENTRY_DSN leaves reporting off.
func Init(cfg *config.Config) error {
	if cfg.SentryDSN == "" {
		return nil
	}
	return sentry.Init(sentry.ClientOptions{
		Dsn:         cfg.SentryDSN,
		Environment: cfg.Environment,
		Release:     cfg.AppVersion,
		HTTPClient:  &http.Client{Timeout: sendTimeout},
	})
}

// Enabled reports whether Init started a client.
func Enabled() bool {
	return sentry.CurrentHub().Client() != nil
}

// Middleware reports panics and passes them on to the outer recovery middleware,
// which still writes the 500 response.
func Middleware() gin.HandlerFunc {
	return sentrygin.New(sentrygin.Options{Repanic: true})
}

// CaptureError reports err with the request attached. Without Middleware it does nothing.
func CaptureError(c *gin.Context, err error) {
	if hub := sentrygin.GetHubFromContext(c); hub != nil {
		hub.CaptureException(err)
	}
}

// Flush gives queued events a short time to leave before the process exits.
func Flush() {
	sentry.Flush(flushTimeout)
}
