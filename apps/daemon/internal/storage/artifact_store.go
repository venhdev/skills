package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"zenfd/internal/domain"
)

// artifactSelectCols is the projection shared by every artifact read.
const artifactSelectCols = "id, task_id, role, name, type, file_path, mime_type, byte_size, sha256, created_at"

// ArtifactRepository persists artifact metadata following the claim-check
// pattern.
type ArtifactRepository struct {
	db *DB
}

// RecordArtifact saves artifact metadata to SQLite adhering to the Claim-Check pattern.
func (r *ArtifactRepository) RecordArtifact(ctx context.Context, artifact domain.Artifact) error {
	if artifact.ID == "" {
		return fmt.Errorf("artifact ID cannot be empty")
	}
	if !artifact.Type.IsValid() {
		return fmt.Errorf("invalid artifact type: %s", artifact.Type)
	}

	if artifact.CreatedAt.IsZero() {
		artifact.CreatedAt = time.Now().UTC()
	}
	if artifact.Role == "" {
		artifact.Role = domain.ArtifactRoleOutput
	}

	query := `
	INSERT INTO artifacts (id, task_id, role, name, type, file_path, mime_type, byte_size, sha256, created_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		if artifact.Role == domain.ArtifactRoleInput {
			if err := assertInputAllowed(ctx, tx, artifact.TaskID); err != nil {
				return err
			}
		}

		_, err := tx.ExecContext(ctx, query,
			artifact.ID, artifact.TaskID, artifact.Role, artifact.Name,
			string(artifact.Type), artifact.FilePath, artifact.MimeType,
			artifact.ByteSize, artifact.Sha256, artifact.CreatedAt)
		return err
	})
}

// assertInputAllowed gates context media to tasks that are still being prepared.
// Attaching an INPUT artifact to a RUNNING or finished task would be invisible
// to the ContextPacket that was already handed to the worker.
func assertInputAllowed(ctx context.Context, tx *sql.Tx, taskID string) error {
	var status string
	err := tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: task %s", ErrNotFound, taskID)
	}
	if err != nil {
		return fmt.Errorf("failed to load task %s: %w", taskID, err)
	}

	switch domain.TaskStatus(status) {
	case domain.StatusReady, domain.StatusPending:
		return nil
	default:
		return fmt.Errorf("%w: input media can only be attached while the task is READY or PENDING (task %s is %s)",
			ErrInvalidState, taskID, status)
	}
}

// GetArtifact retrieves artifact metadata by ID.
func (r *ArtifactRepository) GetArtifact(ctx context.Context, id string) (*domain.Artifact, error) {
	query := `SELECT ` + artifactSelectCols + ` FROM artifacts WHERE id = ?`
	row := r.db.QueryRow(ctx, query, id)

	var art domain.Artifact
	var typeStr string
	var name sql.NullString

	err := row.Scan(
		&art.ID, &art.TaskID, &art.Role, &name,
		&typeStr, &art.FilePath, &art.MimeType, &art.ByteSize,
		&art.Sha256, &art.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query artifact: %w", err)
	}

	if name.Valid {
		art.Name = name.String
	}
	art.Type = domain.ArtifactType(typeStr)
	return &art, nil
}

// GetArtifactsByTask retrieves all artifacts associated with a task, optionally filtered by role ("INPUT" | "OUTPUT").
func (r *ArtifactRepository) GetArtifactsByTask(ctx context.Context, taskID string, role string) ([]domain.Artifact, error) {
	var query string
	var args []any

	if role != "" {
		query = `SELECT ` + artifactSelectCols + `
		FROM artifacts
		WHERE task_id = ? AND role = ?
		ORDER BY created_at ASC`
		args = []any{taskID, strings.ToUpper(role)}
	} else {
		query = `SELECT ` + artifactSelectCols + `
		FROM artifacts
		WHERE task_id = ?
		ORDER BY created_at ASC`
		args = []any{taskID}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query artifacts for task %s: %w", taskID, err)
	}
	defer rows.Close()

	var artifacts []domain.Artifact
	for rows.Next() {
		var art domain.Artifact
		var typeStr string
		var name sql.NullString

		err := rows.Scan(
			&art.ID, &art.TaskID, &art.Role, &name,
			&typeStr, &art.FilePath, &art.MimeType, &art.ByteSize,
			&art.Sha256, &art.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if name.Valid {
			art.Name = name.String
		}
		art.Type = domain.ArtifactType(typeStr)
		artifacts = append(artifacts, art)
	}

	return artifacts, rows.Err()
}
