package errs

import (
	"time"

	"github.com/gin-gonic/gin"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	ErrorKey  string                 `json:"error_key"`
	Message   string                 `json:"message,omitempty"`
	Status    int                    `json:"status"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp string                 `json:"timestamp,omitempty"`
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

	// Set status code
	c.JSON(domainErr.Status, response)
}

// RespondWithErrorAndStatus sends a structured error response with explicit status
func RespondWithErrorAndStatus(c *gin.Context, err error, status int) {
	domainErr := ExtractDomainError(err)

	response := ErrorResponse{
		ErrorKey:  domainErr.Key,
		Message:   domainErr.Message,
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if len(domainErr.Details) > 0 {
		response.Details = domainErr.Details
	}

	c.JSON(status, response)
}

// RespondWithUnauthorized sends an unauthorized error response
func RespondWithUnauthorized(c *gin.Context, message string) {
	RespondWithError(c, NewUnauthorizedError(ErrKeyUnauthorized, message))
}

// RespondWithForbidden sends a forbidden error response
func RespondWithForbidden(c *gin.Context, message string) {
	RespondWithError(c, NewForbiddenError(ErrKeyForbidden, message))
}

// RespondWithNotFound sends a not found error response
func RespondWithNotFound(c *gin.Context, key, message string) {
	RespondWithError(c, NewNotFoundError(key, message))
}

// RespondWithBadRequest sends a bad request error response
func RespondWithBadRequest(c *gin.Context, key, message string) {
	RespondWithError(c, NewBadRequestError(key, message))
}

// RespondWithInternalError sends an internal server error response
func RespondWithInternalError(c *gin.Context, message string) {
	RespondWithError(c, NewInternalError(ErrKeyInternalError, message))
}
