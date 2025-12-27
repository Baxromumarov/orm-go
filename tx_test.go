package orm_go

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type stubTx struct {
	beginErr    error
	commitErr   error
	rollbackErr error
	execErr     error
	queryErr    error
	queryRow    pgx.Row
	rows        pgx.Rows
	sendBatch   pgx.BatchResults

	beginCount    int
	commitCount   int
	rollbackCount int

	execQuery     string
	execArgs      []any
	queryQuery    string
	queryArgs     []any
	queryRowQuery string
	queryRowArgs  []any
}

func (s *stubTx) Begin(context.Context) (pgx.Tx, error) {
	s.beginCount++
	if s.beginErr != nil {
		return nil, s.beginErr
	}
	return s, nil
}

func (s *stubTx) Commit(context.Context) error {
	s.commitCount++
	return s.commitErr
}

func (s *stubTx) Rollback(context.Context) error {
	s.rollbackCount++
	return s.rollbackErr
}

func (s *stubTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}

func (s *stubTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	if s.sendBatch != nil {
		return s.sendBatch
	}
	return &stubBatchResults{}
}

func (s *stubTx) LargeObjects() pgx.LargeObjects { return pgx.LargeObjects{} }

func (s *stubTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}

func (s *stubTx) Exec(_ context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	s.execQuery = sql
	s.execArgs = append([]any(nil), arguments...)
	return pgconn.CommandTag{}, s.execErr
}

func (s *stubTx) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	s.queryQuery = sql
	s.queryArgs = append([]any(nil), args...)
	if s.queryErr != nil {
		return nil, s.queryErr
	}
	if s.rows != nil {
		return s.rows, nil
	}
	return &stubRows{}, nil
}

func (s *stubTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	s.queryRowQuery = sql
	s.queryRowArgs = append([]any(nil), args...)
	if s.queryRow != nil {
		return s.queryRow
	}
	return stubRow{}
}

func (s *stubTx) Conn() *pgx.Conn { return nil }

func TestDBTxNil(t *testing.T) {
	var db *DB
	if err := db.Tx(context.Background(), func(tx *Tx) error { return nil }); err == nil || !errors.Is(err, ErrNilDB) {
		t.Fatalf("unexpected error: %v", err)
	}

	db = &DB{}
	if err := db.Tx(context.Background(), func(tx *Tx) error { return nil }); err == nil || !errors.Is(err, ErrNilDB) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWithTxCommit(t *testing.T) {
	stub := &stubTx{}
	err := withTx(context.Background(), func(ctx context.Context) (pgx.Tx, error) {
		return stub, nil
	}, func(tx *Tx) error {
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.commitCount != 1 || stub.rollbackCount != 0 {
		t.Fatalf("commit/rollback mismatch: commit=%d rollback=%d", stub.commitCount, stub.rollbackCount)
	}
}

func TestWithTxRollbackOnError(t *testing.T) {
	stub := &stubTx{}
	fnErr := errors.New("fail")
	err := withTx(context.Background(), func(ctx context.Context) (pgx.Tx, error) {
		return stub, nil
	}, func(tx *Tx) error {
		return fnErr
	})

	if err == nil || !errors.Is(err, fnErr) {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.commitCount != 0 || stub.rollbackCount != 1 {
		t.Fatalf("commit/rollback mismatch: commit=%d rollback=%d", stub.commitCount, stub.rollbackCount)
	}
}

func TestWithTxRollbackOnPanic(t *testing.T) {
	stub := &stubTx{}

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
		if stub.rollbackCount != 1 {
			t.Fatalf("expected rollback, got %d", stub.rollbackCount)
		}
	}()

	_ = withTx(context.Background(), func(ctx context.Context) (pgx.Tx, error) {
		return stub, nil
	}, func(tx *Tx) error {
		panic("boom")
	})
}

func TestTxModelUsesTransactionPool(t *testing.T) {
	stub := &stubTx{}
	tx := &Tx{tx: stub}
	scope := tx.Model(&User{})

	if scope.pool != tx {
		t.Fatalf("expected scope to use tx pool")
	}
	if scope.input == nil {
		t.Fatalf("expected scope input to be set")
	}
}

func TestTxExecQueryDelegation(t *testing.T) {
	stub := &stubTx{}
	tx := &Tx{tx: stub}

	if _, err := tx.Exec(context.Background(), "UPDATE users SET name=$1", "bob"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.execQuery != "UPDATE users SET name=$1" {
		t.Fatalf("exec query mismatch: %q", stub.execQuery)
	}
	if !reflect.DeepEqual(stub.execArgs, []any{"bob"}) {
		t.Fatalf("exec args mismatch: %#v", stub.execArgs)
	}
}
