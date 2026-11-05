package storage

import (
	"context"
	"database/sql"
	"time"

	"zenfd/internal/domain"
)

// EventLog persists audit events.
type EventLog struct {
	db *DB
}

// AppendEvent records an audit event in the database.
func (l *EventLog) AppendEvent(ctx context.Context, evt domain.EventMessage) error {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}

	query := `INSERT INTO events (run_id, task_id, type, payload, timestamp) VALUES (?, ?, ?, ?, ?)`
	return l.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, evt.RunID, evt.TaskID, evt.Type, evt.Payload, evt.Timestamp)
		return err
	})
}
