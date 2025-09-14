package cities

import (
	"app/internal"
	"app/internal/db"
	apperrors "app/internal/errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *CitiesService
}

// City represents a city for swagger documentation
type City = db.City

// CitiesResponse represents the response structure for cities endpoint
type CitiesResponse struct {
	Data []City `json:"data"`
}

// ErrorResponse represents error response structure
type ErrorResponse struct {
	Error string `json:"error"`
}

func NewHandler(app *internal.App) *Handler {
	return &Handler{
		service: NewCitiesService(app.Queries),
	}
}

// ListCities returns all cities
// @Summary List all cities
// @Description Get a list of all cities
// @Tags cities
// @Accept json
// @Produce json
// @Success 200 {object} CitiesResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/cities [get]
func (h *Handler) ListCities(c *gin.Context) {
	cities, err := h.service.ListCities(c.Request.Context())
	if err != nil {
		if apperrors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Cities not found"})
			return
		}
		c.Error(err)
		return
	}

	// Ensure empty slice is returned as [] not null in JSON
	if cities == nil {
		cities = []City{}
	}

	c.JSON(http.StatusOK, CitiesResponse{Data: cities})
}
