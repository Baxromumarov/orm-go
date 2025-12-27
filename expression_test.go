package orm_go

import (
	"reflect"
	"testing"
)

func TestSQLBuilderArg(t *testing.T) {
	b := &sqlBuilder{}

	if got := b.Arg("first"); got != "$1" {
		t.Fatalf("got %q want %q", got, "$1")
	}
	if got := b.Arg(2); got != "$2" {
		t.Fatalf("got %q want %q", got, "$2")
	}

	if !reflect.DeepEqual(b.args, []any{"first", 2}) {
		t.Fatalf("args mismatch: %#v", b.args)
	}
}

func TestEqBuild(t *testing.T) {
	b := &sqlBuilder{}
	Eq("name", "bob").build(b)

	if got := b.sql.String(); got != "name = $1" {
		t.Fatalf("sql mismatch: %q", got)
	}
	if !reflect.DeepEqual(b.args, []any{"bob"}) {
		t.Fatalf("args mismatch: %#v", b.args)
	}
}

func TestAndOrBuild(t *testing.T) {
	b := &sqlBuilder{}
	expr := And(
		Eq("a", 1),
		Or(Eq("b", 2), Eq("c", 3)),
	)
	expr.build(b)

	wantSQL := "(a = $1 AND (b = $2 OR c = $3))"
	if got := b.sql.String(); got != wantSQL {
		t.Fatalf("sql mismatch: got %q want %q", got, wantSQL)
	}
	if !reflect.DeepEqual(b.args, []any{1, 2, 3}) {
		t.Fatalf("args mismatch: %#v", b.args)
	}
}
