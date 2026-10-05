package health

import (
	"context"
	"net/http"
	"time"

	"app/config"
	"app/internal/cache"
	"app/internal/db"
	"app/internal/logger"

	"github.com/gin-gonic/gin"
)

const (
	statusOK    = "ok"
	statusError = "error"

	checkTimeout = 3 * time.Second
)

type Handler struct {
	queries *db.Queries
	cache   cache.Cache
	cfg     *config.Config
	logger  *logger.Logger
}

func NewHandler(queries *db.Queries, cache cache.Cache, cfg *config.Config, logger *logger.Logger) *Handler {
	return &Handler{queries: queries, cache: cache, cfg: cfg, logger: logger}
}

// Check reports every dependency the API cannot serve requests without. It answers 503
// when any of them is down, so a load balancer stops routing to this instance.
func (h *Handler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), checkTimeout)
	defer cancel()

	checks := gin.H{"database": h.status(ctx, "database", h.queries.Healthcheck(ctx))}
	if pinger, ok := h.cache.(cache.Pinger); ok {
		checks["cache"] = h.status(ctx, "cache", pinger.Ping(ctx))
	}

	healthy := true
	for _, status := range checks {
		if status != statusOK {
			healthy = false
		}
	}

	payload := gin.H{
		"status":  "healthy",
		"app":     h.cfg.AppName,
		"version": h.cfg.AppVersion,
		"env":     h.cfg.Environment,
		"checks":  checks,
	}
	if !healthy {
		payload["status"] = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, payload)
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *Handler) status(ctx context.Context, name string, err error) string {
	if err == nil {
		return statusOK
	}
	h.logger.ErrorContext(ctx, "Health check failed", "check", name, "error", err)
	return statusError
}
