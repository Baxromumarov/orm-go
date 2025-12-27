package orm_go

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

/*Insert

Example usage:

	type User struct {
		ID      int64  `orm:"id"`
		Name    string `orm:"name"`
		Balance int    `orm:"balance"`
	}

	user := User{Name: "Uzb", Balance: 1000}
	err = db.
	Model(&user).
	Insert(context.Background()).
	AutoTableName().
	Returning(
		"id",
		"name",
		"balance",
	).Exec()


*/

type InsertStmt struct {
	scope         *ModelScope
	ctx           context.Context
	returningCols []string
}

// ErrNoInsertRows is returned when a batch insert receives an empty slice.
var ErrNoInsertRows = errors.New("orm: no rows to insert")

// AutoTableName sets the table name based on the model type.
func (s *InsertStmt) AutoTableName() *InsertStmt {
	setAutoTableName(s.scope)
	return s
}

// Table sets the table name explicitly for this INSERT.
func (s *InsertStmt) Table(tableName string) *InsertStmt {
	setTableName(s.scope, tableName)
	return s
}

// Returning adds a RETURNING clause and returns the statement for chaining.
func (s *InsertStmt) Returning(cols ...string) *InsertStmt {
	s.returningCols = cloneStrings(cols)
	return s
}

// Exec builds and executes the INSERT statement.
func (s *InsertStmt) Exec() error {
	if s.ctx == nil {
		s.ctx = context.Background()
	}

	if s.scope != nil {
		if s.scope.table == "" {
			s.scope.table = inferTableNameFromInput(s.scope.input)
		}
	}

	if s.scope != nil && isBatchInput(s.scope.input) {
		return s.execBatch()
	}

	if s.scope != nil {
		if len(s.scope.columns) == 0 && s.scope.input != nil {
			s.scope.columns, s.scope.values = ParseInsertColumns(s.scope.input)
		}
	}

	if err := s.scope.validate(); err != nil {
		return err
	}
	if len(s.scope.columns) == 0 {
		return ErrNoInsertCols
	}

	query, args, err := s.build()
	if err != nil {
		return fmt.Errorf("error building query: %w", err)
	}

	// No RETURNING requested \-\> Exec (no row to scan).
	if len(s.returningCols) == 0 {
		_, err = s.scope.pool.Exec(s.ctx, query, args...)
		if err != nil {
			return fmt.Errorf("error executing query: %w", err)
		}
		return nil
	}

	scanArgs, err := scanArgsForReturning(s.scope.input, s.returningCols)
	if err != nil {
		return err
	}

	if err := s.scope.pool.QueryRow(s.ctx, query, args...).Scan(scanArgs...); err != nil {
		return fmt.Errorf("error executing query with returning: %w", err)
	}

	return nil
}

func (s *InsertStmt) execBatch() error {
	if s.ctx == nil {
		s.ctx = context.Background()
	}

	if s.scope == nil {
		return ErrNilScope
	}
	if err := s.scope.validate(); err != nil {
		return err
	}

	slice, elemType, elemIsPtr, sliceSettable, err := batchSliceInfo(s.scope.input)
	if err != nil {
		return err
	}
	if slice.Len() == 0 {
		return ErrNoInsertRows
	}

	cols, args, err := batchColumnsAndValues(slice)
	if err != nil {
		return err
	}
	if len(cols) == 0 {
		return ErrNoInsertCols
	}
	s.scope.columns = cols

	query := buildBatchInsertQuery(s.scope.table, cols, slice.Len(), s.returningCols)

	if len(s.returningCols) == 0 {
		_, err = s.scope.pool.Exec(s.ctx, query, args...)
		if err != nil {
			return fmt.Errorf("error executing batch insert: %w", err)
		}
		return nil
	}

	if !elemIsPtr && !sliceSettable {
		return errors.New("batch insert with returning requires a pointer to slice")
	}

	rows, err := s.scope.pool.Query(s.ctx, query, args...)
	if err != nil {
		return fmt.Errorf("error executing batch insert with returning: %w", err)
	}
	defer rows.Close()

	meta := buildModelMeta(elemType)
	for i := 0; i < slice.Len(); i++ {
		if !rows.Next() {
			return fmt.Errorf("expected %d returned rows, got %d", slice.Len(), i)
		}

		target, err := batchScanTarget(slice, i, elemIsPtr, sliceSettable)
		if err != nil {
			return err
		}

		scanArgs, err := scanArgsForColumnsValue(target, s.returningCols, meta, "unknown returning column: %s")
		if err != nil {
			return err
		}

		if err := rows.Scan(scanArgs...); err != nil {
			return err
		}
	}

	if rows.Next() {
		return fmt.Errorf("expected %d returned rows, got more", slice.Len())
	}
	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

func (s *InsertStmt) build() (string, []any, error) {
	if s.scope == nil {
		return "", nil, fmt.Errorf("model scope is nil")
	}
	q := strings.Builder{}

	q.WriteString(`INSERT INTO `)
	q.WriteString(s.scope.table)
	q.WriteString(` (`)

	writeIdentList(&q, s.scope.columns)
	q.WriteString(`) VALUES `)
	writePlaceholderGroup(&q, len(s.scope.columns), 1)

	writeReturning(&q, s.returningCols)

	return q.String(), s.scope.values, nil
}

func inferTableNameFromInput(input any) string {
	if name := ParseTableName(input); name != "" {
		return name
	}

	v := reflect.ValueOf(input)
	if !v.IsValid() {
		return ""
	}
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return ""
	}
	if v.Len() == 0 {
		return ""
	}
	return ParseTableName(v.Index(0).Interface())
}

func isBatchInput(input any) bool {
	v := reflect.ValueOf(input)
	if !v.IsValid() {
		return false
	}
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	return v.Kind() == reflect.Slice || v.Kind() == reflect.Array
}

func batchSliceInfo(input any) (reflect.Value, reflect.Type, bool, bool, error) {
	v := reflect.ValueOf(input)
	if !v.IsValid() {
		return reflect.Value{}, nil, false, false, errors.New("batch insert expects a slice or array")
	}

	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}, nil, false, false, ErrNoInsertRows
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return reflect.Value{}, nil, false, false, errors.New("batch insert expects a slice or array")
	}

	elemType := v.Type().Elem()
	elemIsPtr := false
	if elemType.Kind() == reflect.Ptr {
		elemIsPtr = true
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return reflect.Value{}, nil, false, false, errors.New("batch insert expects a slice of structs")
	}

	return v, elemType, elemIsPtr, v.CanSet(), nil
}

func batchColumnsAndValues(slice reflect.Value) ([]string, []any, error) {
	if slice.Len() == 0 {
		return nil, nil, ErrNoInsertRows
	}

	cols, vals, err := parseInsertColumnsFromValue(slice.Index(0), 0)
	if err != nil {
		return nil, nil, err
	}
	allValues := append([]any(nil), vals...)

	for i := 1; i < slice.Len(); i++ {
		rowCols, rowVals, err := parseInsertColumnsFromValue(slice.Index(i), i)
		if err != nil {
			return nil, nil, err
		}
		if !sameColumns(cols, rowCols) {
			return nil, nil, fmt.Errorf("insert columns mismatch at index %d", i)
		}
		allValues = append(allValues, rowVals...)
	}

	return cols, allValues, nil
}

func parseInsertColumnsFromValue(v reflect.Value, index int) ([]string, []any, error) {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, nil, fmt.Errorf("nil element at index %d", index)
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, nil, errors.New("batch insert expects a slice of structs")
	}

	cols, vals := ParseInsertColumns(v.Interface())
	return cols, vals, nil
}

func sameColumns(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func buildBatchInsertQuery(table string, cols []string, rows int, returning []string) string {
	var b strings.Builder
	b.WriteString("INSERT INTO ")
	b.WriteString(table)
	b.WriteString(" (")
	writeIdentList(&b, cols)
	b.WriteString(") VALUES ")

	argPos := 1
	for r := 0; r < rows; r++ {
		if r > 0 {
			b.WriteString(", ")
		}
		argPos = writePlaceholderGroup(&b, len(cols), argPos)
	}

	writeReturning(&b, returning)

	return b.String()
}

func batchScanTarget(slice reflect.Value, index int, elemIsPtr bool, sliceSettable bool) (reflect.Value, error) {
	elem := slice.Index(index)
	if elemIsPtr {
		if elem.IsNil() {
			if !sliceSettable {
				return reflect.Value{}, fmt.Errorf("nil element at index %d", index)
			}
			elem = reflect.New(elem.Type().Elem())
			slice.Index(index).Set(elem)
		}
		return elem.Elem(), nil
	}

	if !elem.CanAddr() {
		return reflect.Value{}, errors.New("batch insert with returning requires a pointer to slice")
	}
	return elem, nil
}
