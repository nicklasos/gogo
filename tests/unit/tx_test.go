package unit

import (
	"context"
	"errors"
	"testing"

	"app/internal/db"
	"app/tests/helpers"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTxRunner_WithTx(t *testing.T) {
	t.Run("should commit when the callback succeeds", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			runner := db.NewTxRunner(tx, queries)

			var created db.User
			err := runner.WithTx(ctx, func(q *db.Queries) error {
				var err error
				created, err = q.CreateUser(ctx, db.CreateUserParams{
					Email:    "tx-commit@example.com",
					Name:     "Tx Commit",
					Password: "hash",
					Roles:    []string{"user"},
				})
				return err
			})
			require.NoError(t, err)

			found, err := queries.GetUserByID(ctx, created.ID)
			require.NoError(t, err)
			assert.Equal(t, "tx-commit@example.com", found.Email)
		})
	})

	t.Run("should roll back when the callback fails", func(t *testing.T) {
		helpers.WithTransaction(t, func(ctx context.Context, tx pgx.Tx, queries *db.Queries) {
			runner := db.NewTxRunner(tx, queries)
			boom := errors.New("boom")

			err := runner.WithTx(ctx, func(q *db.Queries) error {
				_, err := q.CreateUser(ctx, db.CreateUserParams{
					Email:    "tx-rollback@example.com",
					Name:     "Tx Rollback",
					Password: "hash",
					Roles:    []string{"user"},
				})
				require.NoError(t, err)
				return boom
			})
			assert.ErrorIs(t, err, boom)

			_, err = queries.GetUserByEmail(ctx, "tx-rollback@example.com")
			assert.ErrorIs(t, err, pgx.ErrNoRows)
		})
	})
}
