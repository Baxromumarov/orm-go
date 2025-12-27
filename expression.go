package orm_go

import (
	"strconv"
	"strings"
)

type Expr interface {
	build(sb *sqlBuilder)
}

type sqlBuilder struct {
	sql  strings.Builder
	args []any
}

func (b *sqlBuilder) Arg(v any) string {
	b.args = append(b.args, v)
	return "$" + strconv.Itoa(len(b.args))
}

type eqExpr struct {
	col string
	val any
}

func Eq(col string, val any) Expr {
	return eqExpr{col, val}
}

func (e eqExpr) build(b *sqlBuilder) {
	b.sql.WriteString(e.col)
	b.sql.WriteString(" = ")
	b.sql.WriteString(b.Arg(e.val))
}

type andExpr struct {
	exprs []Expr
}

func And(exprs ...Expr) Expr {
	return andExpr{exprs}
}

func (e andExpr) build(b *sqlBuilder) {
	b.sql.WriteString("(")
	for i, ex := range e.exprs {
		if i > 0 {
			b.sql.WriteString(" AND ")
		}
		ex.build(b)
	}
	b.sql.WriteString(")")
}

type orExpr struct {
	exprs []Expr
}

func Or(exprs ...Expr) Expr {
	return orExpr{exprs}
}

func (e orExpr) build(b *sqlBuilder) {
	b.sql.WriteString("(")
	for i, ex := range e.exprs {
		if i > 0 {
			b.sql.WriteString(" OR ")
		}
		ex.build(b)
	}
	b.sql.WriteString(")")
}
