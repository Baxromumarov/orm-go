package orm_go

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type stubRow struct {
	values []any
	err    error
}

func (r stubRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i, d := range dest {
		if i >= len(r.values) {
			return fmt.Errorf("missing value for scan")
		}
		rv := reflect.ValueOf(d)
		if rv.Kind() != reflect.Ptr || rv.IsNil() {
			return fmt.Errorf("scan destination must be pointer")
		}
		rv.Elem().Set(reflect.ValueOf(r.values[i]))
	}
	return nil
}

type stubPool struct {
	execQuery string
	execArgs  []any
	execErr   error

	queryRowQuery string
	queryRowArgs  []any
	row           pgx.Row

	lastCtx context.Context
}

func (p *stubPool) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	p.lastCtx = ctx
	p.execQuery = sql
	p.execArgs = append([]any(nil), args...)
	return pgconn.CommandTag{}, p.execErr
}

func (p *stubPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	p.lastCtx = ctx
	p.queryRowQuery = sql
	p.queryRowArgs = append([]any(nil), args...)
	if p.row != nil {
		return p.row
	}
	return stubRow{}
}

type User struct {
	ID   int    `orm:"id"`
	Name string `orm:"name"`
	Age  int    `orm:"age"`
}
