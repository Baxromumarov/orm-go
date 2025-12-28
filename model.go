package orm_go

import (
	"context"
	"errors"
)

type ModelScope struct {
	table   string
	columns []string
	values  []any
	pool    pooler
	input   any
}

// Model creates a scope for building statements around a model value.
func (db *DB) Model(model any) *ModelScope {
	var pool pooler
	if db.conn != nil {
		pool = db.conn
	}

	return &ModelScope{
		pool:  pool,
		input: model,
	}
}

// Insert starts an INSERT statement for this model.
// Returns InsertBuilder (Stage 1) - must set table before executing.
func (m *ModelScope) Insert(ctx context.Context) InsertBuilder {
	return &insertBuilder{scope: m, ctx: ctx}
}

// Update starts an UPDATE statement for this model.
// Returns UpdateBuilder (Stage 1) - must set table, then values, then WHERE before executing.
func (m *ModelScope) Update(ctx context.Context) UpdateBuilder {
	return &updateBuilder{scope: m, ctx: ctx}
}

// Delete starts a DELETE statement for this model.
// Returns DeleteBuilder (Stage 1) - must set table, then WHERE before executing.
func (m *ModelScope) Delete(ctx context.Context) DeleteBuilder {
	return &deleteBuilder{scope: m, ctx: ctx}
}

// Select starts a SELECT statement for this model.
// Returns SelectBuilder (Stage 1) - must set table before executing.
func (m *ModelScope) Select(ctx context.Context) SelectBuilder {
	return &selectBuilder{scope: m, ctx: ctx}
}

var (
	ErrNilScope     = errors.New("orm: model scope is nil")
	ErrNilDB        = errors.New("orm: db connection is nil")
	ErrMissingTable = errors.New("orm: table name is required")
	ErrNoInsertCols = errors.New("orm: no columns to insert")
)

func (m *ModelScope) validate() error {
	if m == nil {
		return ErrNilScope
	}
	if m.pool == nil {
		return ErrNilDB
	}
	if m.table == "" {
		return ErrMissingTable
	}
	return nil
}
