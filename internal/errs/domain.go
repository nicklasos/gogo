package errs

import (
	"errors"
	"net/http"
)

// DomainError represents a structured application error with an error key
type DomainError struct {
	Key     string                 // Error key (e.g., "examples.not_found")
	Message string                 // Human-readable message (for logging)
	Status  int                    // HTTP status code
	Err     error                  // Wrapped error (for error chain)
	Details map[string]interface{} // Additional context
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Key
}

// Unwrap returns the wrapped error for error chain support
func (e *DomainError) Unwrap() error {
	return e.Err
}

// WithDetails adds additional context to the error
func (e *DomainError) WithDetails(details map[string]interface{}) *DomainError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	for k, v := range details {
		e.Details[k] = v
	}
	return e
}

// NewDomainError creates a new domain error
func NewDomainError(key, message string, status int) *DomainError {
	return &DomainError{
		Key:     key,
		Message: message,
		Status:  status,
	}
}

// WrapDomainError wraps an existing error as a domain error
func WrapDomainError(key, message string, status int, err error) *DomainError {
	return &DomainError{
		Key:     key,
		Message: message,
		Status:  status,
		Err:     err,
	}
}

// ExtractDomainError extracts a DomainError from an error chain
func ExtractDomainError(err error) *DomainError {
	if err == nil {
		return nil
	}

	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr
	}

	// If not a domain error, wrap it as an internal error
	// Don't expose internal error details to users
	return &DomainError{
		Key:     ErrKeyInternalError,
		Message: "Something went wrong",
		Status:  http.StatusInternalServerError,
		Err:     err,
	}
}

// IsDomainError checks if an error is a DomainError
func IsDomainError(err error) bool {
	var domainErr *DomainError
	return errors.As(err, &domainErr)
}

// Common domain error constructors

// NewNotFoundError creates a not found error
func NewNotFoundError(key, message string) *DomainError {
	return NewDomainError(key, message, http.StatusNotFound)
}

// NewBadRequestError creates a bad request error
func NewBadRequestError(key, message string) *DomainError {
	return NewDomainError(key, message, http.StatusBadRequest)
}

// NewUnauthorizedError creates an unauthorized error
func NewUnauthorizedError(key, message string) *DomainError {
	return NewDomainError(key, message, http.StatusUnauthorized)
}

// NewForbiddenError creates a forbidden error
func NewForbiddenError(key, message string) *DomainError {
	return NewDomainError(key, message, http.StatusForbidden)
}

// NewInternalError creates an internal server error
func NewInternalError(key, message string) *DomainError {
	return NewDomainError(key, message, http.StatusInternalServerError)
}

// WrapNotFound wraps an error as a not found error
func WrapNotFound(key, message string, err error) *DomainError {
	return WrapDomainError(key, message, http.StatusNotFound, err)
}

// WrapBadRequest wraps an error as a bad request error
func WrapBadRequest(key, message string, err error) *DomainError {
	return WrapDomainError(key, message, http.StatusBadRequest, err)
}

// WrapInternal wraps an error as an internal server error
func WrapInternal(key, message string, err error) *DomainError {
	return WrapDomainError(key, message, http.StatusInternalServerError, err)
}
