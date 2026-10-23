package integration

import (
	"net/http"
	"testing"
	"time"

	"app/config"
	"app/internal/monitoring"
	"app/tests/testkit"

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

	testkit.New(t).GET("/_pulse").AssertNotFound()
}

func TestMonitoring_RecordsRequestsAndProtectsTheDashboard(t *testing.T) {
	app := testkit.New(t)
	p := pulse.New(pulse.Config{App: "TestApp", Username: "admin", Password: "secret", HostInterval: -1})
	defer p.Close()

	app.Pulse = p

	get := func(path, user, password string) int {
		req, err := http.NewRequest(http.MethodGet, app.URL(path), nil)
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
	routes, err := p.Totals(app.Ctx, pulse.MetricHTTP, now.Add(-time.Minute), now.Add(time.Minute))
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
}
