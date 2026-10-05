package errs

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// WrapDatabaseError wraps common database errors into DomainErrors.
// pgx.ErrNoRows becomes a 404; other errors become 500 with full error logged.
func WrapDatabaseError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return NewNotFoundError(ErrKeyNotFound, "resource not found")
	}
	return WrapInternal(ErrKeyInternalError, "database error", err)
}

// IsNotFound checks if error is a not found error (status 404)
func IsNotFound(err error) bool {
	domainErr := ExtractDomainError(err)
	return domainErr != nil && domainErr.Status == http.StatusNotFound
}
