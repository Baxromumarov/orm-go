package orm_go

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Tx wraps a pgx transaction to execute ORM statements within it.
type Tx struct {
	tx pgx.Tx
}

// Exec executes a statement within the transaction.
func (t *Tx) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return t.tx.Exec(ctx, sql, args...)
}

// QueryRow queries a single row within the transaction.
func (t *Tx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

// Query executes a query within the transaction.
func (t *Tx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}

// SendBatch sends a batch within the transaction.
func (t *Tx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return t.tx.SendBatch(ctx, b)
}

// Tx runs fn inside a database transaction.
func (db *DB) Tx(ctx context.Context, fn func(tx *Tx) error) error {
	if db == nil || db.Config == nil || db.conn == nil {
		return ErrNilDB
	}
	return withTx(ctx, db.conn.Begin, fn)
}

// Model creates a scope that uses this transaction for queries.
func (tx *Tx) Model(model any) *ModelScope {
	return &ModelScope{
		pool:  tx,
		input: model,
	}
}

func withTx(
	ctx context.Context,
	begin func(context.Context) (pgx.Tx, error),
	fn func(tx *Tx) error,
) error {
	if ctx == nil {
		ctx = context.Background()
	}

	pgxTx, err := begin(ctx)
	if err != nil {
		return err
	}

	tx := &Tx{tx: pgxTx}

	defer func() {
		if p := recover(); p != nil {
			_ = pgxTx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = pgxTx.Rollback(ctx)
		return err
	}

	return pgxTx.Commit(ctx)
}
