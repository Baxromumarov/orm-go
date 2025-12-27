package orm_go

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestSelectStmtAutoTableNameAndTable(t *testing.T) {
	scope := &ModelScope{input: &User{}}
	stmt := &SelectStmt{scope: scope}

	stmt.AutoTableName()
	if scope.table != "users" {
		t.Fatalf("got table %q", scope.table)
	}

	stmt.Table("custom_table")
	if scope.table != "custom_table" {
		t.Fatalf("got table %q", scope.table)
	}
}

func TestSelectStmtColumnsCopies(t *testing.T) {
	scope := &ModelScope{}
	stmt := &SelectStmt{scope: scope}
	cols := []string{"id", "name"}
	stmt.Columns(cols...)
	cols[0] = "changed"

	if scope.columns[0] != "id" {
		t.Fatalf("expected copy of columns")
	}
}

func TestSelectStmtBuildSelectStar(t *testing.T) {
	scope := &ModelScope{
		pool:  &stubPool{},
		table: "users",
	}
	stmt := &SelectStmt{scope: scope}

	got, args, err := stmt.build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "SELECT * FROM users" {
		t.Fatalf("query mismatch: %q", got)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args, got %#v", args)
	}
}

func TestSelectStmtBuildWithWhereLimitOffset(t *testing.T) {
	scope := &ModelScope{
		pool:    &stubPool{},
		table:   "users",
		columns: []string{"id", "name"},
	}
	stmt := &SelectStmt{scope: scope}
	stmt.Where(Eq("id", 1)).Limit(10).Offset(5)

	got, args, err := stmt.build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "SELECT id, name FROM users WHERE id = $1 LIMIT $2 OFFSET $3"
	if got != want {
		t.Fatalf("query mismatch: got %q want %q", got, want)
	}
	if len(args) != 3 || args[0] != 1 {
		t.Fatalf("args mismatch: %#v", args)
	}
	limitPtr, ok := args[1].(*int)
	if !ok || *limitPtr != 10 {
		t.Fatalf("limit arg mismatch: %#v", args[1])
	}
	offsetPtr, ok := args[2].(*int)
	if !ok || *offsetPtr != 5 {
		t.Fatalf("offset arg mismatch: %#v", args[2])
	}
}

func TestSelectStmtBuildWithJoinQualified(t *testing.T) {
	scope := &ModelScope{
		pool:    &stubPool{},
		table:   "users",
		columns: []string{"users.id", "profiles.user_id"},
	}
	stmt := &SelectStmt{scope: scope}
	stmt.Join("profiles", Inner, Eq("users.id", 1))
	stmt.Where(Eq("users.id", 1))

	got, args, err := stmt.build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "SELECT users.id, profiles.user_id FROM users INNER JOIN profiles ON users.id = $1 WHERE users.id = $2"
	if got != want {
		t.Fatalf("query mismatch: got %q want %q", got, want)
	}
	if !reflect.DeepEqual(args, []any{1, 1}) {
		t.Fatalf("args mismatch: %#v", args)
	}
}

func TestSelectStmtBuildAmbiguousColumns(t *testing.T) {
	scope := &ModelScope{
		pool:    &stubPool{},
		table:   "users",
		columns: []string{"id"},
	}
	stmt := &SelectStmt{scope: scope}
	stmt.Join("profiles", Inner, Eq("users.id", 1))

	if _, _, err := stmt.build(); err == nil || err.Error() != "ambiguous column: id" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSelectStmtBuildAmbiguousWhere(t *testing.T) {
	scope := &ModelScope{
		pool:    &stubPool{},
		table:   "users",
		columns: []string{"users.id"},
	}
	stmt := &SelectStmt{scope: scope}
	stmt.Join("profiles", Inner, Eq("users.id", 1))
	stmt.Where(Eq("id", 1))

	if _, _, err := stmt.build(); err == nil || err.Error() != "ambiguous column: id" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSelectStmtBuildAmbiguousJoinOn(t *testing.T) {
	scope := &ModelScope{
		pool:    &stubPool{},
		table:   "users",
		columns: []string{"users.id"},
	}
	stmt := &SelectStmt{scope: scope}
	stmt.Join("profiles", Inner, Eq("id", 1))

	if _, _, err := stmt.build(); err == nil || err.Error() != "ambiguous column: id" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSelectStmtOneDestErrors(t *testing.T) {
	scope := &ModelScope{
		pool:  &stubPool{},
		table: "users",
	}
	stmt := &SelectStmt{scope: scope}

	if err := stmt.One(nil); err == nil || !strings.Contains(err.Error(), "pointer to struct") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := stmt.One(User{}); err == nil || !strings.Contains(err.Error(), "pointer to struct") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := stmt.One(&[]User{}); err == nil || !strings.Contains(err.Error(), "pointer to struct") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSelectStmtManyDestErrors(t *testing.T) {
	scope := &ModelScope{
		pool:  &stubPool{},
		table: "users",
	}
	stmt := &SelectStmt{scope: scope}

	if err := stmt.Many(nil); err == nil || !strings.Contains(err.Error(), "pointer to slice") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := stmt.Many([]User{}); err == nil || !strings.Contains(err.Error(), "pointer to slice") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := stmt.Many(&User{}); err == nil || !strings.Contains(err.Error(), "pointer to slice") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := stmt.Many(&[]int{}); err == nil || !strings.Contains(err.Error(), "slice of structs") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSelectStmtOneNotFound(t *testing.T) {
	pool := &stubPool{rows: newScriptedRows(nil)}
	scope := &ModelScope{
		pool:  pool,
		table: "users",
	}
	stmt := &SelectStmt{scope: scope}

	err := stmt.One(&User{})
	if err == nil || !errors.Is(err, ErrNotFound) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSelectStmtOneMultipleRows(t *testing.T) {
	pool := &stubPool{rows: newScriptedRows([][]any{
		{1, "a", 10},
		{2, "b", 20},
	})}
	scope := &ModelScope{
		pool:  pool,
		table: "users",
	}
	stmt := &SelectStmt{scope: scope}

	var user User
	err := stmt.One(&user)
	if err == nil || !errors.Is(err, ErrMultipleRows) {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 1 || user.Name != "a" || user.Age != 10 {
		t.Fatalf("first row not scanned: %#v", user)
	}
}

func TestSelectStmtOneSuccess(t *testing.T) {
	pool := &stubPool{rows: newScriptedRows([][]any{{1, "Ben"}})}
	scope := &ModelScope{
		pool:    pool,
		table:   "users",
		columns: []string{"id", "name"},
	}
	stmt := &SelectStmt{scope: scope}
	stmt.Where(Eq("id", 1))

	var user User
	if err := stmt.One(&user); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 1 || user.Name != "Ben" || user.Age != 0 {
		t.Fatalf("unexpected user: %#v", user)
	}
	if pool.queryQuery != "SELECT id, name FROM users WHERE id = $1" {
		t.Fatalf("query mismatch: %q", pool.queryQuery)
	}
	if !reflect.DeepEqual(pool.queryArgs, []any{1}) {
		t.Fatalf("args mismatch: %#v", pool.queryArgs)
	}
}

func TestSelectStmtManySuccessValueSlice(t *testing.T) {
	pool := &stubPool{rows: newScriptedRows([][]any{
		{1, "a"},
		{2, "b"},
	})}
	scope := &ModelScope{
		pool:    pool,
		table:   "users",
		columns: []string{"id", "name"},
	}
	stmt := &SelectStmt{scope: scope}

	var users []User
	if err := stmt.Many(&users); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].ID != 1 || users[0].Name != "a" {
		t.Fatalf("unexpected first user: %#v", users[0])
	}
}

func TestSelectStmtManySuccessPointerSlice(t *testing.T) {
	pool := &stubPool{rows: newScriptedRows([][]any{
		{1, "a", 10},
		{2, "b", 20},
	})}
	scope := &ModelScope{
		pool:    pool,
		table:   "users",
		columns: []string{"id", "name", "age"},
	}
	stmt := &SelectStmt{scope: scope}

	var users []*User
	if err := stmt.Many(&users); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[1].ID != 2 || users[1].Age != 20 {
		t.Fatalf("unexpected second user: %#v", users[1])
	}
}

func TestSelectStmtManyRowsErr(t *testing.T) {
	rows := newScriptedRows(nil)
	rows.err = errors.New("rows err")
	pool := &stubPool{rows: rows}
	scope := &ModelScope{
		pool:  pool,
		table: "users",
	}
	stmt := &SelectStmt{scope: scope}

	var users []User
	err := stmt.Many(&users)
	if err == nil || !errors.Is(err, rows.err) {
		t.Fatalf("unexpected error: %v", err)
	}
}
