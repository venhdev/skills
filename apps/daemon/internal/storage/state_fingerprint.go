package storage

import (
	"context"
	"strconv"
)

// StateFingerprinter returns a cheap fingerprint of the whole tracking plane.
// The gateway polls it to detect commits made by another process (for example
// the stdio MCP server) that never pass through the daemon event hub.
type StateFingerprinter struct {
	db *DB
}

// StateFingerprint reads the highest event id. Every tracked mutation appends an
// event through database triggers, so a single indexed MAX() is enough and the
// watcher can poll frequently without scanning entity tables.
func (f *StateFingerprinter) StateFingerprint(ctx context.Context) (string, error) {
	var id int64
	if err := f.db.QueryRow(ctx, "SELECT COALESCE(MAX(id), 0) FROM events").Scan(&id); err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}
