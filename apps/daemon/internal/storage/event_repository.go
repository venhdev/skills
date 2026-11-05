package storage

import (
	"context"
	"fmt"
	"time"

	"zenfd/internal/domain"
)

// EventRepository reads the trigger-produced activity log.
type EventRepository struct {
	db *DB
}

// ListEvents returns up to limit events newer than afterID in ascending order.
// With afterID zero it returns the most recent window, which is what the live
// log panel needs on first load and after a reconnect.
func (r *EventRepository) ListEvents(ctx context.Context, afterID int64, limit int) ([]domain.EventMessage, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, COALESCE(plan_id, ''), COALESCE(task_id, ''), type, payload, timestamp
		FROM (
			SELECT id, plan_id, task_id, type, payload, timestamp
			FROM events
			WHERE id > ?
			ORDER BY id DESC
			LIMIT ?
		)
		ORDER BY id ASC
	`, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	defer rows.Close()

	events := make([]domain.EventMessage, 0, limit)
	for rows.Next() {
		var (
			evt       domain.EventMessage
			timestamp time.Time
		)
		if err := rows.Scan(&evt.ID, &evt.PlanID, &evt.TaskID, &evt.Type, &evt.Payload, &timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		evt.Timestamp = timestamp
		events = append(events, evt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read events: %w", err)
	}
	return events, nil
}
