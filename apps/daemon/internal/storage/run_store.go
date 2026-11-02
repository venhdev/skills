package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"zenfd/internal/domain"
)

// CreateRun registers a new execution attempt of a task.
func (s *Store) CreateRun(ctx context.Context, run domain.Run) error {
	if run.ID == "" {
		return fmt.Errorf("run ID cannot be empty")
	}
	if run.TaskID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	now := time.Now().UTC()
	if run.StartedAt.IsZero() {
		run.StartedAt = now
	}
	if run.LastBeatAt.IsZero() {
		run.LastBeatAt = now
	}

	query := `
	INSERT INTO runs (id, task_id, repo_path, pid, pgid, status, log_path, exit_code, started_at, finished_at, last_beat_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			run.ID, run.TaskID, run.RepoPath, run.PID, run.PGID,
			string(run.Status), run.LogPath, run.ExitCode,
			run.StartedAt, run.FinishedAt, run.LastBeatAt)
		return err
	})
}

// UpdateRunHeartbeat updates the last heartbeat timestamp for an active run.
func (s *Store) UpdateRunHeartbeat(ctx context.Context, runID string, now time.Time) error {
	query := `UPDATE runs SET last_beat_at = ? WHERE id = ? AND status = 'RUNNING'`
	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, query, now.UTC(), runID)
		if err != nil {
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: active run %s not found", ErrNotFound, runID)
		}
		return nil
	})
}

// FinishRun marks a run as completed or failed with an exit code and finished timestamp.
func (s *Store) FinishRun(ctx context.Context, runID string, status domain.TaskStatus, exitCode int) error {
	now := time.Now().UTC()
	query := `UPDATE runs SET status = ?, exit_code = ?, finished_at = ? WHERE id = ?`

	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, query, string(status), exitCode, now, runID)
		if err != nil {
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: run %s not found", ErrNotFound, runID)
		}
		return nil
	})
}

// GetActiveRuns returns all currently running runs for liveness audit.
func (s *Store) GetActiveRuns(ctx context.Context) ([]domain.Run, error) {
	query := `
	SELECT id, task_id, repo_path, pid, pgid, status, log_path, exit_code, started_at, finished_at, last_beat_at
	FROM runs
	WHERE status = 'RUNNING'
	`
	rows, err := s.db.sqlDB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query active runs: %w", err)
	}
	defer rows.Close()

	var runs []domain.Run
	for rows.Next() {
		var r domain.Run
		var statusStr string
		err := rows.Scan(
			&r.ID, &r.TaskID, &r.RepoPath, &r.PID, &r.PGID,
			&statusStr, &r.LogPath, &r.ExitCode,
			&r.StartedAt, &r.FinishedAt, &r.LastBeatAt,
		)
		if err != nil {
			return nil, err
		}
		r.Status = domain.TaskStatus(statusStr)
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// RecordArtifact saves artifact metadata to SQLite adhering to the Claim-Check pattern.
func (s *Store) RecordArtifact(ctx context.Context, artifact domain.Artifact) error {
	if artifact.ID == "" {
		return fmt.Errorf("artifact ID cannot be empty")
	}
	if !artifact.Type.IsValid() {
		return fmt.Errorf("invalid artifact type: %s", artifact.Type)
	}

	if artifact.CreatedAt.IsZero() {
		artifact.CreatedAt = time.Now().UTC()
	}

	query := `
	INSERT INTO artifacts (id, run_id, task_id, type, file_path, mime_type, byte_size, sha256, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			artifact.ID, artifact.RunID, artifact.TaskID, string(artifact.Type),
			artifact.FilePath, artifact.MimeType, artifact.ByteSize,
			artifact.Sha256, artifact.CreatedAt)
		return err
	})
}

// GetArtifact retrieves artifact metadata by ID.
func (s *Store) GetArtifact(ctx context.Context, id string) (*domain.Artifact, error) {
	query := `
	SELECT id, run_id, task_id, type, file_path, mime_type, byte_size, sha256, created_at
	FROM artifacts
	WHERE id = ?
	`
	row := s.db.sqlDB.QueryRowContext(ctx, query, id)

	var art domain.Artifact
	var typeStr string
	err := row.Scan(
		&art.ID, &art.RunID, &art.TaskID, &typeStr,
		&art.FilePath, &art.MimeType, &art.ByteSize,
		&art.Sha256, &art.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query artifact: %w", err)
	}

	art.Type = domain.ArtifactType(typeStr)
	return &art, nil
}

// AppendEvent records an audit event in the database.
func (s *Store) AppendEvent(ctx context.Context, evt domain.EventMessage) error {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}

	query := `INSERT INTO events (run_id, task_id, type, payload, timestamp) VALUES (?, ?, ?, ?, ?)`
	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query, evt.RunID, evt.TaskID, evt.Type, evt.Payload, evt.Timestamp)
		return err
	})
}
