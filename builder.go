package orm_go

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBBuilder is the entry point for building a DB connection.
// Stage 1: Must provide DSN first.
type DBBuilder interface {
	WithDSN(dsn string) DBBuilderWithDSN
}

// DBBuilderWithDSN is reached after DSN is set.
// Stage 2: Must provide Context next.
type DBBuilderWithDSN interface {
	WithContext(ctx context.Context) DBBuilderReady
}

// DBBuilderReady is reached after all required fields are set.
// Stage 3: Can optionally configure pool settings, then Connect.
type DBBuilderReady interface {
	WithPoolSize(size int) DBBuilderReady
	WithConnTimeout(d time.Duration) DBBuilderReady
	Connect() (*DB, error)
}

// NewBuilder creates a new DB builder starting at Stage 1.
func NewBuilder() DBBuilder {
	return &dbBuilder{}
}

// dbBuilder is the unexported concrete builder implementing all stages.
type dbBuilder struct {
	dsn         string
	ctx         context.Context
	poolSize    int
	connTimeout time.Duration
}

// WithDSN sets the database connection string and advances to Stage 2.
func (b *dbBuilder) WithDSN(dsn string) DBBuilderWithDSN {
	b.dsn = dsn
	return b
}

// WithContext sets the context and advances to Stage 3.
func (b *dbBuilder) WithContext(ctx context.Context) DBBuilderReady {
	b.ctx = ctx
	return b
}

// WithPoolSize sets the connection pool size (optional).
func (b *dbBuilder) WithPoolSize(size int) DBBuilderReady {
	b.poolSize = size
	return b
}

// WithConnTimeout sets the connection timeout (optional).
func (b *dbBuilder) WithConnTimeout(d time.Duration) DBBuilderReady {
	b.connTimeout = d
	return b
}

// Connect creates the DB instance with the configured settings.
func (b *dbBuilder) Connect() (*DB, error) {
	config, err := pgxpool.ParseConfig(b.dsn)
	if err != nil {
		return nil, err
	}

	if b.poolSize > 0 {
		config.MaxConns = int32(b.poolSize)
	}
	if b.connTimeout > 0 {
		config.ConnConfig.ConnectTimeout = b.connTimeout
	}

	pool, err := pgxpool.NewWithConfig(b.ctx, config)
	if err != nil {
		return nil, err
	}

	db := &DB{
		Config: &Config{
			Dsn:  b.dsn,
			Ctx:  b.ctx,
			conn: pool,
		},
	}

	return db, nil
}
