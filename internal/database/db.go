package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	// PostgreSQL driver.
	_ "github.com/lib/pq"
)

// DB wraps the database connection and provides query methods
type DB struct {
	conn   *sql.DB
	logger *slog.Logger
}

// New creates a new database connection with proper pool settings
func New(connStr string, logger *slog.Logger) (*DB, error) {
	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("database connection established")

	return &DB{
		conn:   conn,
		logger: logger,
	}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}

	return nil
}

// Conn returns the underlying database connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// Query executes a query with context timeout
func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db.logger.Debug("executing query", "query", query, "args", args)

	return db.conn.QueryContext(ctx, query, args...)
}

// QueryRow executes a query expected to return at most one row
func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db.logger.Debug("executing query row", "query", query, "args", args)

	return db.conn.QueryRowContext(ctx, query, args...)
}

// Exec executes a query that doesn't return rows
func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	db.logger.Debug("executing statement", "query", query, "args", args)

	return db.conn.ExecContext(ctx, query, args...)
}
