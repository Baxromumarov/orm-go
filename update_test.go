package orm_go

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestUpdateStmtAutoTableNameAndTable(t *testing.T) {
	scope := &ModelScope{input: &User{}}
	stmt := &UpdateStmt{scope: scope}

	stmt.AutoTableName()
	if scope.table != "users" {
		t.Fatalf("got table %q", scope.table)
	}

	stmt.Table("custom_table")
	if scope.table != "custom_table" {
		t.Fatalf("got table %q", scope.table)
	}
}

func TestUpdateStmtReturningCopies(t *testing.T) {
	stmt := &UpdateStmt{}
	cols := []string{"id", "name"}
	stmt.Returning(cols...)
	cols[0] = "changed"

	if stmt.returningCols[0] != "id" {
		t.Fatalf("expected copy of returning columns")
	}
}

func TestUpdateStmtBuildErrors(t *testing.T) {
	stmt := &UpdateStmt{}
	if _, _, err := stmt.build(); err == nil || err.Error() != "model scope is nil" {
		t.Fatalf("unexpected error: %v", err)
	}

	stmt = &UpdateStmt{
		scope: &ModelScope{},
		where: Eq("id", 1),
	}
	if _, _, err := stmt.build(); err == nil || err.Error() != "table name is required" {
		t.Fatalf("unexpected error: %v", err)
	}

	stmt = &UpdateStmt{
		scope: &ModelScope{table: "users"},
	}
	if _, _, err := stmt.build(); err == nil || err.Error() != "where clause is required" {
		t.Fatalf("unexpected error: %v", err)
	}

	stmt = &UpdateStmt{
		scope: &ModelScope{table: "users"},
		where: Eq("id", 1),
	}
	if _, _, err := stmt.build(); err == nil || err.Error() != "no columns to update" {
		t.Fatalf("unexpected error: %v", err)
	}

	stmt = &UpdateStmt{
		scope: &ModelScope{table: "users", columns: []string{"name"}},
		where: Eq("id", 1),
	}
	if _, _, err := stmt.build(); err == nil || err.Error() != "columns and values length mismatch" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateStmtBuildWithSetsAndReturning(t *testing.T) {
	stmt := &UpdateStmt{
		scope: &ModelScope{table: "users"},
		where: And(Eq("id", 1), Eq("active", true)),
	}
	stmt.Set("name", "bob").Set("age", 30).Where(And(Eq("id", 1), Eq("active", true))).Returning("id", "name")

	got, args, err := stmt.build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "UPDATE users SET name = $1, age = $2 WHERE (id = $3 AND active = $4) RETURNING id, name"
	if got != want {
		t.Fatalf("query mismatch: got %q want %q", got, want)
	}
	if !reflect.DeepEqual(args, []any{"bob", 30, 1, true}) {
		t.Fatalf("args mismatch: %#v", args)
	}
}

func TestUpdateStmtBuildWithScopeColumns(t *testing.T) {
	stmt := &UpdateStmt{
		scope: &ModelScope{
			table:   "users",
			columns: []string{"name", "age"},
			values:  []any{"bob", 30},
		},
		where: Eq("id", 1),
	}

	got, args, err := stmt.build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "UPDATE users SET name = $1, age = $2 WHERE id = $3"
	if got != want {
		t.Fatalf("query mismatch: got %q want %q", got, want)
	}
	if !reflect.DeepEqual(args, []any{"bob", 30, 1}) {
		t.Fatalf("args mismatch: %#v", args)
	}
}

func TestUpdateStmtExecNilScope(t *testing.T) {
	stmt := &UpdateStmt{}
	if err := stmt.Exec(); err == nil || !errors.Is(err, ErrNilScope) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateStmtExecNilPool(t *testing.T) {
	scope := &ModelScope{
		table:   "users",
		columns: []string{"name"},
	}
	stmt := &UpdateStmt{scope: scope}

	if err := stmt.Exec(); err == nil || !errors.Is(err, ErrNilDB) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateStmtExecTableNameRequired(t *testing.T) {
	pool := &stubPool{}
	scope := &ModelScope{
		pool:    pool,
		input:   nil,
		columns: []string{"name"},
	}
	stmt := &UpdateStmt{scope: scope}
	stmt.Where(Eq("id", 1))

	if err := stmt.Exec(); err == nil || !errors.Is(err, ErrMissingTable) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateStmtExecNoColumns(t *testing.T) {
	pool := &stubPool{}
	scope := &ModelScope{
		pool:  pool,
		table: "users",
	}
	stmt := &UpdateStmt{scope: scope}
	stmt.Where(Eq("id", 1))

	err := stmt.Exec()
	if err == nil || !strings.Contains(err.Error(), "error building update query") || !strings.Contains(err.Error(), "no columns to update") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateStmtExecBuildError(t *testing.T) {
	pool := &stubPool{}
	scope := &ModelScope{
		pool:    pool,
		input:   &User{},
		table:   "users",
		columns: []string{"name"},
	}
	stmt := &UpdateStmt{scope: scope}

	err := stmt.Exec()
	if err == nil || !strings.Contains(err.Error(), "error building update query") || !strings.Contains(err.Error(), "where clause is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateStmtExecNoReturningSuccess(t *testing.T) {
	pool := &stubPool{}
	scope := &ModelScope{
		pool:    pool,
		input:   &User{},
		table:   "users",
		columns: []string{"name"},
	}
	stmt := &UpdateStmt{scope: scope}
	stmt.Set("name", "bob").Set("age", 30).Where(Eq("id", 1))

	if err := stmt.Exec(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pool.execQuery != "UPDATE users SET name = $1, age = $2 WHERE id = $3" {
		t.Fatalf("query mismatch: %q", pool.execQuery)
	}
	if !reflect.DeepEqual(pool.execArgs, []any{"bob", 30, 1}) {
		t.Fatalf("args mismatch: %#v", pool.execArgs)
	}
	if pool.lastCtx == nil {
		t.Fatalf("expected context to be set")
	}
}

func TestUpdateStmtExecAutoTableAndColumns(t *testing.T) {
	pool := &stubPool{}
	user := &User{Name: "bob", Age: 30}
	scope := &ModelScope{
		pool:  pool,
		input: user,
	}
	stmt := &UpdateStmt{scope: scope}
	stmt.Where(Eq("id", 1))

	if err := stmt.Exec(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scope.table != "users" {
		t.Fatalf("expected table to be set, got %q", scope.table)
	}
	if pool.execQuery != "UPDATE users SET name = $1, age = $2 WHERE id = $3" {
		t.Fatalf("query mismatch: %q", pool.execQuery)
	}
	if !reflect.DeepEqual(pool.execArgs, []any{"bob", 30, 1}) {
		t.Fatalf("args mismatch: %#v", pool.execArgs)
	}
}

func TestUpdateStmtExecNoReturningError(t *testing.T) {
	execErr := errors.New("exec fail")
	pool := &stubPool{execErr: execErr}
	scope := &ModelScope{
		pool:    pool,
		input:   &User{},
		table:   "users",
		columns: []string{"name"},
	}
	stmt := &UpdateStmt{scope: scope}
	stmt.Set("name", "bob").Where(Eq("id", 1))

	err := stmt.Exec()
	if err == nil || !errors.Is(err, execErr) || !strings.Contains(err.Error(), "error executing update") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateStmtExecReturningUnknownColumn(t *testing.T) {
	pool := &stubPool{}
	user := &User{Name: "bob"}
	scope := &ModelScope{
		pool:    pool,
		input:   user,
		table:   "users",
		columns: []string{"name"},
	}
	stmt := &UpdateStmt{scope: scope}
	stmt.Set("name", "bob").Where(Eq("id", 1)).Returning("missing")

	if err := stmt.Exec(); err == nil || err.Error() != "unknown returning column: missing" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateStmtExecReturningFieldNotAddressable(t *testing.T) {
	pool := &stubPool{}
	user := User{Name: "bob"}
	scope := &ModelScope{
		pool:    pool,
		input:   user,
		table:   "users",
		columns: []string{"name"},
	}
	stmt := &UpdateStmt{scope: scope}
	stmt.Set("name", "bob").Where(Eq("id", 1)).Returning("name")

	if err := stmt.Exec(); err == nil || err.Error() != "field name is not addressable" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateStmtExecReturningQueryRowError(t *testing.T) {
	scanErr := errors.New("scan fail")
	pool := &stubPool{row: stubRow{err: scanErr}}
	user := &User{Name: "bob"}
	scope := &ModelScope{
		pool:    pool,
		input:   user,
		table:   "users",
		columns: []string{"name"},
	}
	stmt := &UpdateStmt{scope: scope}
	stmt.Set("name", "bob").Where(Eq("id", 1)).Returning("id")

	err := stmt.Exec()
	if err == nil || !errors.Is(err, scanErr) || !strings.Contains(err.Error(), "error executing update with returning") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateStmtExecReturningSuccess(t *testing.T) {
	pool := &stubPool{row: stubRow{values: []any{12, "alice"}}}
	user := &User{Name: "bob"}
	scope := &ModelScope{
		pool:    pool,
		input:   user,
		table:   "users",
		columns: []string{"name"},
	}
	stmt := &UpdateStmt{scope: scope}
	stmt.Set("name", "bob").Where(Eq("id", 1)).Returning("id", "name")

	if err := stmt.Exec(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 12 || user.Name != "alice" {
		t.Fatalf("scan did not update fields: %#v", user)
	}
	if pool.queryRowQuery != "UPDATE users SET name = $1 WHERE id = $2 RETURNING id, name" {
		t.Fatalf("query mismatch: %q", pool.queryRowQuery)
	}
	if !reflect.DeepEqual(pool.queryRowArgs, []any{"bob", 1}) {
		t.Fatalf("args mismatch: %#v", pool.queryRowArgs)
	}
}
