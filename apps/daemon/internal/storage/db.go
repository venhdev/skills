package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ncruces/go-sqlite3/driver"
	"github.com/ncruces/go-sqlite3/ext/fts5"
)

// DB encapsulates the connection pool and migration runner for SQLite WAL.
type DB struct {
	sqlDB   *sql.DB
	writeMu sync.Mutex // Serializes write transactions to prevent lock contention
}

// Open creates parent directories, opens the SQLite database with WAL pragmas, and migrates tables.
func Open(dbPath string) (*DB, error) {
	if dbPath != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create db directory: %w", err)
		}
	}

	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_txlock=immediate", dbPath)
	// fts5 is registered so legacy FTS tables can be dropped during migration.
	conn, err := driver.Open(dsn, fts5.Register)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Connection pool tuning for SQLite WAL
	conn.SetMaxOpenConns(4)
	conn.SetMaxIdleConns(4)
	conn.SetConnMaxLifetime(0)

	// Verify connection liveness
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	// Verify WAL mode
	var journalMode string
	if err := conn.QueryRow("PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to verify journal mode: %w", err)
	}
	if strings.ToUpper(journalMode) != "WAL" && dbPath != ":memory:" {
		_ = conn.Close()
		return nil, fmt.Errorf("expected WAL journal mode, got: %s", journalMode)
	}

	ctx := context.Background()

	// Apply schema migrations
	if _, err := conn.ExecContext(ctx, schemaDDL); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to apply schema migrations: %w", err)
	}

	// Drop legacy docs/specs/snapshot/session tables (idempotent).
	if _, err := conn.ExecContext(ctx, dropLegacyDDL); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to drop legacy tables: %w", err)
	}

	// Add columns that a legacy database is missing.
	if err := applyIncrementalMigrations(ctx, conn); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to apply incremental migrations: %w", err)
	}

	return &DB{sqlDB: conn}, nil
}

// Query executes a read-only query that returns rows without acquiring writeMu.
func (d *DB) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.sqlDB.QueryContext(ctx, query, args...)
}

// QueryRow executes a read-only query that returns at most one row without acquiring writeMu.
func (d *DB) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return d.sqlDB.QueryRowContext(ctx, query, args...)
}

// ExecImmediate executes a write operation inside an immediate transaction,
// serializing access via writeMu to prevent in-process writer contention.
// The DSN sets _txlock=immediate, so BeginTx with LevelDefault maps to
// BEGIN IMMEDIATE and acquires the SQLite reserved lock before the closure runs.
func (d *DB) ExecImmediate(ctx context.Context, fn func(tx *sql.Tx) error) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	tx, err := d.sqlDB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelDefault})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Close gracefully terminates the database connection pool.
func (d *DB) Close() error {
	return d.sqlDB.Close()
}
