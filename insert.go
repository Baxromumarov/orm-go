package orm_go

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
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

	if err != nil {
		fmt.Println(">>> Error:", err)
		return
	}
*/

type InsertStmt struct {
	scope         *ModelScope
	ctx           context.Context
	returningCols []string
}

type fieldMeta struct {
	index  int
	column string
}

type modelMeta struct {
	byColumn map[string]fieldMeta
}

func (s *InsertStmt) AutoTableName() *InsertStmt {
	if s.scope != nil {
		s.scope.table = ParseTableName(s.scope.input)
	}

	return s
}

func (s *InsertStmt) Table(tableName string) *InsertStmt {
	if s.scope != nil {
		s.scope.table = tableName
	}
	return s
}

func (s *InsertStmt) Returning(cols ...string) *InsertStmt {
	s.returningCols = append([]string(nil), cols...)
	return s
}

func (s *InsertStmt) Exec() error {
	if s.ctx == nil {
		s.ctx = context.Background()
	}

	if s.scope != nil {
		if s.scope.table == "" {
			s.scope.table = ParseTableName(s.scope.input)
		}
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

	query, err := s.build()
	if err != nil {
		return fmt.Errorf("error building query: %w", err)
	}

	// No RETURNING requested \-\> Exec (no row to scan).
	if len(s.returningCols) == 0 {
		_, err = s.scope.pool.Exec(s.ctx, query, s.scope.values...)
		if err != nil {
			return fmt.Errorf("error executing query: %w", err)
		}
		return nil
	}

	var meta = buildModelMeta(s.scope.input)
	var v = reflect.ValueOf(s.scope.input)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	scanArgs := make([]any, 0, len(s.returningCols))

	for _, col := range s.returningCols {
		fm, ok := meta.byColumn[col]
		if !ok {
			return fmt.Errorf("unknown returning column: %s", col)
		}

		field := v.Field(fm.index)
		if !field.CanAddr() {
			return fmt.Errorf("field %s is not addressable", col)
		}

		scanArgs = append(scanArgs, field.Addr().Interface())
	}

	if err := s.scope.pool.QueryRow(s.ctx, query, s.scope.values...).Scan(scanArgs...); err != nil {
		return fmt.Errorf("error executing query with returning: %w", err)
	}

	return nil
}

func (s *InsertStmt) build() (string, error) {
	if s.scope == nil {
		return "", fmt.Errorf("model scope is nil")
	}
	q := strings.Builder{}

	q.WriteString(`INSERT INTO `)
	q.WriteString(s.scope.table)
	q.WriteString(` (`)

	for i, col := range s.scope.columns {
		if i > 0 {
			q.WriteString(", ")

		}
		q.WriteString(col)

	}
	q.WriteString(`) VALUES (`)
	for i := range s.scope.columns {
		if i > 0 {
			q.WriteString(", ")
		}
		q.WriteString("$")
		q.WriteString(strconv.Itoa(i + 1))
	}

	q.WriteString(")")

	if len(s.returningCols) > 0 {
		q.WriteString(" RETURNING ")
		for i, ret := range s.returningCols {
			if i > 0 {
				q.WriteString(", ")
			}
			q.WriteString(ret)
		}
	}

	return q.String(), nil
}

func buildModelMeta(model any) *modelMeta {
	meta := &modelMeta{
		byColumn: make(map[string]fieldMeta),
	}

	var t reflect.Type
	switch v := model.(type) {
	case reflect.Type:
		t = v
	default:
		if model == nil {
			return meta
		}
		t = reflect.TypeOf(model)
	}

	// unwrap pointers
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t == nil || t.Kind() != reflect.Struct {
		return meta
	}

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)

		raw := sf.Tag.Get(TagKey)
		if raw == "" {
			continue
		}

		col := strings.Split(raw, ",")[0]

		meta.byColumn[col] = fieldMeta{
			index:  i,
			column: col,
		}
	}

	return meta
}
