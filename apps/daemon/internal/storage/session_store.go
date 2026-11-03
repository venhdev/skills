package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"zenfd/internal/domain"
)

// RegisterSession registers or updates an agent session.
func (s *Store) RegisterSession(ctx context.Context, session domain.AgentSession) error {
	now := time.Now().UTC()
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}
	session.LastBeatAt = now
	if session.Status == "" {
		session.Status = "ACTIVE"
	}
	if session.Role == "" {
		session.Role = "SUBAGENT"
	}

	tasksJSON, _ := json.Marshal(session.ClaimedTasks)

	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		query := `
		INSERT INTO agent_sessions (
			session_id, parent_session_id, role, name, pid, status, claimed_tasks, created_at, last_beat_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			parent_session_id = excluded.parent_session_id,
			role = excluded.role,
			name = excluded.name,
			pid = excluded.pid,
			status = excluded.status,
			claimed_tasks = excluded.claimed_tasks,
			last_beat_at = excluded.last_beat_at
		`
		_, err := tx.ExecContext(ctx, query,
			session.SessionID, session.ParentSessionID, session.Role, session.Name,
			session.PID, session.Status, string(tasksJSON), session.CreatedAt, session.LastBeatAt,
		)
		return err
	})
}

// ListSessions returns all agent sessions, ordered by last_beat_at DESC.
func (s *Store) ListSessions(ctx context.Context) ([]domain.AgentSession, error) {
	query := `
	SELECT session_id, parent_session_id, role, name, pid, status, claimed_tasks, created_at, last_beat_at
	FROM agent_sessions
	ORDER BY last_beat_at DESC
	`
	rows, err := s.db.sqlDB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query agent sessions: %w", err)
	}
	defer rows.Close()

	var sessions []domain.AgentSession
	for rows.Next() {
		var sess domain.AgentSession
		var tasksJSON string
		err := rows.Scan(
			&sess.SessionID, &sess.ParentSessionID, &sess.Role, &sess.Name,
			&sess.PID, &sess.Status, &tasksJSON, &sess.CreatedAt, &sess.LastBeatAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent session: %w", err)
		}
		_ = json.Unmarshal([]byte(tasksJSON), &sess.ClaimedTasks)
		if sess.ClaimedTasks == nil {
			sess.ClaimedTasks = []string{}
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// HeartbeatSession updates the last heartbeat timestamp for a session.
func (s *Store) HeartbeatSession(ctx context.Context, sessionID string) error {
	now := time.Now().UTC()
	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "UPDATE agent_sessions SET last_beat_at = ?, status = 'ACTIVE' WHERE session_id = ?", now, sessionID)
		if err != nil {
			return err
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return ErrNotFound
		}
		return nil
	})
}
