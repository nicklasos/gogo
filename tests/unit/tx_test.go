package unit

import (
	"errors"
	"testing"

	"app/internal/db"
	"app/tests/testkit"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTxRunner_WithTx(t *testing.T) {
	t.Run("should commit when the callback succeeds", func(t *testing.T) {
		app := testkit.New(t)
		runner := app.TxRunner

		var created db.User
		err := runner.WithTx(app.Ctx, func(q *db.Queries) error {
			var err error
			created, err = q.CreateUser(app.Ctx, db.CreateUserParams{
				Email:    "tx-commit@example.com",
				Name:     "Tx Commit",
				Password: "hash",
				Roles:    []string{"user"},
			})
			return err
		})
		require.NoError(t, err)

		found, err := app.Queries.GetUserByID(app.Ctx, created.ID)
		require.NoError(t, err)
		assert.Equal(t, "tx-commit@example.com", found.Email)
	})

	t.Run("should roll back when the callback fails", func(t *testing.T) {
		app := testkit.New(t)
		runner := app.TxRunner
		boom := errors.New("boom")

		err := runner.WithTx(app.Ctx, func(q *db.Queries) error {
			_, err := q.CreateUser(app.Ctx, db.CreateUserParams{
				Email:    "tx-rollback@example.com",
				Name:     "Tx Rollback",
				Password: "hash",
				Roles:    []string{"user"},
			})
			require.NoError(t, err)
			return boom
		})
		assert.ErrorIs(t, err, boom)

		_, err = app.Queries.GetUserByEmail(app.Ctx, "tx-rollback@example.com")
		assert.ErrorIs(t, err, pgx.ErrNoRows)
	})
}
