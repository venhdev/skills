package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"zenf/internal/domain"
)

// PlanRepository persists plans and the atomic plan/task authoring bundle.
type PlanRepository struct {
	db *DB
}

const planSelectColumns = `slug, project_slug, title, status, archived_at, created_at, updated_at`

func scanPlan(s scanner) (*domain.Plan, error) {
	var p domain.Plan
	var status string
	var archivedAt sql.NullTime
	if err := s.Scan(&p.Slug, &p.ProjectSlug, &p.Title, &status, &archivedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	p.Status = domain.PlanStatus(status)
	if archivedAt.Valid {
		p.ArchivedAt = &archivedAt.Time
	}
	return &p, nil
}

// CreatePlanWithTasks writes a plan and its tasks in one atomic transaction.
// Re-ingesting an existing plan patches authoring metadata only; lifecycle,
// claim, notes, and review history are never overwritten.
func (r *PlanRepository) CreatePlanWithTasks(ctx context.Context, plan domain.Plan, tasks []domain.Task) error {
	now := time.Now().UTC()
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	plan.UpdatedAt = now
	if plan.Status == "" {
		plan.Status = domain.PlanStatusActive
	}
	if err := plan.Validate(); err != nil {
		return err
	}

	seen := make(map[string]bool, len(tasks))
	for i := range tasks {
		tasks[i].ProjectSlug = plan.ProjectSlug
		tasks[i].PlanSlug = plan.Slug
		if tasks[i].Status == "" {
			tasks[i].Status = domain.StatusPending
			if len(tasks[i].Dependencies) == 0 {
				tasks[i].Status = domain.StatusReady
			}
		}
		if err := tasks[i].Validate(); err != nil {
			return err
		}
		if seen[tasks[i].ID] {
			return fmt.Errorf("%w: duplicate task id %s in plan bundle", ErrInvalidGraph, tasks[i].ID)
		}
		seen[tasks[i].ID] = true
	}

	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		if _, err := getProjectTx(ctx, tx, plan.ProjectSlug); err != nil {
			return err
		}

		var existingProject string
		err := tx.QueryRowContext(ctx, "SELECT project_slug FROM plans WHERE slug = ?", plan.Slug).Scan(&existingProject)
		switch {
		case errors.Is(err, sql.ErrNoRows):
		case err != nil:
			return fmt.Errorf("failed to inspect plan %s: %w", plan.Slug, err)
		case existingProject != plan.ProjectSlug:
			return fmt.Errorf("%w: plan %s already belongs to project %s", ErrInvalidState, plan.Slug, existingProject)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO plans (slug, project_slug, title, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(slug) DO UPDATE SET
				title = excluded.title,
				status = excluded.status,
				updated_at = excluded.updated_at`,
			plan.Slug, plan.ProjectSlug, plan.Title, string(plan.Status), plan.CreatedAt, plan.UpdatedAt); err != nil {
			return fmt.Errorf("failed to upsert plan %s: %w", plan.Slug, err)
		}

		for i := range tasks {
			if err := upsertTaskAuthoringTx(ctx, tx, &tasks[i], now); err != nil {
				return err
			}
		}
		if err := validateCompleteGraphTx(ctx, tx, plan.ProjectSlug, plan.Slug); err != nil {
			return err
		}
		return resyncContainerStatuses(ctx, tx, plan.ProjectSlug, plan.Slug, tasks, now)
	})
}

func (r *PlanRepository) GetPlan(ctx context.Context, slug string) (*domain.Plan, error) {
	row := r.db.QueryRow(ctx, "SELECT "+planSelectColumns+" FROM plans WHERE slug = ?", slug)
	plan, err := scanPlan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: plan %s", ErrNotFound, slug)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query plan %s: %w", slug, err)
	}
	return plan, nil
}

func (r *PlanRepository) ListPlans(ctx context.Context, projectSlug string, includeArchived bool) ([]domain.Plan, error) {
	query := "SELECT " + planSelectColumns + " FROM plans"
	var conditions []string
	var args []any
	if projectSlug != "" {
		conditions = append(conditions, "project_slug = ?")
		args = append(args, projectSlug)
	}
	if !includeArchived {
		conditions = append(conditions, "archived_at IS NULL")
	}
	if len(conditions) > 0 {
		query += " WHERE " + joinConditions(conditions)
	}
	query += " ORDER BY slug ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}
	defer rows.Close()

	plans := []domain.Plan{}
	for rows.Next() {
		plan, err := scanPlan(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plan: %w", err)
		}
		plans = append(plans, *plan)
	}
	return plans, rows.Err()
}

func (r *PlanRepository) UpdatePlan(ctx context.Context, slug string, patch domain.PlanPatch) error {
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		plan, err := getPlanTx(ctx, tx, slug)
		if err != nil {
			return err
		}
		if patch.Title != nil {
			plan.Title = *patch.Title
		}
		if patch.Status != nil {
			plan.Status = *patch.Status
		}
		if err := plan.Validate(); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE plans SET title = ?, status = ?, updated_at = ? WHERE slug = ?`,
			plan.Title, string(plan.Status), now, slug)
		if err != nil {
			return fmt.Errorf("failed to update plan %s: %w", slug, err)
		}
		return nil
	})
}

func (r *PlanRepository) ArchivePlan(ctx context.Context, slug string, archived bool) error {
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		var archivedAt any
		if archived {
			archivedAt = now
		}
		res, err := tx.ExecContext(ctx,
			"UPDATE plans SET archived_at = ?, updated_at = ? WHERE slug = ?", archivedAt, now, slug)
		if err != nil {
			return fmt.Errorf("failed to archive plan %s: %w", slug, err)
		}
		rows, err := affectedOrErr(res, "archive plan", slug)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: plan %s", ErrNotFound, slug)
		}
		return nil
	})
}

func (r *PlanRepository) DeletePlan(ctx context.Context, slug string) error {
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "DELETE FROM plans WHERE slug = ?", slug)
		if err != nil {
			return fmt.Errorf("failed to delete plan %s: %w", slug, err)
		}
		rows, err := affectedOrErr(res, "delete plan", slug)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: plan %s", ErrNotFound, slug)
		}
		return nil
	})
}

func getPlanTx(ctx context.Context, tx *sql.Tx, slug string) (*domain.Plan, error) {
	row := tx.QueryRowContext(ctx, "SELECT "+planSelectColumns+" FROM plans WHERE slug = ?", slug)
	plan, err := scanPlan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: plan %s", ErrNotFound, slug)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query plan %s: %w", slug, err)
	}
	return plan, nil
}
