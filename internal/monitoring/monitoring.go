// Package monitoring wires gopulse, the dashboard served by the API itself, into the
// application: requests, errors, SQL queries, logs and host statistics.
package monitoring

import (
	"log/slog"

	"app/config"

	"github.com/jackc/pgx/v5"
	pulse "github.com/nicklasos/gopulse"
	"github.com/nicklasos/gopulse/pulsepgx"
	"github.com/nicklasos/gopulse/pulseredis"
	"github.com/redis/go-redis/v9"
)

// New starts the recorder, or returns nil when PULSE_PASSWORD is not set. History is kept
// in Redis, so it survives deploys and several instances share one dashboard.
// Call Close on the result during shutdown; Close on nil is safe through the helper below.
func New(cfg *config.Config, rdb *redis.Client) *pulse.Pulse {
	if cfg.PulsePassword == "" {
		return nil
	}
	return pulse.New(pulse.Config{
		App:      cfg.AppName,
		Path:     cfg.PulsePath,
		Username: cfg.PulseUsername,
		Password: cfg.PulsePassword,
		Store:    pulseredis.New(rdb, cfg.AppName),
	})
}

// Close flushes what was recorded. It does nothing when monitoring is off.
func Close(p *pulse.Pulse) {
	if p != nil {
		p.Close()
	}
}

// Record adds a value to a custom metric, for a card on a custom dashboard page.
// It does nothing when monitoring is off.
func Record(p *pulse.Pulse, metric, key string, value float64) {
	if p != nil {
		p.Record(metric, key, value)
	}
}

// QueryTracer times every SQL statement of a pgx pool. It is nil when monitoring is off,
// which db.NewConnection treats as "no tracer".
func QueryTracer(p *pulse.Pulse) pgx.QueryTracer {
	if p == nil {
		return nil
	}
	return pulsepgx.New(p)
}

// LogHandler copies log records to the dashboard on their way to the real output.
// It is nil when monitoring is off, which logger.Config.Wrap treats as "no wrapping".
func LogHandler(p *pulse.Pulse) func(slog.Handler) slog.Handler {
	if p == nil {
		return nil
	}
	return p.SlogHandler
}
