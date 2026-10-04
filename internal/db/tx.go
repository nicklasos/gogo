package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// TxBeginner is satisfied by *pgxpool.Pool and by pgx.Tx. With a pgx.Tx the nested
// transaction is a savepoint, which is what lets tests run inside a rolled-back transaction.
type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type TxRunner struct {
	begin   TxBeginner
	queries *Queries
}

func NewTxRunner(begin TxBeginner, queries *Queries) *TxRunner {
	return &TxRunner{begin: begin, queries: queries}
}

// WithTx runs fn in a transaction: it commits when fn returns nil and rolls back otherwise.
func (r *TxRunner) WithTx(ctx context.Context, fn func(q *Queries) error) error {
	tx, err := r.begin.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(r.queries.WithTx(tx)); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
