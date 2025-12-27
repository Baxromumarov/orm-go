package orm_go

import (
	"context"
	"errors"
	"fmt"
)

type DeleteStmt struct {
	scope         *ModelScope
	ctx           context.Context
	where         Expr
	returningCols []string
}

type Delete interface {
	AutoTableName() *DeleteStmt
	Table(tableName string) *DeleteStmt
	Where(expr Expr) *DeleteStmt
	Returning(cols ...string) *DeleteStmt
	Exec() error
	build() (string, []any, error)
}

var _ Delete = (*DeleteStmt)(nil)

// AutoTableName sets the table name based on the model type.
func (ds *DeleteStmt) AutoTableName() *DeleteStmt {
	setAutoTableName(ds.scope)
	return ds
}

// Table sets the table name explicitly for this DELETE.
func (ds *DeleteStmt) Table(tableName string) *DeleteStmt {
	setTableName(ds.scope, tableName)

	return ds
}

// Where sets the WHERE clause for the DELETE.
func (ds *DeleteStmt) Where(expr Expr) *DeleteStmt {
	ds.where = expr
	return ds
}

// Returning adds a RETURNING clause and returns the statement for chaining.
func (ds *DeleteStmt) Returning(cols ...string) *DeleteStmt {
	ds.returningCols = cloneStrings(cols)
	return ds
}

// Exec builds and executes the DELETE statement.
func (ds *DeleteStmt) Exec() error {
	if ds.ctx == nil {
		ds.ctx = context.Background()
	}

	if err := ds.scope.validate(); err != nil {
		return err
	}

	query, args, err := ds.build()
	if err != nil {
		return err
	}

	if len(ds.returningCols) == 0 {
		_, err = ds.scope.pool.Exec(ds.ctx, query, args...)
		if err != nil {
			return fmt.Errorf("error executing update: %w", err)
		}
		return nil
	}

	scanArgs, err := scanArgsForReturning(ds.scope.input, ds.returningCols)
	if err != nil {
		return err
	}

	if err := ds.scope.pool.QueryRow(
		ds.ctx,
		query,
		args...,
	).Scan(
		scanArgs...,
	); err != nil {
		return fmt.Errorf("error executing update with returning: %w", err)
	}

	return nil

}

var (
	ErrMissingWhereClause = errors.New("orm: DELETE statements must have a WHERE clause to prevent accidental full-table deletions")
)

func (ds *DeleteStmt) build() (string, []any, error) {

	if ds.scope == nil {
		return "", nil, ErrNilScope
	}
	if ds.where == nil {
		return "", nil, ErrMissingWhereClause
	}

	sb := &sqlBuilder{}
	sb.sql.WriteString("DELETE FROM ")
	sb.sql.WriteString(ds.scope.table)
	appendWhereClause(sb, ds.where)
	writeReturning(&sb.sql, ds.returningCols)

	return sb.sql.String(), sb.args, nil
}
