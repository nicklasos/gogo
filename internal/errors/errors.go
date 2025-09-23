package errors

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Common application errors
var (
	ErrNotFound   = errors.New("resource not found")
	ErrBadRequest = errors.New("bad request")
)

// WrapDatabaseError wraps common database errors
func WrapDatabaseError(err error) error {
	if err == nil {
		return nil
	}

	// Check for pgx no rows error
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}

	return fmt.Errorf("database error: %w", err)
}

// WrapNotFound wraps not found errors with custom message
func WrapNotFound(message string) error {
	return fmt.Errorf("%s: %w", message, ErrNotFound)
}

// WrapInternal wraps internal errors with custom message
func WrapInternal(message string, err error) error {
	return fmt.Errorf("%s: %w", message, err)
}

// WrapBadRequest wraps bad request errors with custom message
func WrapBadRequest(message string, err error) error {
	return fmt.Errorf("%s: %w", message, ErrBadRequest)
}

// IsNotFound checks if error is ErrNotFound
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsBadRequest checks if error is ErrBadRequest
func IsBadRequest(err error) bool {
	return errors.Is(err, ErrBadRequest)
}
