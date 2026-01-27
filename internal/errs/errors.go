package errs

import (
	"errors"
	"net/http"
)

// Common application errors (deprecated - use DomainError constructors instead)
var (
	ErrNotFound   = errors.New("resource not found")
	ErrBadRequest = errors.New("bad request")
)

// IsNotFound checks if error is a not found error
func IsNotFound(err error) bool {
	domainErr := ExtractDomainError(err)
	return domainErr != nil && domainErr.Status == http.StatusNotFound
}

// IsBadRequest checks if error is a bad request error
func IsBadRequest(err error) bool {
	domainErr := ExtractDomainError(err)
	return domainErr != nil && domainErr.Status == http.StatusBadRequest
}
