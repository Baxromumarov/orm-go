package orm_go

import "testing"

type benchUser struct {
	ID      int64  `orm:"id"`
	Name    string `orm:"name"`
	Balance int    `orm:"balance"`
	Age     int    `orm:"age"`
}

func BenchmarkParseInsertColumns(b *testing.B) {
	u := benchUser{ID: 1, Name: "Ben", Balance: 100, Age: 30}
	b.ReportAllocs()
	for b.Loop() {
		ParseInsertColumns(u)
	}
}

func BenchmarkInsertBuild(b *testing.B) {
	scope := &ModelScope{
		table:   "users",
		columns: []string{"name", "age", "balance"},
	}
	stmt := &InsertStmt{scope: scope}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = stmt.build()
	}
}

func BenchmarkInsertBuildReturning(b *testing.B) {
	scope := &ModelScope{
		table:   "users",
		columns: []string{"name", "age", "balance"},
	}
	stmt := &InsertStmt{scope: scope}
	stmt.Returning("id", "name")
	b.ReportAllocs()
	for b.Loop() {
		_, _ = stmt.build()
	}
}

func BenchmarkBatchInsertQuery(b *testing.B) {
	cols := []string{"name", "age", "balance"}
	b.ReportAllocs()
	for b.Loop() {
		_ = buildBatchInsertQuery("users", cols, 100, nil)
	}
}

func BenchmarkUpdateBuildWithSets(b *testing.B) {
	scope := &ModelScope{table: "users"}
	stmt := &UpdateStmt{
		scope: scope,
		where: Eq("id", 1),
	}
	stmt.Set("name", "Ben").Set("balance", 200)
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = stmt.build()
	}
}

func BenchmarkUpdateBuildWithColumns(b *testing.B) {
	scope := &ModelScope{
		table:   "users",
		columns: []string{"name", "balance"},
		values:  []any{"Ben", 200},
	}
	stmt := &UpdateStmt{
		scope: scope,
		where: Eq("id", 1),
	}
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = stmt.build()
	}
}

func BenchmarkSelectBuildSimple(b *testing.B) {
	scope := &ModelScope{
		table:   "users",
		columns: []string{"id", "name"},
		pool:    &stubPool{},
	}
	stmt := &SelectStmt{
		scope: scope,
		where: Eq("id", 1),
	}
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = stmt.build()
	}
}

func BenchmarkSelectBuildWithJoin(b *testing.B) {
	scope := &ModelScope{
		table:   "users",
		columns: []string{"users.id", "profiles.user_id"},
		pool:    &stubPool{},
	}
	stmt := &SelectStmt{
		scope: scope,
		where: Eq("users.id", 1),
	}
	stmt.Join("profiles", Inner, Eq("users.id", 1))
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = stmt.build()
	}
}

func BenchmarkExpressionBuild(b *testing.B) {
	expr := And(
		Eq("a", 1),
		Or(Eq("b", 2), Eq("c", 3)),
	)
	b.ReportAllocs()
	for b.Loop() {
		sb := &sqlBuilder{}
		expr.build(sb)
	}
}
