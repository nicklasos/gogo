package health

import (
	"app/internal"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, app *internal.App) {
	handler := NewHandler(app.Queries, app.Cache, app.Config, app.Logger)

	r.Match([]string{"GET", "HEAD"}, "/health", handler.Check)
	app.Api.Match([]string{"GET", "HEAD"}, "/health", handler.Check)
}
