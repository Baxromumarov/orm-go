package orm_go

import (
	"strconv"
	"strings"
)

func cloneStrings(in []string) []string {
	return append([]string(nil), in...)
}

func setAutoTableName(scope *ModelScope) {
	if scope != nil {
		scope.table = ParseTableName(scope.input)
	}
}

func setTableName(scope *ModelScope, tableName string) {
	if scope != nil {
		scope.table = tableName
	}
}

func writeIdentList(b *strings.Builder, items []string) {
	for i, item := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(item)
	}
}

func writeReturning(b *strings.Builder, cols []string) {
	if len(cols) == 0 {
		return
	}
	b.WriteString(" RETURNING ")
	writeIdentList(b, cols)
}

func writePlaceholderGroup(b *strings.Builder, count int, start int) int {
	b.WriteString("(")
	for i := 0; i < count; i++ {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString("$")
		b.WriteString(strconv.Itoa(start + i))
	}
	b.WriteString(")")
	return start + count
}

func appendWhereClause(sb *sqlBuilder, expr Expr) {
	sb.sql.WriteString(" WHERE ")
	expr.build(sb)
}
