// Package factory creates database records with sensible defaults, for tests and for
// seeding a development database. Every factory takes options for the fields a caller
// cares about and fills in the rest:
//
//	user := factory.User(t, tx)
//	admin := factory.User(t, tx, factory.WithRoles("admin"), factory.WithEmail("boss@example.com"))
//	example := factory.Example(t, tx, user.ID, factory.WithTitle("Release notes"))
//
// A failure stops the caller through TB, so there is no error to check.
package factory

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
)

// DefaultPassword is the password of every user a factory creates, unless WithPassword is used.
const DefaultPassword = "password123"

// TB is the part of *testing.T the factories need. A seeder passes its own implementation.
type TB interface {
	Helper()
	Fatalf(format string, args ...any)
}

// DB is satisfied by a transaction (tests) and by a connection pool (seeding).
type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

var sequence atomic.Int64

// unique returns a value that does not repeat within a process or between runs.
func unique() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), sequence.Add(1))
}

var ctx = context.Background()
