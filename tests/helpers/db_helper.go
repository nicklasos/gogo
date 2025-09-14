package helpers

import (
	"context"
	"testing"

	"app/internal/db"
	"app/tests"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// WithTransaction runs a test function within a database transaction that is rolled back after completion
// This ensures test isolation and prevents data pollution between tests
func WithTransaction(t *testing.T, fn func(ctx context.Context, tx pgx.Tx, queries *db.Queries)) {
	ctx := context.Background()
	pool := tests.GetTestDBPool()

	// Begin transaction
	tx, err := pool.Begin(ctx)
	require.NoError(t, err, "Failed to begin transaction")

	// Create queries instance with transaction
	queries := db.New(tx)

	// Ensure transaction is always rolled back
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			t.Logf("Failed to rollback transaction: %v", rollbackErr)
		}
	}()

	// Run the test function
	fn(ctx, tx, queries)
}

// WithTransactionQueries is a simplified version that only provides queries
func WithTransactionQueries(t *testing.T, fn func(ctx context.Context, queries *db.Queries)) {
	WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
		fn(ctx, queries)
	})
}
