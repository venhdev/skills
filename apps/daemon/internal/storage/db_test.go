package storage_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
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
	err = db.RawDB().QueryRow("PRAGMA journal_mode;").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Failed to query journal mode: %v", err)
	}
	if strings.ToUpper(journalMode) != "WAL" {
		t.Errorf("Expected journal_mode WAL, got: %s", journalMode)
	}

	// Verify Busy Timeout
	var busyTimeout int
	err = db.RawDB().QueryRow("PRAGMA busy_timeout;").Scan(&busyTimeout)
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
		err := db.RawDB().QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&count)
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
	err = db.RawDB().QueryRow("SELECT title FROM plans WHERE slug = 'plan-1'").Scan(&title)
	if err != nil {
		t.Fatalf("Failed to query inserted plan: %v", err)
	}
	if title != "Plan 1" {
		t.Errorf("Expected title 'Plan 1', got '%s'", title)
	}
}
