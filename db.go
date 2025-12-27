package orm_go

import (
	//"context"
	//
	//"github.com/jackc/pgx/v5"

	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Dsn  string
	Ctx  context.Context
	conn *pgxpool.Pool
}

type DB struct {
	*Config
	Err error
}

// Connect creates a DB instance and initializes the pgx pool.
func Connect(config *Config) (*DB, error) {
	db := &DB{Config: config}

	pool, err := pgxpool.New(config.Ctx, config.Dsn)
	if err != nil {
		return nil, err
	}

	db.conn = pool

	return db, nil
}

// Close releases the underlying pgx pool.
func (db *DB) Close() {
	if db.conn != nil {
		db.conn.Close()
	}
}

// Conn returns the underlying pgx pool.
func (db *DB) Conn() *pgxpool.Pool { return db.conn }

// DB returns the receiver for fluent chaining.
func (db *DB) DB() *DB { return db }
