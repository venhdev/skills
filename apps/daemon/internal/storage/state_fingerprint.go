package storage

import (
	"context"
	"fmt"
	"strconv"
)

// StateFingerprinter returns a cheap fingerprint of the plans/tasks state. The
// gateway polls it to detect commits made by other processes (for example the
// stdio MCP server) that never pass through the daemon event hub.
type StateFingerprinter struct {
	db *DB
}

// StateFingerprint reads the revision counter maintained by database triggers.
// It is a single-row lookup, so the watcher can poll frequently without
// scanning the plans/tasks tables.
func (f *StateFingerprinter) StateFingerprint(ctx context.Context) (string, error) {
	var revision int64
	if err := f.db.QueryRow(ctx, `SELECT revision FROM state_revision WHERE id = 1`).Scan(&revision); err != nil {
		return "", fmt.Errorf("failed to read state revision: %w", err)
	}
	return strconv.FormatInt(revision, 10), nil
}
