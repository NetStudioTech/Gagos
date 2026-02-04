package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgresBackend implements StorageBackend using PostgreSQL
type PostgresBackend struct {
	db  *sql.DB
	url string
}

// NewPostgresBackend creates a new PostgreSQL storage backend
func NewPostgresBackend(url string) *PostgresBackend {
	return &PostgresBackend{url: url}
}

func (p *PostgresBackend) Init() error {
	var err error
	p.db, err = sql.Open("postgres", p.url)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}

	// Configure connection pool
	p.db.SetMaxOpenConns(10)
	p.db.SetMaxIdleConns(5)
	p.db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := p.db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping postgres: %w", err)
	}

	// Create table for key-value storage
	_, err = p.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS gagos_kv (
			bucket VARCHAR(64) NOT NULL,
			key VARCHAR(255) NOT NULL,
			value BYTEA,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (bucket, key)
		);
		CREATE INDEX IF NOT EXISTS idx_gagos_kv_bucket ON gagos_kv(bucket);
	`)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	log.Info().Str("type", "postgres").Msg("Storage initialized")
	return nil
}

func (p *PostgresBackend) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *PostgresBackend) Type() string {
	return StorageTypePostgres
}

func (p *PostgresBackend) Set(bucket, key string, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := p.db.ExecContext(ctx, `
		INSERT INTO gagos_kv (bucket, key, value, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (bucket, key) DO UPDATE SET value = $3, updated_at = CURRENT_TIMESTAMP
	`, bucket, key, value)
	return err
}

func (p *PostgresBackend) Get(bucket, key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var value []byte
	err := p.db.QueryRowContext(ctx,
		"SELECT value FROM gagos_kv WHERE bucket = $1 AND key = $2",
		bucket, key,
	).Scan(&value)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return value, err
}

func (p *PostgresBackend) Delete(bucket, key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := p.db.ExecContext(ctx,
		"DELETE FROM gagos_kv WHERE bucket = $1 AND key = $2",
		bucket, key,
	)
	return err
}

func (p *PostgresBackend) List(bucket string) ([][]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := p.db.QueryContext(ctx,
		"SELECT value FROM gagos_kv WHERE bucket = $1 ORDER BY created_at DESC",
		bucket,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items [][]byte
	for rows.Next() {
		var value []byte
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		items = append(items, value)
	}
	return items, rows.Err()
}

func (p *PostgresBackend) ListKeys(bucket string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := p.db.QueryContext(ctx,
		"SELECT key FROM gagos_kv WHERE bucket = $1 ORDER BY created_at DESC",
		bucket,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// GetDB returns the underlying SQL database
func (p *PostgresBackend) GetDB() *sql.DB {
	return p.db
}
