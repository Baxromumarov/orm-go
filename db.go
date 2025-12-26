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

func Connect(config *Config) (*DB, error) {
	db := &DB{Config: config}

	pool, err := pgxpool.New(config.Ctx, config.Dsn)
	if err != nil {
		return nil, err
	}

	db.conn = pool

	return db, nil
}

func (db *DB) Close() {
	if db.conn != nil {
		db.conn.Close()
	}
}

func (db *DB) Conn() *pgxpool.Pool { return db.conn }
func (db *DB) DB() *DB             { return db }
