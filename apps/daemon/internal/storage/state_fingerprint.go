package storage

import (
	"context"
	"fmt"
)

// StateFingerprinter returns a cheap fingerprint of the plans/tasks state. The
// gateway polls it to detect commits made by other processes (for example the
// stdio MCP server) that never pass through the daemon event hub.
type StateFingerprinter struct {
	db *DB
}

// StateFingerprint summarizes plans and tasks counts plus their newest update.
func (f *StateFingerprinter) StateFingerprint(ctx context.Context) (string, error) {
	var plansCount, tasksCount int64
	var plansUpdated, tasksUpdated string
	err := f.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM plans),
			(SELECT COALESCE(MAX(updated_at), '') FROM plans),
			(SELECT COUNT(*) FROM tasks),
			(SELECT COALESCE(MAX(updated_at), '') FROM tasks)
	`).Scan(&plansCount, &plansUpdated, &tasksCount, &tasksUpdated)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d|%s|%d|%s", plansCount, plansUpdated, tasksCount, tasksUpdated), nil
}
