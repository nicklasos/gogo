// Package factory creates database records with sensible defaults, for tests and for
// seeding a development database. Every factory takes options for the fields a caller
// cares about and fills in the rest:
//
//	create := factory.New(t, tx)
//	user := create.User()
//	admin := create.User(factory.WithRoles("admin"), factory.WithEmail("boss@example.com"))
//	example := create.Example(user.ID, factory.WithTitle("Release notes"))
//
// In a test the factory is already there as app.Create (see tests/testkit).
// A failure stops the caller through TB, so there is no error to check.
package factory

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"app/internal/db"
)

// DefaultPassword is the password of every user a factory creates, unless WithPassword is used.
const DefaultPassword = "password123"

// TB is the part of *testing.T the factories need. A seeder passes its own implementation.
type TB interface {
	Helper()
	Fatalf(format string, args ...any)
}

// Factory creates records on one connection and reports failures to one TB.
type Factory struct {
	t       TB
	queries *db.Queries
}

// New takes a transaction (tests) or a connection pool (seeding).
func New(t TB, conn db.DBTX) *Factory {
	return &Factory{t: t, queries: db.New(conn)}
}

var sequence atomic.Int64

// unique returns a value that does not repeat within a process or between runs.
func unique() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), sequence.Add(1))
}

var ctx = context.Background()
