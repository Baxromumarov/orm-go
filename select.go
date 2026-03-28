package orm_go

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// SelectBuilder is Stage 1: Must set table first.
type SelectBuilder interface {
	Table(name string) SelectBuilderWithTable
	AutoTableName() SelectBuilderWithTable
}

// SelectBuilderWithTable is Stage 2: Table is set, can add filters/options or execute.
type SelectBuilderWithTable interface {
	Columns(cols ...string) SelectBuilderWithTable
	Where(expr Expr) SelectBuilderWithTable
	Join(table string, joinType JoinType, on Expr) SelectBuilderWithTable
	Limit(n int) SelectBuilderWithTable
	Offset(n int) SelectBuilderWithTable
	One(dest any) error
	Many(dest any) error
	Count() (int, error)
}

// selectBuilder is the unexported concrete builder implementing all select stages.
type selectBuilder struct {
	scope *ModelScope
	ctx   context.Context
	where Expr
	pagination
	joins []joinClause
}

type JoinType string

const (
	Inner    JoinType = "INNER"
	Left     JoinType = "LEFT"
	Right    JoinType = "RIGHT"
	FullJoin JoinType = "FULL"
)

type joinClause struct {
	typ   JoinType
	table string
	on    Expr
}

type pagination struct {
	limit  *int
	offset *int
}

// Ensure selectBuilder implements all interfaces at compile time.
var (
	_ SelectBuilder          = (*selectBuilder)(nil)
	_ SelectBuilderWithTable = (*selectBuilder)(nil)
)

// SelectStmt is a type alias for backward compatibility.
// Deprecated: Use SelectBuilder/SelectBuilderWithTable interfaces instead.
type SelectStmt = selectBuilder

// AutoTableName sets the table name based on the model type.
// Implements SelectBuilder interface, returns SelectBuilderWithTable.
func (ss *selectBuilder) AutoTableName() SelectBuilderWithTable {
	if ss.scope != nil {
		ss.scope.table = ParseTableName(ss.scope.input)
	}
	return ss
}

// Table sets the table name explicitly for this SELECT.
// Implements SelectBuilder interface, returns SelectBuilderWithTable.
func (ss *selectBuilder) Table(tableName string) SelectBuilderWithTable {
	if ss.scope != nil {
		ss.scope.table = tableName
	}

	return ss
}

// Columns sets the column list for the SELECT.
// Implements SelectBuilderWithTable interface.
func (ss *selectBuilder) Columns(cols ...string) SelectBuilderWithTable {
	ss.scope.columns = append([]string(nil), cols...)
	return ss
}

// Join adds a JOIN clause to the SELECT.
// Implements SelectBuilderWithTable interface.
func (ss *selectBuilder) Join(table string, joinType JoinType, on Expr) SelectBuilderWithTable {
	ss.joins = append(ss.joins, joinClause{
		typ:   joinType,
		table: table,
		on:    on,
	})
	return ss
}

// Where sets the WHERE clause for the SELECT.
// Implements SelectBuilderWithTable interface.
func (ss *selectBuilder) Where(expr Expr) SelectBuilderWithTable {
	ss.where = expr
	return ss
}

// Count executes a SELECT COUNT(*) with the current filters.
// Implements SelectBuilderWithTable interface.
func (ss *selectBuilder) Count() (int, error) {
	if ss.ctx == nil {
		ss.ctx = context.Background()
	}

	if err := ss.scope.validate(); err != nil {
		return 0, err
	}
	sb := &sqlBuilder{}
	sb.sql.WriteString("SELECT COUNT(*) FROM ")
	sb.sql.WriteString(ss.scope.table)

	if len(ss.joins) > 0 {
		if err := validateQualifiedColumns(ss.scope.columns); err != nil {
			return 0, err
		}
		if err := validateExprQualified(ss.where); err != nil {
			return 0, err
		}
		for _, j := range ss.joins {
			if err := validateExprQualified(j.on); err != nil {
				return 0, err
			}
		}
	}

	for _, j := range ss.joins {
		sb.sql.WriteString(" ")
		sb.sql.WriteString(string(j.typ))
		sb.sql.WriteString(" JOIN ")
		sb.sql.WriteString(j.table)
		sb.sql.WriteString(" ON ")
		j.on.build(sb)
	}

	if ss.where != nil {
		sb.sql.WriteString(" WHERE ")
		ss.where.build(sb)
	}

	query := sb.sql.String()
	args := sb.args
	var count int
	err := ss.scope.pool.QueryRow(ss.ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

var (
	ErrNotFound     = errors.New("orm: no rows found")
	ErrMultipleRows = errors.New("orm: more than one row returned. For multiple rows, use Many()")
)

// One executes the SELECT and scans a single row into dest.
// Implements SelectBuilderWithTable interface.
func (ss *selectBuilder) One(dest any) error {
	if ss.ctx == nil {
		ss.ctx = context.Background()
	}

	if err := ss.scope.validate(); err != nil {
		return err
	}

	destVal, destType, err := validateOneDest(dest)
	if err != nil {
		return err
	}
	if len(ss.scope.columns) == 0 {
		ss.scope.columns = columnsFromType(destType)
		if len(ss.scope.columns) == 0 {
			return errors.New("orm: no columns to select")
		}
	}

	query, args, err := ss.build()
	if err != nil {
		return err
	}

	meta := GetModelMeta(destType)
	scanArgs, err := ScanArgsForColumns(destVal, ss.scope.columns, meta)
	if err != nil {
		return err
	}

	rows, err := ss.scope.pool.Query(ss.ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	// No rows
	if !rows.Next() {
		return ErrNotFound
	}
	// Scan first row
	if err := rows.Scan(scanArgs...); err != nil {
		return err
	}

	// Second row exists → error
	if rows.Next() {
		return ErrMultipleRows
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

// Many executes the SELECT and scans rows into dest (a slice pointer).
// Implements SelectBuilderWithTable interface.
func (ss *selectBuilder) Many(dest any) error {
	if ss.ctx == nil {
		ss.ctx = context.Background()
	}

	if err := ss.scope.validate(); err != nil {
		return err
	}

	sliceVal, elemType, elemIsPtr, err := validateManyDest(dest)
	if err != nil {
		return err
	}
	if len(ss.scope.columns) == 0 {
		ss.scope.columns = columnsFromType(elemType)
		if len(ss.scope.columns) == 0 {
			return errors.New("orm: no columns to select")
		}
	}

	query, args, err := ss.build()
	if err != nil {
		return err
	}

	rows, err := ss.scope.pool.Query(ss.ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	meta := GetModelMeta(elemType)

	for rows.Next() {
		var elem reflect.Value
		if elemIsPtr {
			elem = reflect.New(elemType)
		} else {
			elem = reflect.New(elemType).Elem()
		}
		target := elem
		if elemIsPtr {
			target = elem.Elem()
		}

		scanArgs, err := ScanArgsForColumns(target, ss.scope.columns, meta)
		if err != nil {
			return err
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return err
		}

		if elemIsPtr {
			sliceVal.Set(reflect.Append(sliceVal, elem))
		} else {
			sliceVal.Set(reflect.Append(sliceVal, target))
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

// Limit sets a LIMIT for the SELECT.
// Implements SelectBuilderWithTable interface.
func (ss *selectBuilder) Limit(limit int) SelectBuilderWithTable {
	ss.limit = &limit
	return ss
}

// Offset sets an OFFSET for the SELECT.
// Implements SelectBuilderWithTable interface.
func (ss *selectBuilder) Offset(offset int) SelectBuilderWithTable {
	ss.offset = &offset
	return ss
}

/*
SELECT * FROM users WHERE id = 12;

SELECT id, name FROM users WHERE id = 12;
*/
func (ss *selectBuilder) build() (string, []any, error) {
	if ss.ctx == nil {
		ss.ctx = context.Background()
	}

	if err := ss.scope.validate(); err != nil {
		return "", nil, err
	}

	sb := &sqlBuilder{}
	sb.sql.WriteString("SELECT ")

	if len(ss.scope.columns) == 0 {
		sb.sql.WriteString("*")
	} else {
		sb.WriteColumnList(ss.scope.columns)
	}

	sb.sql.WriteString(" FROM ")
	sb.sql.WriteString(ss.scope.table)

	if len(ss.joins) > 0 {
		if err := validateQualifiedColumns(ss.scope.columns); err != nil {
			return "", nil, err
		}
		if err := validateExprQualified(ss.where); err != nil {
			return "", nil, err
		}
		for _, j := range ss.joins {
			if err := validateExprQualified(j.on); err != nil {
				return "", nil, err
			}
		}
	}

	for _, j := range ss.joins {
		sb.sql.WriteString(" ")
		sb.sql.WriteString(string(j.typ))
		sb.sql.WriteString(" JOIN ")
		sb.sql.WriteString(j.table)
		sb.sql.WriteString(" ON ")
		j.on.build(sb)
	}

	if ss.where != nil {
		sb.sql.WriteString(" WHERE ")
		ss.where.build(sb)
	}

	if ss.limit != nil {
		sb.sql.WriteString(" LIMIT ")
		sb.sql.WriteString(sb.Arg(ss.limit))
	}
	if ss.offset != nil {
		sb.sql.WriteString(" OFFSET ")
		sb.sql.WriteString(sb.Arg(ss.offset))
	}

	return sb.sql.String(), sb.args, nil
}

func columnIsUnqualified(col string) bool {
	col = strings.TrimSpace(col)
	if col == "" {
		return false
	}
	if strings.Contains(col, ".") {
		return false
	}
	for _, r := range col {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '_' {
			continue
		}
		return false
	}
	return true
}

func validateOneDest(dest any) (reflect.Value, reflect.Type, error) {
	if dest == nil {
		return reflect.Value{}, nil, errors.New("dest must be a non-nil pointer to struct")
	}

	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return reflect.Value{}, nil, errors.New("dest must be a non-nil pointer to struct")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, nil, errors.New("dest must be a pointer to struct")
	}

	return v, v.Type(), nil
}

func validateManyDest(dest any) (reflect.Value, reflect.Type, bool, error) {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return reflect.Value{}, nil, false, errors.New("dest must be a pointer to slice")
	}

	v = v.Elem()
	if v.Kind() != reflect.Slice {
		return reflect.Value{}, nil, false, errors.New("dest must be a pointer to slice")
	}

	elemType := v.Type().Elem()
	elemIsPtr := false
	if elemType.Kind() == reflect.Ptr {
		elemIsPtr = true
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return reflect.Value{}, nil, false, errors.New("dest must be a pointer to slice of structs")
	}

	return v, elemType, elemIsPtr, nil
}

func columnsFromType(t reflect.Type) []string {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}

	cols := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		raw := sf.Tag.Get(TagKey)
		if raw == "" {
			continue
		}
		col := strings.Split(raw, ",")[0]
		if col == "" {
			continue
		}
		cols = append(cols, col)
	}
	return cols
}

// scanArgsForColumns is now in metadata.go as ScanArgsForColumns

func validateQualifiedColumns(columns []string) error {
	for _, col := range columns {
		if columnIsUnqualified(col) {
			return fmt.Errorf("ambiguous column: %s", col)
		}
	}
	return nil
}

func validateExprQualified(expr Expr) error {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case eqExpr:
		if columnIsUnqualified(e.col) {
			return fmt.Errorf("ambiguous column: %s", e.col)
		}
	case andExpr:
		for _, ex := range e.exprs {
			if err := validateExprQualified(ex); err != nil {
				return err
			}
		}
	case orExpr:
		for _, ex := range e.exprs {
			if err := validateExprQualified(ex); err != nil {
				return err
			}
		}
	}

	return nil
}
