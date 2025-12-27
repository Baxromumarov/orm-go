package orm_go

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestDeleteStmtAutoTableNameAndTable(t *testing.T) {
	scope := &ModelScope{input: &User{}}
	stmt := &DeleteStmt{scope: scope}

	stmt.AutoTableName()
	if scope.table != "users" {
		t.Fatalf("got table %q", scope.table)
	}

	stmt.Table("custom_table")
	if scope.table != "custom_table" {
		t.Fatalf("got table %q", scope.table)
	}
}

func TestDeleteStmtReturningCopies(t *testing.T) {
	stmt := &DeleteStmt{}
	cols := []string{"id", "name"}
	stmt.Returning(cols...)
	cols[0] = "changed"

	if stmt.returningCols[0] != "id" {
		t.Fatalf("expected copy of returning columns")
	}
}

func TestDeleteStmtBuildErrors(t *testing.T) {
	stmt := &DeleteStmt{}
	if _, _, err := stmt.build(); err == nil || !errors.Is(err, ErrNilScope) {
		t.Fatalf("unexpected error: %v", err)
	}

	stmt = &DeleteStmt{
		scope: &ModelScope{table: "users"},
	}
	if _, _, err := stmt.build(); err == nil || !errors.Is(err, ErrMissingWhereClause) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteStmtBuildWithReturning(t *testing.T) {
	stmt := &DeleteStmt{
		scope: &ModelScope{table: "users"},
		where: Eq("id", 1),
	}
	stmt.Returning("id", "name")

	got, args, err := stmt.build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "DELETE FROM users WHERE id = $1 RETURNING id, name"
	if got != want {
		t.Fatalf("query mismatch: got %q want %q", got, want)
	}
	if !reflect.DeepEqual(args, []any{1}) {
		t.Fatalf("args mismatch: %#v", args)
	}
}

func TestDeleteStmtExecNilScope(t *testing.T) {
	stmt := &DeleteStmt{}
	if err := stmt.Exec(); err == nil || !errors.Is(err, ErrNilScope) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteStmtExecNilPool(t *testing.T) {
	scope := &ModelScope{
		table: "users",
	}
	stmt := &DeleteStmt{scope: scope}
	stmt.Where(Eq("id", 1))

	if err := stmt.Exec(); err == nil || !errors.Is(err, ErrNilDB) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteStmtExecMissingTable(t *testing.T) {
	pool := &stubPool{}
	scope := &ModelScope{
		pool: pool,
	}
	stmt := &DeleteStmt{scope: scope}
	stmt.Where(Eq("id", 1))

	if err := stmt.Exec(); err == nil || !errors.Is(err, ErrMissingTable) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteStmtExecMissingWhere(t *testing.T) {
	pool := &stubPool{}
	scope := &ModelScope{
		pool:  pool,
		table: "users",
	}
	stmt := &DeleteStmt{scope: scope}

	if err := stmt.Exec(); err == nil || !errors.Is(err, ErrMissingWhereClause) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteStmtExecNoReturningSuccess(t *testing.T) {
	pool := &stubPool{}
	scope := &ModelScope{
		pool:  pool,
		table: "users",
	}
	stmt := &DeleteStmt{scope: scope}
	stmt.Where(Eq("id", 1))

	if err := stmt.Exec(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool.execQuery != "DELETE FROM users WHERE id = $1" {
		t.Fatalf("query mismatch: %q", pool.execQuery)
	}
	if !reflect.DeepEqual(pool.execArgs, []any{1}) {
		t.Fatalf("args mismatch: %#v", pool.execArgs)
	}
}

func TestDeleteStmtExecNoReturningError(t *testing.T) {
	execErr := errors.New("exec fail")
	pool := &stubPool{execErr: execErr}
	scope := &ModelScope{
		pool:  pool,
		table: "users",
	}
	stmt := &DeleteStmt{scope: scope}
	stmt.Where(Eq("id", 1))

	err := stmt.Exec()
	if err == nil || !errors.Is(err, execErr) || !strings.Contains(err.Error(), "error executing update") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteStmtExecReturningUnknownColumn(t *testing.T) {
	pool := &stubPool{}
	user := &User{Name: "bob"}
	scope := &ModelScope{
		pool:  pool,
		input: user,
		table: "users",
	}
	stmt := &DeleteStmt{scope: scope}
	stmt.Where(Eq("id", 1)).Returning("missing")

	if err := stmt.Exec(); err == nil || err.Error() != "unknown returning column: missing" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteStmtExecReturningFieldNotAddressable(t *testing.T) {
	pool := &stubPool{}
	user := User{Name: "bob"}
	scope := &ModelScope{
		pool:  pool,
		input: user,
		table: "users",
	}
	stmt := &DeleteStmt{scope: scope}
	stmt.Where(Eq("id", 1)).Returning("name")

	if err := stmt.Exec(); err == nil || err.Error() != "field name is not addressable" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteStmtExecReturningQueryRowError(t *testing.T) {
	scanErr := errors.New("scan fail")
	pool := &stubPool{row: stubRow{err: scanErr}}
	user := &User{Name: "bob"}
	scope := &ModelScope{
		pool:  pool,
		input: user,
		table: "users",
	}
	stmt := &DeleteStmt{scope: scope}
	stmt.Where(Eq("id", 1)).Returning("id")

	err := stmt.Exec()
	if err == nil || !errors.Is(err, scanErr) || !strings.Contains(err.Error(), "error executing update with returning") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteStmtExecReturningSuccess(t *testing.T) {
	pool := &stubPool{row: stubRow{values: []any{12, "alice"}}}
	user := &User{Name: "bob"}
	scope := &ModelScope{
		pool:  pool,
		input: user,
		table: "users",
	}
	stmt := &DeleteStmt{scope: scope}
	stmt.Where(Eq("id", 1)).Returning("id", "name")

	if err := stmt.Exec(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 12 || user.Name != "alice" {
		t.Fatalf("scan did not update fields: %#v", user)
	}
	if pool.queryRowQuery != "DELETE FROM users WHERE id = $1 RETURNING id, name" {
		t.Fatalf("query mismatch: %q", pool.queryRowQuery)
	}
	if !reflect.DeepEqual(pool.queryRowArgs, []any{1}) {
		t.Fatalf("args mismatch: %#v", pool.queryRowArgs)
	}
}
