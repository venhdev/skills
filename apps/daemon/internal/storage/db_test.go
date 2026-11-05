package storage_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"zenfd/internal/storage"
)

func TestDB_InitWAL(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_state.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Verify WAL Mode
	var journalMode string
	err = db.QueryRow(context.Background(), "PRAGMA journal_mode;").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Failed to query journal mode: %v", err)
	}
	if strings.ToUpper(journalMode) != "WAL" {
		t.Errorf("Expected journal_mode WAL, got: %s", journalMode)
	}

	// Verify Busy Timeout
	var busyTimeout int
	err = db.QueryRow(context.Background(), "PRAGMA busy_timeout;").Scan(&busyTimeout)
	if err != nil {
		t.Fatalf("Failed to query busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Errorf("Expected busy_timeout 5000, got: %d", busyTimeout)
	}
}

func TestDB_TablesCreated(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_state.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	expectedTables := []string{"plans", "tasks", "runs", "artifacts", "events"}
	for _, tableName := range expectedTables {
		var count int
		err := db.QueryRow(context.Background(), "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count)
		if err != nil {
			t.Fatalf("Failed to query table %s: %v", tableName, err)
		}
		if count != 1 {
			t.Errorf("Expected table %s to exist, count was %d", tableName, count)
		}
	}
}

func TestDB_ExecImmediate(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_state.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	now := time.Now().UTC()
	err = db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO plans (slug, title, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			"plan-1", "Plan 1", "ACTIVE", now, now)
		return err
	})
	if err != nil {
		t.Fatalf("ExecImmediate failed: %v", err)
	}

	// Query back
	var title string
	err = db.QueryRow(context.Background(), "SELECT title FROM plans WHERE slug = 'plan-1'").Scan(&title)
	if err != nil {
		t.Fatalf("Failed to query inserted plan: %v", err)
	}
	if title != "Plan 1" {
		t.Errorf("Expected title 'Plan 1', got '%s'", title)
	}
}

func TestDB_ExecImmediateCrossConnectionLock(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "cross_connection.db")

	db1, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db1: %v", err)
	}
	defer db1.Close()

	db2, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db2: %v", err)
	}
	defer db2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	if err := db1.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "INSERT INTO plans (slug, title, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
			"cross-plan", "Cross Plan", "ACTIVE", now, now)
		return err
	}); err != nil {
		t.Fatalf("Failed to seed plan: %v", err)
	}

	readStarted := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseFirst := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseFirst()

	firstErr := make(chan error, 1)
	go func() {
		firstErr <- db1.ExecImmediate(ctx, func(tx *sql.Tx) error {
			var title string
			if err := tx.QueryRowContext(ctx, "SELECT title FROM plans WHERE slug = ?", "cross-plan").Scan(&title); err != nil {
				return err
			}
			close(readStarted)
			<-release
			_, err := tx.ExecContext(ctx, "UPDATE plans SET title = ? WHERE slug = ?", "db1-title", "cross-plan")
			return err
		})
	}()

	select {
	case <-readStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("first transaction did not start")
	}

	secondErr := make(chan error, 1)
	go func() {
		secondErr <- db2.ExecImmediate(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, "UPDATE plans SET title = ? WHERE slug = ?", "db2-title", "cross-plan")
			return err
		})
	}()

	// With _txlock=immediate, db1 holds the SQLite reserved lock from BEGIN,
	// so db2 must wait even though db1 has not issued its UPDATE yet.
	select {
	case err := <-secondErr:
		t.Fatalf("second write committed before first transaction finished (immediate lock missing): %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	releaseFirst()
	if err := <-firstErr; err != nil {
		t.Fatalf("first transaction failed: %v", err)
	}
	if err := <-secondErr; err != nil {
		t.Fatalf("second transaction failed: %v", err)
	}
}
