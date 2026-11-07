package storage

import (
	"context"
	"fmt"
	"time"

	"zenf/internal/domain"
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
		SELECT id, COALESCE(project_slug, ''), COALESCE(plan_slug, ''),
		       COALESCE(task_id, ''), entity_type, entity_id, type,
		       COALESCE(actor, ''), payload, timestamp
		FROM (
			SELECT id, project_slug, plan_slug, task_id, entity_type, entity_id,
			       type, actor, payload, timestamp
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
		if err := rows.Scan(
			&evt.ID, &evt.ProjectSlug, &evt.PlanSlug, &evt.TaskID,
			&evt.EntityType, &evt.EntityID, &evt.Type, &evt.Actor,
			&evt.Payload, &timestamp,
		); err != nil {
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

// LatestEventID returns the highest event id, or zero when the log is empty.
func (r *EventRepository) LatestEventID(ctx context.Context) (int64, error) {
	var id int64
	if err := r.db.QueryRow(ctx, "SELECT COALESCE(MAX(id), 0) FROM events").Scan(&id); err != nil {
		return 0, fmt.Errorf("failed to read latest event id: %w", err)
	}
	return id, nil
}
