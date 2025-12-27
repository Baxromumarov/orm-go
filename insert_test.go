package orm_go

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestInsertStmtAutoTableNameAndTable(t *testing.T) {
	scope := &ModelScope{input: &User{}}
	stmt := &InsertStmt{scope: scope}

	stmt.AutoTableName()
	if scope.table != "users" {
		t.Fatalf("got table %q", scope.table)
	}

	stmt.Table("custom_table")
	if scope.table != "custom_table" {
		t.Fatalf("got table %q", scope.table)
	}
}

func TestInsertStmtReturningCopies(t *testing.T) {
	stmt := &InsertStmt{}
	cols := []string{"id", "name"}
	stmt.Returning(cols...)
	cols[0] = "changed"

	if stmt.returningCols[0] != "id" {
		t.Fatalf("expected copy of returning columns")
	}
}

func TestInsertStmtBuild(t *testing.T) {
	stmt := &InsertStmt{}
	if _, err := stmt.build(); err == nil || err.Error() != "model scope is nil" {
		t.Fatalf("unexpected error: %v", err)
	}

	scope := &ModelScope{table: "users", columns: []string{"name", "age"}}
	stmt = &InsertStmt{
		scope:         scope,
		returningCols: []string{"id", "name"},
	}
	got, err := stmt.build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "INSERT INTO users (name, age) VALUES ($1, $2) RETURNING id, name"
	if got != want {
		t.Fatalf("query mismatch: got %q want %q", got, want)
	}
}

func TestBuildModelMeta(t *testing.T) {
	type MetaUser struct {
		ID      int    `orm:"id"`
		Name    string `orm:"name"`
		Ignored string
		Empty   string `orm:""`
	}

	meta := buildModelMeta(&MetaUser{})
	if _, ok := meta.byColumn["id"]; !ok {
		t.Fatalf("missing id column")
	}
	if _, ok := meta.byColumn["name"]; !ok {
		t.Fatalf("missing name column")
	}
	if _, ok := meta.byColumn["ignored"]; ok {
		t.Fatalf("unexpected ignored column")
	}

	metaValue := buildModelMeta(MetaUser{})
	if _, ok := metaValue.byColumn["id"]; !ok {
		t.Fatalf("missing id column for value")
	}
}

func TestInsertStmtExecNilScope(t *testing.T) {
	stmt := &InsertStmt{}
	if err := stmt.Exec(); err == nil || !errors.Is(err, ErrNilScope) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInsertStmtExecNilPool(t *testing.T) {
	scope := &ModelScope{
		table:   "users",
		columns: []string{"name"},
	}
	stmt := &InsertStmt{scope: scope}

	if err := stmt.Exec(); err == nil || !errors.Is(err, ErrNilDB) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInsertStmtExecTableNameRequired(t *testing.T) {
	pool := &stubPool{}
	scope := &ModelScope{
		pool:    pool,
		input:   nil,
		table:   "",
		columns: []string{"name"},
		values:  []any{"bob"},
	}
	stmt := &InsertStmt{scope: scope}

	if err := stmt.Exec(); err == nil || !errors.Is(err, ErrMissingTable) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInsertStmtExecNoColumns(t *testing.T) {
	pool := &stubPool{}
	scope := &ModelScope{
		pool:  pool,
		input: &User{},
		table: "users",
	}
	stmt := &InsertStmt{scope: scope}

	if err := stmt.Exec(); err == nil || !errors.Is(err, ErrNoInsertCols) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInsertStmtExecNoReturningSuccess(t *testing.T) {
	pool := &stubPool{}
	scope := &ModelScope{
		pool:    pool,
		table:   "users",
		columns: []string{"name", "age"},
		values:  []any{"bob", 30},
	}
	stmt := &InsertStmt{scope: scope}

	if err := stmt.Exec(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pool.execQuery != "INSERT INTO users (name, age) VALUES ($1, $2)" {
		t.Fatalf("exec query mismatch: %q", pool.execQuery)
	}
	if !reflect.DeepEqual(pool.execArgs, []any{"bob", 30}) {
		t.Fatalf("exec args mismatch: %#v", pool.execArgs)
	}
	if pool.lastCtx == nil {
		t.Fatalf("expected context to be set")
	}
}

func TestInsertStmtExecAutoTableAndColumns(t *testing.T) {
	pool := &stubPool{}
	user := &User{Name: "bob", Age: 30}
	scope := &ModelScope{
		pool:  pool,
		input: user,
	}
	stmt := &InsertStmt{scope: scope}

	if err := stmt.Exec(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if scope.table != "users" {
		t.Fatalf("expected table to be set, got %q", scope.table)
	}
	if pool.execQuery != "INSERT INTO users (name, age) VALUES ($1, $2)" {
		t.Fatalf("exec query mismatch: %q", pool.execQuery)
	}
	if !reflect.DeepEqual(pool.execArgs, []any{"bob", 30}) {
		t.Fatalf("exec args mismatch: %#v", pool.execArgs)
	}
}

func TestInsertStmtExecNoReturningError(t *testing.T) {
	execErr := errors.New("exec fail")
	pool := &stubPool{execErr: execErr}
	scope := &ModelScope{
		pool:    pool,
		table:   "users",
		columns: []string{"name"},
		values:  []any{"bob"},
	}
	stmt := &InsertStmt{scope: scope}

	err := stmt.Exec()
	if err == nil || !errors.Is(err, execErr) || !strings.Contains(err.Error(), "error executing query") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInsertStmtExecReturningUnknownColumn(t *testing.T) {
	pool := &stubPool{}
	user := &User{Name: "bob"}
	scope := &ModelScope{
		pool:    pool,
		input:   user,
		table:   "users",
		columns: []string{"name"},
		values:  []any{"bob"},
	}
	stmt := &InsertStmt{scope: scope}
	stmt.Returning("missing")

	if err := stmt.Exec(); err == nil || err.Error() != "unknown returning column: missing" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInsertStmtExecReturningFieldNotAddressable(t *testing.T) {
	pool := &stubPool{}
	user := User{Name: "bob"}
	scope := &ModelScope{
		pool:    pool,
		input:   user,
		table:   "users",
		columns: []string{"name"},
		values:  []any{"bob"},
	}
	stmt := &InsertStmt{scope: scope}
	stmt.Returning("name")

	if err := stmt.Exec(); err == nil || err.Error() != "field name is not addressable" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInsertStmtExecReturningQueryRowError(t *testing.T) {
	scanErr := errors.New("scan fail")
	pool := &stubPool{row: stubRow{err: scanErr}}
	user := &User{Name: "bob"}
	scope := &ModelScope{
		pool:    pool,
		input:   user,
		table:   "users",
		columns: []string{"name"},
		values:  []any{"bob"},
	}
	stmt := &InsertStmt{scope: scope}
	stmt.Returning("id")

	err := stmt.Exec()
	if err == nil || !errors.Is(err, scanErr) || !strings.Contains(err.Error(), "error executing query with returning") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInsertStmtExecReturningSuccess(t *testing.T) {
	pool := &stubPool{row: stubRow{values: []any{10, "alice"}}}
	user := &User{Name: "bob"}
	scope := &ModelScope{
		pool:    pool,
		input:   user,
		table:   "users",
		columns: []string{"name"},
		values:  []any{"bob"},
	}
	stmt := &InsertStmt{scope: scope}
	stmt.Returning("id", "name")

	if err := stmt.Exec(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if user.ID != 10 || user.Name != "alice" {
		t.Fatalf("scan did not update fields: %#v", user)
	}
	if pool.queryRowQuery != "INSERT INTO users (name) VALUES ($1) RETURNING id, name" {
		t.Fatalf("query mismatch: %q", pool.queryRowQuery)
	}
	if !reflect.DeepEqual(pool.queryRowArgs, []any{"bob"}) {
		t.Fatalf("args mismatch: %#v", pool.queryRowArgs)
	}
}
