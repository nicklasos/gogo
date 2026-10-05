package errs

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	ErrorKey  string                 `json:"error_key"`
	Message   string                 `json:"message,omitempty" validate:"optional"`
	Status    int                    `json:"status"`
	Details   map[string]interface{} `json:"details,omitempty" validate:"optional"`
	Timestamp string                 `json:"timestamp,omitempty" validate:"optional"`
}

// RespondWithError sends a structured error response
func RespondWithError(c *gin.Context, err error) {
	domainErr := ExtractDomainError(err)

	response := ErrorResponse{
		ErrorKey:  domainErr.Key,
		Message:   domainErr.Message,
		Status:    domainErr.Status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if len(domainErr.Details) > 0 {
		response.Details = domainErr.Details
	}

	// A 5xx is recorded on the context, where the ErrorHandler middleware logs it once
	// with the request ID. Expected failures (4xx) are not logged: they are the client's.
	if domainErr.Status >= http.StatusInternalServerError {
		_ = c.Error(err)
	}

	c.JSON(domainErr.Status, response)
}

// RespondWithUnauthorized sends an unauthorized error response
func RespondWithUnauthorized(c *gin.Context, message string) {
	RespondWithError(c, NewUnauthorizedError(ErrKeyUnauthorized, message))
}

// RespondWithBadRequest sends a bad request error response
func RespondWithBadRequest(c *gin.Context, key, message string) {
	RespondWithError(c, NewBadRequestError(key, message))
}

// RespondWithInternalError sends an internal server error response
func RespondWithInternalError(c *gin.Context, message string) {
	RespondWithError(c, NewInternalError(ErrKeyInternalError, message))
}
