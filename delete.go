package orm_go

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

// DeleteBuilder is Stage 1: Must set table first.
type DeleteBuilder interface {
	Table(name string) DeleteBuilderWithTable
	AutoTableName() DeleteBuilderWithTable
}

// DeleteBuilderWithTable is Stage 2: Table is set, MUST set WHERE clause next.
// WHERE is required to prevent accidental full-table deletions.
type DeleteBuilderWithTable interface {
	Where(expr Expr) DeleteBuilderReady
}

// DeleteBuilderReady is Stage 3: WHERE is set, can add options or execute.
type DeleteBuilderReady interface {
	Returning(cols ...string) DeleteBuilderReady
	Exec() error
}

// deleteBuilder is the unexported concrete builder implementing all delete stages.
type deleteBuilder struct {
	scope         *ModelScope
	ctx           context.Context
	where         Expr
	returningCols []string
}

// Ensure deleteBuilder implements all interfaces at compile time.
var (
	_ DeleteBuilder          = (*deleteBuilder)(nil)
	_ DeleteBuilderWithTable = (*deleteBuilder)(nil)
	_ DeleteBuilderReady     = (*deleteBuilder)(nil)
)

// DeleteStmt is a type alias for backward compatibility.
// Deprecated: Use DeleteBuilder/DeleteBuilderWithTable/DeleteBuilderReady interfaces instead.
type DeleteStmt = deleteBuilder

// AutoTableName sets the table name based on the model type.
// Implements DeleteBuilder interface, returns DeleteBuilderWithTable.
func (ds *deleteBuilder) AutoTableName() DeleteBuilderWithTable {
	if ds.scope != nil {
		ds.scope.table = ParseTableName(ds.scope.input)
	}
	return ds
}

// Table sets the table name explicitly for this DELETE.
// Implements DeleteBuilder interface, returns DeleteBuilderWithTable.
func (ds *deleteBuilder) Table(tableName string) DeleteBuilderWithTable {
	if ds.scope != nil {
		ds.scope.table = tableName
	}

	return ds
}

// Where sets the WHERE clause for the DELETE.
// Implements DeleteBuilderWithTable interface, returns DeleteBuilderReady.
func (ds *deleteBuilder) Where(expr Expr) DeleteBuilderReady {
	ds.where = expr
	return ds
}

// Returning adds a RETURNING clause and returns the statement for chaining.
// Implements DeleteBuilderReady interface.
func (ds *deleteBuilder) Returning(cols ...string) DeleteBuilderReady {
	ds.returningCols = append([]string(nil), cols...)
	return ds
}

// Exec builds and executes the DELETE statement.
// Implements DeleteBuilderReady interface.
func (ds *deleteBuilder) Exec() error {
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

	meta := GetModelMeta(ds.scope.input)

	v := reflect.ValueOf(ds.scope.input)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	scanArgs, err := ScanArgsForReturning(v, ds.returningCols, meta)
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

func (ds *deleteBuilder) build() (string, []any, error) {

	if ds.scope == nil {
		return "", nil, ErrNilScope
	}
	if ds.where == nil {
		return "", nil, ErrMissingWhereClause
	}

	sb := &sqlBuilder{}
	sb.sql.WriteString("DELETE FROM ")
	sb.sql.WriteString(ds.scope.table)
	sb.sql.WriteString(" WHERE ")

	ds.where.build(sb)
	sb.WriteReturning(ds.returningCols)

	return sb.sql.String(), sb.args, nil
}
