package orm_go

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
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
	if s.scope == nil || s.scope.pool == nil {
		return fmt.Errorf("db connection is nil")
	}
	if s.scope.table == "" {
		s.scope.table = ParseTableName(s.scope.input)
	}
	if s.scope.table == "" {
		return fmt.Errorf("table name is required")
	}
	if len(s.scope.columns) == 0 {
		return fmt.Errorf("no columns to insert")
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

	q := `INSERT INTO ` + s.scope.table + ` (`
	for i, col := range s.scope.columns {
		if i > 0 {
			q += ", "

		}
		q += col

	}
	q += ") VALUES ("
	for i := range s.scope.columns {
		if i > 0 {
			q += ", "
		}
		q += "$" + strconv.Itoa(i+1)
	}
	q += ")"
	if len(s.returningCols) > 0 {
		q += " RETURNING "
		for i, ret := range s.returningCols {
			if i > 0 {
				q += ", "
			}
			q += ret
		}
	}

	return q, nil
}

func buildModelMeta(model any) *modelMeta {
	t := reflect.TypeOf(model)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	meta := &modelMeta{
		byColumn: make(map[string]fieldMeta),
	}

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		tag := sf.Tag.Get(TagKey)
		if tag == "" {
			continue
		}
		meta.byColumn[tag] = fieldMeta{
			index:  i,
			column: tag,
		}
	}

	return meta
}
