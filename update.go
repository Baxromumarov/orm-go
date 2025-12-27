package orm_go

import (
	"context"
	"fmt"
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

// AutoTableName sets the table name based on the model type.
func (s *UpdateStmt) AutoTableName() *UpdateStmt {
	setAutoTableName(s.scope)
	return s
}

// Table sets the table name explicitly for this UPDATE.
func (s *UpdateStmt) Table(tableName string) *UpdateStmt {
	setTableName(s.scope, tableName)
	return s
}

// Set adds a column assignment for the UPDATE.
func (s *UpdateStmt) Set(col string, val any) *UpdateStmt {
	s.sets = append(s.sets, setClause{
		col: col,
		val: val,
	})

	return s
}

// Where sets the WHERE clause for the UPDATE.
func (s *UpdateStmt) Where(expr Expr) *UpdateStmt {
	s.where = expr
	return s
}

// Returning adds a RETURNING clause and returns the statement for chaining.
func (s *UpdateStmt) Returning(cols ...string) *UpdateStmt {
	s.returningCols = cloneStrings(cols)
	return s
}

// Exec builds and executes the UPDATE statement.
func (s *UpdateStmt) Exec() error {
	if s.ctx == nil {
		s.ctx = context.Background()
	}

	if s.scope != nil {
		if s.scope.table == "" {
			s.scope.table = ParseTableName(s.scope.input)
		}
		if len(s.sets) == 0 && len(s.scope.columns) == 0 && s.scope.input != nil {
			s.scope.columns, s.scope.values = ParseInsertColumns(s.scope.input)
		}
	}

	if err := s.scope.validate(); err != nil {
		return err
	}

	query, args, err := s.build()
	if err != nil {
		return fmt.Errorf("error building update query: %w", err)
	}

	if len(s.returningCols) == 0 {
		_, err = s.scope.pool.Exec(s.ctx, query, args...)
		if err != nil {
			return fmt.Errorf("error executing update: %w", err)
		}
		return nil
	}

	scanArgs, err := scanArgsForReturning(s.scope.input, s.returningCols)
	if err != nil {
		return err
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
		writeSetClauses(sb, s.sets)
	} else {
		if err := writeColumnAssignments(sb, s.scope.columns, s.scope.values); err != nil {
			return "", nil, err
		}
	}

	appendWhereClause(sb, s.where)
	writeReturning(&sb.sql, s.returningCols)

	return sb.sql.String(), sb.args, nil
}

func writeSetClauses(sb *sqlBuilder, sets []setClause) {
	for i, set := range sets {
		if i > 0 {
			sb.sql.WriteString(", ")
		}
		sb.sql.WriteString(set.col)
		sb.sql.WriteString(" = ")
		sb.sql.WriteString(sb.Arg(set.val))
	}
}

func writeColumnAssignments(sb *sqlBuilder, columns []string, values []any) error {
	if len(columns) == 0 {
		return fmt.Errorf("no columns to update")
	}
	if len(columns) != len(values) {
		return fmt.Errorf("columns and values length mismatch")
	}

	for i, col := range columns {
		if i > 0 {
			sb.sql.WriteString(", ")
		}
		sb.sql.WriteString(col)
		sb.sql.WriteString(" = ")
		sb.sql.WriteString(sb.Arg(values[i]))
	}

	return nil
}
