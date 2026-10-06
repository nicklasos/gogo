package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app/config"
	"app/internal"
	"app/internal/cache"
	"app/internal/db"
	"app/internal/mail"
	"app/internal/monitoring"
	"app/internal/server"
	"app/tests/helpers"

	"github.com/jackc/pgx/v5"
	pulse "github.com/nicklasos/gopulse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonitoring_IsOffWithoutAPassword(t *testing.T) {
	assert.Nil(t, monitoring.New(&config.Config{AppName: "TestApp"}, nil))

	// The helpers must be safe to call with monitoring off
	monitoring.Record(nil, "orders", "created", 1)
	monitoring.Close(nil)
	assert.Nil(t, monitoring.QueryTracer(nil))
	assert.Nil(t, monitoring.LogHandler(nil))

	helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
		srv := helpers.CreateTestServer(t, ctx, tx, queries)
		defer srv.Close()
		assert.Equal(t, http.StatusNotFound, srv.GET("/_pulse").StatusCode)
	})
}

func TestMonitoring_RecordsRequestsAndProtectsTheDashboard(t *testing.T) {
	helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
		p := pulse.New(pulse.Config{App: "TestApp", Username: "admin", Password: "secret", HostInterval: -1})
		defer p.Close()

		cfg := &config.Config{JWTSecret: helpers.TestJWTSecret, AppName: "TestApp", UploadFolder: t.TempDir(), AllowRegistration: true}
		log := helpers.GetTestLogger(t)
		router := server.NewEngine(cfg, log, p)
		server.RegisterRoutes(router, &internal.App{
			Config:  cfg,
			Queries: queries,
			Tx:      db.NewTxRunner(tx, queries),
			Cache:   cache.NewMemoryCache(),
			Logger:  log,
			Mail:    mail.NewMemorySender(),
			Pulse:   p,
			Api:     router.Group("/api/v1"),
		})
		srv := httptest.NewServer(router)
		defer srv.Close()

		get := func(path, user, password string) int {
			req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
			require.NoError(t, err)
			if user != "" {
				req.SetBasicAuth(user, password)
			}
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			return resp.StatusCode
		}

		assert.Equal(t, http.StatusUnauthorized, get("/_pulse", "", ""))
		assert.Equal(t, http.StatusUnauthorized, get("/_pulse", "admin", "wrong"))
		assert.Equal(t, http.StatusOK, get("/_pulse", "admin", "secret"))

		for i := 0; i < 3; i++ {
			assert.Equal(t, http.StatusOK, get("/health", "", ""))
		}
		assert.Equal(t, http.StatusUnauthorized, get("/api/v1/auth/me", "", ""))
		assert.Equal(t, http.StatusNotFound, get("/no-such-route", "", ""))
		p.Flush()

		now := time.Now()
		routes, err := p.Totals(ctx, pulse.MetricHTTP, now.Add(-time.Minute), now.Add(time.Minute))
		require.NoError(t, err)

		counts := map[string]int64{}
		for route, agg := range routes {
			counts[route] = agg.Count
		}
		assert.Equal(t, int64(3), counts["GET /health"])
		assert.Equal(t, int64(1), counts["GET /api/v1/auth/me"])
		assert.Equal(t, int64(1), counts["GET "+pulse.Unmatched], "unknown paths share one bucket")
		for route := range counts {
			assert.NotContains(t, route, "_pulse", "the dashboard does not record its own requests")
		}
	})
}
