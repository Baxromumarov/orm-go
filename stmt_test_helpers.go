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

	queryQuery string
	queryArgs  []any
	rows       pgx.Rows
	queryErr   error

	batchResults pgx.BatchResults

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

func (p *stubPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	p.lastCtx = ctx
	p.queryQuery = sql
	p.queryArgs = append([]any(nil), args...)
	if p.queryErr != nil {
		return nil, p.queryErr
	}
	if p.rows != nil {
		return p.rows, nil
	}
	return &stubRows{}, nil
}

func (p *stubPool) SendBatch(ctx context.Context, _ *pgx.Batch) pgx.BatchResults {
	p.lastCtx = ctx
	if p.batchResults != nil {
		return p.batchResults
	}
	return &stubBatchResults{}
}

type stubRows struct{}

func (r *stubRows) Close()                        {}
func (r *stubRows) Err() error                    { return nil }
func (r *stubRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *stubRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}
func (r *stubRows) Next() bool             { return false }
func (r *stubRows) Scan(_ ...any) error    { return nil }
func (r *stubRows) Values() ([]any, error) { return nil, nil }
func (r *stubRows) RawValues() [][]byte    { return nil }
func (r *stubRows) Conn() *pgx.Conn        { return nil }

type scriptedRows struct {
	values  [][]any
	idx     int
	err     error
	scanErr error
}

func newScriptedRows(values [][]any) *scriptedRows {
	return &scriptedRows{values: values, idx: -1}
}

func (r *scriptedRows) Close()                        {}
func (r *scriptedRows) Err() error                    { return r.err }
func (r *scriptedRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *scriptedRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}
func (r *scriptedRows) Next() bool {
	if r.idx+1 >= len(r.values) {
		return false
	}
	r.idx++
	return true
}
func (r *scriptedRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	if r.idx < 0 || r.idx >= len(r.values) {
		return fmt.Errorf("scan out of range")
	}
	return stubRow{values: r.values[r.idx]}.Scan(dest...)
}
func (r *scriptedRows) Values() ([]any, error) {
	if r.idx < 0 || r.idx >= len(r.values) {
		return nil, nil
	}
	return r.values[r.idx], nil
}
func (r *scriptedRows) RawValues() [][]byte { return nil }
func (r *scriptedRows) Conn() *pgx.Conn     { return nil }

type stubBatchResults struct{}

func (b *stubBatchResults) Exec() (pgconn.CommandTag, error) { return pgconn.CommandTag{}, nil }
func (b *stubBatchResults) Query() (pgx.Rows, error)         { return &stubRows{}, nil }
func (b *stubBatchResults) QueryRow() pgx.Row                { return stubRow{} }
func (b *stubBatchResults) Close() error                     { return nil }

type User struct {
	ID   int    `orm:"id"`
	Name string `orm:"name"`
	Age  int    `orm:"age"`
}
