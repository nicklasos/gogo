package errs

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

const postgresUniqueViolation = "23505"

// DomainErrorFromPostgresUniqueViolation maps known unique constraints to client-facing errors.
func DomainErrorFromPostgresUniqueViolation(err error) *DomainError {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != postgresUniqueViolation {
		return nil
	}
	if isUsersEmailUniqueConstraint(strings.ToLower(pgErr.ConstraintName)) {
		return NewBadRequestError(ErrKeyAuthUserExists, "User with this email already exists")
	}
	return nil
}

func isUsersEmailUniqueConstraint(name string) bool {
	if name == "users_email_key" || name == "idx_users_email_lower" {
		return true
	}
	return strings.Contains(name, "users") && strings.Contains(name, "email")
}
