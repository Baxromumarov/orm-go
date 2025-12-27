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

// WriteColumnList writes "col1, col2, col3" to the SQL buffer.
func (b *sqlBuilder) WriteColumnList(cols []string) {
	for i, col := range cols {
		if i > 0 {
			b.sql.WriteString(", ")
		}
		b.sql.WriteString(col)
	}
}

// WritePlaceholders writes "$1, $2, $3" for count placeholders, starting from current arg count.
func (b *sqlBuilder) WritePlaceholders(count int) {
	start := len(b.args) + 1
	for i := 0; i < count; i++ {
		if i > 0 {
			b.sql.WriteString(", ")
		}
		b.sql.WriteString("$")
		b.sql.WriteString(strconv.Itoa(start + i))
	}
}

// WriteReturning writes " RETURNING col1, col2" if cols is non-empty.
func (b *sqlBuilder) WriteReturning(cols []string) {
	if len(cols) == 0 {
		return
	}
	b.sql.WriteString(" RETURNING ")
	b.WriteColumnList(cols)
}

// WriteSetClause writes "col1 = $1, col2 = $2" for UPDATE SET.
func (b *sqlBuilder) WriteSetClause(cols []string, vals []any) {
	for i, col := range cols {
		if i > 0 {
			b.sql.WriteString(", ")
		}
		b.sql.WriteString(col)
		b.sql.WriteString(" = ")
		b.sql.WriteString(b.Arg(vals[i]))
	}
}

type eqExpr struct {
	col string
	val any
}

// Eq builds a "col = value" expression.
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

// And combines expressions with AND.
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

// Or combines expressions with OR.
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
