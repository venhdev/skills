package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"zenf/internal/domain"
)

// ProjectRepository persists the top-level working environments.
type ProjectRepository struct {
	db *DB
}

const projectSelectColumns = `slug, name, root_path, description, status, created_at, updated_at`

func scanProject(s scanner) (*domain.Project, error) {
	var p domain.Project
	var status string
	if err := s.Scan(&p.Slug, &p.Name, &p.RootPath, &p.Description, &status, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	p.Status = domain.ProjectStatus(status)
	return &p, nil
}

func (r *ProjectRepository) CreateProject(ctx context.Context, project domain.Project) error {
	now := time.Now().UTC()
	if project.CreatedAt.IsZero() {
		project.CreatedAt = now
	}
	project.UpdatedAt = now
	if project.Status == "" {
		project.Status = domain.ProjectStatusActive
	}
	if err := project.Validate(); err != nil {
		return err
	}

	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO projects (slug, name, root_path, description, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			project.Slug, project.Name, project.RootPath, project.Description,
			string(project.Status), project.CreatedAt, project.UpdatedAt)
		if err != nil {
			if isUniqueViolation(err) {
				return fmt.Errorf("%w: project %s", ErrAlreadyExists, project.Slug)
			}
			return fmt.Errorf("failed to create project %s: %w", project.Slug, err)
		}
		return nil
	})
}

func (r *ProjectRepository) GetProject(ctx context.Context, slug string) (*domain.Project, error) {
	row := r.db.QueryRow(ctx, "SELECT "+projectSelectColumns+" FROM projects WHERE slug = ?", slug)
	project, err := scanProject(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: project %s", ErrNotFound, slug)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query project %s: %w", slug, err)
	}
	return project, nil
}

func (r *ProjectRepository) ListProjects(ctx context.Context, includeArchived bool) ([]domain.Project, error) {
	query := "SELECT " + projectSelectColumns + " FROM projects"
	if !includeArchived {
		query += " WHERE status <> 'ARCHIVED'"
	}
	query += " ORDER BY slug ASC"

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	projects := []domain.Project{}
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, *project)
	}
	return projects, rows.Err()
}

func (r *ProjectRepository) UpdateProject(ctx context.Context, slug string, patch domain.ProjectPatch) error {
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		project, err := getProjectTx(ctx, tx, slug)
		if err != nil {
			return err
		}
		if patch.Name != nil {
			project.Name = *patch.Name
		}
		if patch.RootPath != nil {
			project.RootPath = *patch.RootPath
		}
		if patch.Description != nil {
			project.Description = *patch.Description
		}
		if patch.Status != nil {
			project.Status = *patch.Status
		}
		if err := project.Validate(); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE projects SET name = ?, root_path = ?, description = ?, status = ?, updated_at = ?
			WHERE slug = ?`,
			project.Name, project.RootPath, project.Description, string(project.Status), now, slug)
		if err != nil {
			return fmt.Errorf("failed to update project %s: %w", slug, err)
		}
		return nil
	})
}

func (r *ProjectRepository) ArchiveProject(ctx context.Context, slug string, archived bool) error {
	status := domain.ProjectStatusActive
	if archived {
		status = domain.ProjectStatusArchived
	}
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			"UPDATE projects SET status = ?, updated_at = ? WHERE slug = ?", string(status), now, slug)
		if err != nil {
			return fmt.Errorf("failed to archive project %s: %w", slug, err)
		}
		rows, err := affectedOrErr(res, "archive project", slug)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: project %s", ErrNotFound, slug)
		}
		return nil
	})
}

func (r *ProjectRepository) DeleteProject(ctx context.Context, slug string) error {
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "DELETE FROM projects WHERE slug = ?", slug)
		if err != nil {
			return fmt.Errorf("failed to delete project %s: %w", slug, err)
		}
		rows, err := affectedOrErr(res, "delete project", slug)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: project %s", ErrNotFound, slug)
		}
		return nil
	})
}

func getProjectTx(ctx context.Context, tx *sql.Tx, slug string) (*domain.Project, error) {
	row := tx.QueryRowContext(ctx, "SELECT "+projectSelectColumns+" FROM projects WHERE slug = ?", slug)
	project, err := scanProject(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: project %s", ErrNotFound, slug)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query project %s: %w", slug, err)
	}
	return project, nil
}
