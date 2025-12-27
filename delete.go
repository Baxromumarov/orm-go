package orm_go

import (
	"context"
	"errors"
	"fmt"
	"reflect"
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

func (ds *DeleteStmt) AutoTableName() *DeleteStmt {
	if ds.scope != nil {
		ds.scope.table = ParseTableName(ds.scope.input)
	}
	return ds
}

func (ds *DeleteStmt) Table(tableName string) *DeleteStmt {
	if ds.scope != nil {
		ds.scope.table = tableName
	}

	return ds
}

func (ds *DeleteStmt) Where(expr Expr) *DeleteStmt {
	ds.where = expr
	return ds
}

func (ds *DeleteStmt) Returning(cols ...string) *DeleteStmt {
	ds.returningCols = append([]string(nil), cols...)
	return ds
}

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
	fmt.Println("FINAL DELETE QUERY:", query)
	fmt.Println("ARGS:", args)

	if len(ds.returningCols) == 0 {
		_, err = ds.scope.pool.Exec(ds.ctx, query, args...)
		if err != nil {
			return fmt.Errorf("error executing update: %w", err)
		}
		return nil
	}

	meta := buildModelMeta(ds.scope.input)

	v := reflect.ValueOf(ds.scope.input)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	scanArgs := make([]any, 0, len(ds.returningCols))

	for _, col := range ds.returningCols {
		fm, ok := meta.byColumn[col]
		if !ok {
			return fmt.Errorf("unknown returning column: %s", col)
		}

		field := v.Field(fm.index)
		if !field.CanAddr() {
			return fmt.Errorf("field %s is not addressable", col)
		}

		scanArgs = append(scanArgs, field.Addr().Interface())
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
	sb.sql.WriteString(" WHERE ")

	ds.where.build(sb)

	if len(ds.returningCols) > 0 {
		sb.sql.WriteString(" RETURNING ")
		for i, ret := range ds.returningCols {
			if i > 0 {
				sb.sql.WriteString(", ")
			}
			sb.sql.WriteString(ret)
		}
	}

	return sb.sql.String(), sb.args, nil
}
