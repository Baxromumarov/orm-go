package orm_go

import (
	"reflect"
	"strings"
	"testing"
)

type selectV2User struct {
	ID      int64   `orm:"id"`
	Name    *string `orm:"name"`
	Balance *int    `orm:"balance"`
}

func TestSelectV2ExecScansStructWithValueFields(t *testing.T) {
	name := "Ben"
	balance := 100
	pool := &stubPool{rows: newScriptedRows([][]any{{int64(1), &name, &balance}})}
	user := selectV2User{}
	scope := &ModelScope{pool: pool, input: &user}

	if err := scope.SelectV2(user.ID, user.Name, user.Balance).Exec(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pool.queryQuery != "SELECT id, name, balance FROM selectv2users" {
		t.Fatalf("query mismatch: %q", pool.queryQuery)
	}
	if len(pool.queryArgs) != 0 {
		t.Fatalf("expected no args, got %#v", pool.queryArgs)
	}
	if user.ID != 1 || user.Name == nil || *user.Name != "Ben" || user.Balance == nil || *user.Balance != 100 {
		t.Fatalf("unexpected user: %#v", user)
	}
}

func TestSelectV2BuildWithFieldPointers(t *testing.T) {
	user := User{}
	scope := &ModelScope{pool: &stubPool{}, input: &user}

	query, args, err := scope.SelectV2(&user.ID, &user.Age).
		Where(&user.ID).Gt(12).
		Limit(1).
		build()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "SELECT id, age FROM users WHERE id > $1 LIMIT $2"
	if query != want {
		t.Fatalf("query mismatch: got %q want %q", query, want)
	}
	if len(args) != 2 || args[0] != 12 {
		t.Fatalf("args mismatch: %#v", args)
	}
	limitPtr, ok := args[1].(*int)
	if !ok || *limitPtr != 1 {
		t.Fatalf("limit arg mismatch: %#v", args[1])
	}
}

func TestSelectV2ValueFieldAmbiguous(t *testing.T) {
	user := User{}
	scope := &ModelScope{pool: &stubPool{}, input: &user}

	_, _, err := scope.SelectV2(user.ID).build()
	if err == nil || !strings.Contains(err.Error(), "ambiguous field match") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSelectV2ExecScansSlice(t *testing.T) {
	pool := &stubPool{rows: newScriptedRows([][]any{{1, "a"}, {2, "b"}})}
	var users []User
	scope := &ModelScope{
		pool:    pool,
		input:   &users,
		columns: []string{"id", "name"},
	}

	if err := scope.SelectV2().Exec(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pool.queryQuery != "SELECT id, name FROM users" {
		t.Fatalf("query mismatch: %q", pool.queryQuery)
	}
	if !reflect.DeepEqual(users, []User{{ID: 1, Name: "a"}, {ID: 2, Name: "b"}}) {
		t.Fatalf("unexpected users: %#v", users)
	}
}
