package orm_go

import (
	"context"
	"fmt"
	"reflect"
)

type UpdateStmt struct {
	scope         *ModelScope
	ctx           context.Context
	sets          []setClause
	where         Expr
	returningCols []string
}
type setClause struct {
	col string
	val any
}

func (s *UpdateStmt) AutoTableName() *UpdateStmt {
	if s.scope != nil {
		s.scope.table = ParseTableName(s.scope.input)
	}
	return s
}

func (s *UpdateStmt) Table(tableName string) *UpdateStmt {
	if s.scope != nil {
		s.scope.table = tableName
	}
	return s
}

func (s *UpdateStmt) Set(col string, val any) *UpdateStmt {
	s.sets = append(s.sets, setClause{
		col: col,
		val: val,
	})

	return s
}

func (s *UpdateStmt) Where(expr Expr) *UpdateStmt {
	s.where = expr
	return s
}

func (s *UpdateStmt) Returning(cols ...string) *UpdateStmt {
	s.returningCols = append([]string(nil), cols...)
	return s
}

func (s *UpdateStmt) Exec() error {
	if s.ctx == nil {
		s.ctx = context.Background()
	}

	if err := s.scope.validate(); err != nil {
		return err
	}

	query, args, err := s.build()
	if err != nil {
		return fmt.Errorf("error building update query: %w", err)
	}
	fmt.Println("FINAL UPDATE QUERY:", query)

	if len(s.returningCols) == 0 {
		_, err = s.scope.pool.Exec(s.ctx, query, args...)
		if err != nil {
			return fmt.Errorf("error executing update: %w", err)
		}
		return nil
	}

	meta := buildModelMeta(s.scope.input)

	v := reflect.ValueOf(s.scope.input)
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

	if err := s.scope.pool.QueryRow(s.ctx, query, args...).Scan(scanArgs...); err != nil {
		return fmt.Errorf("error executing update with returning: %w", err)
	}

	return nil
}

func (s *UpdateStmt) build() (string, []any, error) {
	if s.scope == nil {
		return "", nil, fmt.Errorf("model scope is nil")
	}
	if s.scope.table == "" {
		return "", nil, fmt.Errorf("table name is required")
	}
	if s.where == nil {
		return "", nil, fmt.Errorf("where clause is required")
	}

	sb := &sqlBuilder{}
	sb.sql.WriteString("UPDATE ")
	sb.sql.WriteString(s.scope.table)
	sb.sql.WriteString(" SET ")

	if len(s.sets) > 0 {
		for i, set := range s.sets {
			if i > 0 {
				sb.sql.WriteString(", ")
			}
			sb.sql.WriteString(set.col)
			sb.sql.WriteString(" = ")
			sb.sql.WriteString(sb.Arg(set.val))
		}
	} else {
		if len(s.scope.columns) == 0 {
			return "", nil, fmt.Errorf("no columns to update")
		}
		if len(s.scope.columns) != len(s.scope.values) {
			return "", nil, fmt.Errorf("columns and values length mismatch")
		}

		for i, col := range s.scope.columns {
			if i > 0 {
				sb.sql.WriteString(", ")
			}

			sb.sql.WriteString(col)
			sb.sql.WriteString(" = ")
			sb.sql.WriteString(sb.Arg(s.scope.values[i]))
		}
	}

	sb.sql.WriteString(" WHERE ")
	s.where.build(sb)

	if len(s.returningCols) > 0 {
		sb.sql.WriteString(" RETURNING ")
		for i, ret := range s.returningCols {
			if i > 0 {
				sb.sql.WriteString(", ")
			}
			sb.sql.WriteString(ret)
		}
	}

	return sb.sql.String(), sb.args, nil
}
