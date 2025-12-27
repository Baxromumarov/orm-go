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

func (db *DB) Model(model any) *ModelScope {
	cols, vals := ParseInsertColumns(model)

	var pool pooler
	if db.conn != nil {
		pool = db.conn
	}

	return &ModelScope{
		pool:    pool,
		input:   model,
		columns: cols,
		values:  vals,
	}
}

func (m *ModelScope) Insert(ctx context.Context) *InsertStmt {
	return &InsertStmt{scope: m, ctx: ctx}
}
func (m *ModelScope) Update(ctx context.Context) *UpdateStmt {
	return &UpdateStmt{scope: m, ctx: ctx}
}
func (m *ModelScope) Delete(ctx context.Context) *DeleteStmt {
	return &DeleteStmt{scope: m, ctx: ctx}
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
