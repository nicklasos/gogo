package health

import (
	"net/http"

	"app/config"
	"app/internal/db"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	queries *db.Queries
	cfg     *config.Config
}

func NewHandler(queries *db.Queries, cfg *config.Config) *Handler {
	return &Handler{queries: queries, cfg: cfg}
}

func (h *Handler) Check(c *gin.Context) {
	payload := gin.H{
		"app":     h.cfg.AppName,
		"version": h.cfg.AppVersion,
		"env":     h.cfg.Environment,
	}

	if err := h.queries.Healthcheck(c.Request.Context()); err != nil {
		payload["status"] = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, payload)
		return
	}

	payload["status"] = "healthy"
	c.JSON(http.StatusOK, payload)
}
