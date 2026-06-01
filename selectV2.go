package orm_go

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type SelectV2Stmt struct {
	fields []any
	scope  *ModelScope
	ctx    context.Context
	where  selectV2Expr
	pagination
}

type SelectBuilderI interface {
	AutoTableName() *SelectV2Stmt
	Table(tableName string) *SelectV2Stmt
	WithContext(ctx context.Context) *SelectV2Stmt
	Where(field any) *SelectV2ConditionBuilder
	Limit(limit int) *SelectV2Stmt
	Offset(offset int) *SelectV2Stmt
	Exec() error
	build() (string, []any, error)
}

var _ SelectBuilderI = (*SelectV2Stmt)(nil)

type selectV2Expr interface {
	buildSelectV2(sb *sqlBuilder, resolver *selectV2Resolver) error
}

type v2ComparisonExpr struct {
	field any
	op    string
	value any
}

func (e v2ComparisonExpr) buildSelectV2(sb *sqlBuilder, resolver *selectV2Resolver) error {
	col, err := resolver.column(e.field)
	if err != nil {
		return err
	}

	sb.sql.WriteString(col)
	sb.sql.WriteString(" ")
	sb.sql.WriteString(e.op)
	sb.sql.WriteString(" ")
	sb.sql.WriteString(sb.Arg(e.value))
	return nil
}

type v2LogicalExpr struct {
	op    string
	exprs []selectV2Expr
}

func (e v2LogicalExpr) buildSelectV2(sb *sqlBuilder, resolver *selectV2Resolver) error {
	sb.sql.WriteString("(")
	for i, expr := range e.exprs {
		if expr == nil {
			return errors.New("nil SelectV2 expression")
		}
		if i > 0 {
			sb.sql.WriteString(" ")
			sb.sql.WriteString(e.op)
			sb.sql.WriteString(" ")
		}
		if err := expr.buildSelectV2(sb, resolver); err != nil {
			return err
		}
	}
	sb.sql.WriteString(")")
	return nil
}

func (m *ModelScope) SelectV2(fields ...any) *SelectV2Stmt {
	return &SelectV2Stmt{
		fields: append([]any(nil), fields...),
		scope:  m,
	}
}

// AutoTableName sets the table name based on the model type.
func (s *SelectV2Stmt) AutoTableName() *SelectV2Stmt {
	if s.scope != nil {
		s.scope.table = ParseTableName(s.scope.input)
	}
	return s
}

// Table sets the table name explicitly for this SELECT.
func (s *SelectV2Stmt) Table(tableName string) *SelectV2Stmt {
	if s.scope != nil {
		s.scope.table = tableName
	}
	return s
}

// WithContext sets the context used by Exec/build. If omitted, context.Background is used.
func (s *SelectV2Stmt) WithContext(ctx context.Context) *SelectV2Stmt {
	s.ctx = ctx
	return s
}

// Where starts a string-free WHERE condition for this SELECT.
// Example: stmt.Where(user.ID).Gt(10)
func (s *SelectV2Stmt) Where(field any) *SelectV2ConditionBuilder {
	return &SelectV2ConditionBuilder{stmt: s, field: field}
}

type SelectV2ConditionBuilder struct {
	stmt  *SelectV2Stmt
	field any
}

func (b *SelectV2ConditionBuilder) Eq(value any) *SelectV2Stmt {
	return b.compare("=", value)
}

func (b *SelectV2ConditionBuilder) Ne(value any) *SelectV2Stmt {
	return b.compare("!=", value)
}

func (b *SelectV2ConditionBuilder) Gt(value any) *SelectV2Stmt {
	return b.compare(">", value)
}

func (b *SelectV2ConditionBuilder) Gte(value any) *SelectV2Stmt {
	return b.compare(">=", value)
}

func (b *SelectV2ConditionBuilder) Lt(value any) *SelectV2Stmt {
	return b.compare("<", value)
}

func (b *SelectV2ConditionBuilder) Lte(value any) *SelectV2Stmt {
	return b.compare("<=", value)
}

func (b *SelectV2ConditionBuilder) compare(op string, value any) *SelectV2Stmt {
	b.stmt.where = v2ComparisonExpr{
		field: b.field,
		op:    op,
		value: value,
	}
	return b.stmt
}

// Limit sets a LIMIT for this SELECT.
func (s *SelectV2Stmt) Limit(limit int) *SelectV2Stmt {
	s.limit = &limit
	return s
}

// Offset sets an OFFSET for this SELECT.
func (s *SelectV2Stmt) Offset(offset int) *SelectV2Stmt {
	s.offset = &offset
	return s
}

// Exec executes the SELECT and scans into the model passed to Model(...).
// If Model was given a pointer to a struct, Exec expects one row.
// If Model was given a pointer to a slice, Exec scans many rows.
func (s *SelectV2Stmt) Exec() error {
	resolver, err := s.prepare()
	if err != nil {
		return err
	}

	if s.scope.input == nil {
		return errors.New("orm: select destination is nil")
	}

	v := reflect.ValueOf(s.scope.input)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return errors.New("orm: select destination must be a non-nil pointer")
	}

	switch v.Elem().Kind() {
	case reflect.Struct:
		return s.execOne(resolver)
	case reflect.Slice:
		return s.execMany(resolver)
	default:
		return errors.New("orm: select destination must be a pointer to struct or slice")
	}
}

func (s *SelectV2Stmt) execOne(resolver *selectV2Resolver) error {
	destVal, destType, err := validateOneDest(s.scope.input)
	if err != nil {
		return err
	}
	if len(s.scope.columns) == 0 {
		s.scope.columns = columnsFromType(destType)
		if len(s.scope.columns) == 0 {
			return errors.New("orm: no columns to select")
		}
	}

	query, args, err := s.buildWithResolver(resolver)
	if err != nil {
		return err
	}

	meta := GetModelMeta(destType)
	scanArgs, err := ScanArgsForColumns(destVal, s.scope.columns, meta)
	if err != nil {
		return err
	}

	rows, err := s.scope.pool.Query(s.ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrNotFound
	}
	if err := rows.Scan(scanArgs...); err != nil {
		return err
	}
	if rows.Next() {
		return ErrMultipleRows
	}
	return rows.Err()
}

func (s *SelectV2Stmt) execMany(resolver *selectV2Resolver) error {
	sliceVal, elemType, elemIsPtr, err := validateManyDest(s.scope.input)
	if err != nil {
		return err
	}
	if len(s.scope.columns) == 0 {
		s.scope.columns = columnsFromType(elemType)
		if len(s.scope.columns) == 0 {
			return errors.New("orm: no columns to select")
		}
	}

	query, args, err := s.buildWithResolver(resolver)
	if err != nil {
		return err
	}

	rows, err := s.scope.pool.Query(s.ctx, query, args...)
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

		scanArgs, err := ScanArgsForColumns(target, s.scope.columns, meta)
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

	return rows.Err()
}

func (s *SelectV2Stmt) build() (string, []any, error) {
	resolver, err := s.prepare()
	if err != nil {
		return "", nil, err
	}
	return s.buildWithResolver(resolver)
}

func (s *SelectV2Stmt) prepare() (*selectV2Resolver, error) {
	if s == nil || s.scope == nil {
		return nil, ErrNilScope
	}
	if s.ctx == nil {
		s.ctx = context.Background()
	}
	if s.scope.table == "" {
		s.scope.table = ParseTableName(s.scope.input)
	}

	resolver, err := newSelectV2Resolver(s.scope.input)
	if err != nil {
		return nil, err
	}
	if len(s.fields) > 0 {
		cols, err := selectV2ColumnsFromFields(resolver, s.fields)
		if err != nil {
			return nil, err
		}
		s.scope.columns = cols
	}
	if err := s.scope.validate(); err != nil {
		return nil, err
	}
	return resolver, nil
}

func (s *SelectV2Stmt) buildWithResolver(resolver *selectV2Resolver) (string, []any, error) {
	sb := &sqlBuilder{}
	sb.sql.WriteString("SELECT ")

	if len(s.scope.columns) == 0 {
		sb.sql.WriteString("*")
	} else {
		sb.WriteColumnList(s.scope.columns)
	}

	sb.sql.WriteString(" FROM ")
	sb.sql.WriteString(s.scope.table)

	if s.where != nil {
		sb.sql.WriteString(" WHERE ")
		if err := s.where.buildSelectV2(sb, resolver); err != nil {
			return "", nil, err
		}
	}

	if s.limit != nil {
		sb.sql.WriteString(" LIMIT ")
		sb.sql.WriteString(sb.Arg(s.limit))
	}
	if s.offset != nil {
		sb.sql.WriteString(" OFFSET ")
		sb.sql.WriteString(sb.Arg(s.offset))
	}

	return sb.sql.String(), sb.args, nil
}

type selectV2Resolver struct {
	modelValue reflect.Value
	modelType  reflect.Type
}

func newSelectV2Resolver(input any) (*selectV2Resolver, error) {
	modelValue, modelType := selectV2ModelValueAndType(input)
	if modelType == nil || modelType.Kind() != reflect.Struct {
		return nil, errors.New("orm: SelectV2 requires Model to contain a struct or slice of structs")
	}
	return &selectV2Resolver{modelValue: modelValue, modelType: modelType}, nil
}

func (r *selectV2Resolver) column(field any) (string, error) {
	return selectV2ColumnFromField(r.modelValue, r.modelType, field)
}

func selectV2ColumnsFromFields(resolver *selectV2Resolver, fields []any) ([]string, error) {
	cols := make([]string, 0, len(fields))
	for i, field := range fields {
		col, err := resolver.column(field)
		if err != nil {
			return nil, fmt.Errorf("select field %d: %w", i, err)
		}
		cols = append(cols, col)
	}
	return cols, nil
}

func selectV2ModelValueAndType(input any) (reflect.Value, reflect.Type) {
	if input == nil {
		return reflect.Value{}, nil
	}

	v := reflect.ValueOf(input)
	for v.IsValid() && v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}, selectV2StructType(v.Type())
		}
		v = v.Elem()
	}

	if !v.IsValid() {
		return reflect.Value{}, nil
	}

	switch v.Kind() {
	case reflect.Struct:
		return v, v.Type()
	case reflect.Slice, reflect.Array:
		return reflect.Value{}, selectV2StructType(v.Type().Elem())
	default:
		return reflect.Value{}, nil
	}
}

func selectV2StructType(t reflect.Type) reflect.Type {
	for t != nil {
		switch t.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Array:
			t = t.Elem()
		default:
			if t.Kind() == reflect.Struct {
				return t
			}
			return nil
		}
	}
	return nil
}

func selectV2ColumnFromField(modelValue reflect.Value, modelType reflect.Type, selected any) (string, error) {
	selectedValue := reflect.ValueOf(selected)
	if !selectedValue.IsValid() {
		return "", errors.New("nil is not a selectable field")
	}

	matches := make([]string, 0, 1)
	for i := 0; i < modelType.NumField(); i++ {
		structField := modelType.Field(i)
		if structField.PkgPath != "" {
			continue
		}

		column := selectV2ColumnName(structField)
		if column == "" {
			continue
		}

		if !modelValue.IsValid() {
			if selectV2FieldTypeMatches(structField, selectedValue) {
				matches = append(matches, column)
			}
			continue
		}

		fieldValue := modelValue.Field(i)
		if selectV2FieldMatches(fieldValue, selectedValue) {
			matches = append(matches, column)
		}
	}

	switch len(matches) {
	case 0:
		return "", errors.New("field does not match a tagged model field")
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous field match (%s); pass a field pointer from the same model value", strings.Join(matches, ", "))
	}
}

func selectV2ColumnName(field reflect.StructField) string {
	raw := field.Tag.Get(TagKey)
	if raw == "" {
		return ""
	}
	col := strings.Split(raw, ",")[0]
	return strings.TrimSpace(col)
}

func selectV2FieldMatches(modelField reflect.Value, selected reflect.Value) bool {
	if modelField.CanAddr() && selected.Kind() == reflect.Ptr && selected.Type() == modelField.Addr().Type() {
		return selected.Pointer() == modelField.Addr().Pointer()
	}

	if !modelField.CanInterface() || !selected.Type().AssignableTo(modelField.Type()) {
		return false
	}

	return reflect.DeepEqual(selected.Interface(), modelField.Interface())
}

func selectV2FieldTypeMatches(field reflect.StructField, selected reflect.Value) bool {
	if selected.Kind() == reflect.Ptr {
		return selected.Type().Elem() == field.Type
	}
	return selected.Type().AssignableTo(field.Type)
}
