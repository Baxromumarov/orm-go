package orm_go

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ModelScope struct {
	table   string
	columns []string
	values  []any
	pool    *pgxpool.Pool
	input   any
}

func (db *DB) Model(model any) *ModelScope {
	cols, vals := ParseInsertColumns(model)
	return &ModelScope{
		pool:    db.conn,
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
