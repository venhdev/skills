package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"zenfd/internal/domain"
)

// PlanRepository persists plans and the plan/task bundles written by authoring
// tools.
type PlanRepository struct {
	db *DB
}

// SavePlanWithTasks writes a plan and its tasks in a single atomic transaction.
// Existing tasks are patched (authoring metadata only): re-ingesting a dossier
// can never reset a task's status, claim, notes, or review history. The whole
// resulting graph is validated before any row is written.
func (r *PlanRepository) SavePlanWithTasks(ctx context.Context, plan domain.Plan, tasks []domain.Task) error {
	if plan.Slug == "" {
		return fmt.Errorf("plan slug cannot be empty")
	}

	now := time.Now().UTC()
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	plan.UpdatedAt = now

	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		planQuery := `
		INSERT INTO plans (slug, title, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(slug) DO UPDATE SET
			title = excluded.title,
			status = excluded.status,
			updated_at = excluded.updated_at
		`
		if _, err := tx.ExecContext(ctx, planQuery, plan.Slug, plan.Title, plan.Status, plan.CreatedAt, plan.UpdatedAt); err != nil {
			return fmt.Errorf("failed to upsert plan: %w", err)
		}

		graph, err := loadPlanDeps(ctx, tx, plan.Slug)
		if err != nil {
			return err
		}
		for i := range tasks {
			tasks[i].PlanID = plan.Slug
			graph[tasks[i].ID] = tasks[i].Dependencies
		}
		if err := validateCompleteGraph(graph); err != nil {
			return err
		}

		for _, task := range tasks {
			if err := patchTaskTx(ctx, tx, &task, now); err != nil {
				return err
			}
		}

		// Dossier re-ingest can add tasks whose prerequisites already finished
		// (or failed); derive their initial status instead of leaving PENDING.
		return resyncTaskStatuses(ctx, tx, plan.Slug, tasks, now)
	})
}

// GetPlan retrieves a plan by slug.
func (r *PlanRepository) GetPlan(ctx context.Context, slug string) (*domain.Plan, error) {
	query := `SELECT slug, title, status, archived_at, created_at, updated_at FROM plans WHERE slug = ?`
	row := r.db.QueryRow(ctx, query, slug)

	var p domain.Plan
	var archivedAt sql.NullTime
	err := row.Scan(&p.Slug, &p.Title, &p.Status, &archivedAt, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query plan: %w", err)
	}
	if archivedAt.Valid {
		p.ArchivedAt = &archivedAt.Time
	}

	return &p, nil
}

// ListPlans returns plans ordered by slug. Archived plans are filtered out
// unless includeArchived is true.
func (r *PlanRepository) ListPlans(ctx context.Context, includeArchived bool) ([]domain.Plan, error) {
	query := `SELECT slug, title, status, archived_at, created_at, updated_at FROM plans`
	if !includeArchived {
		query += ` WHERE archived_at IS NULL`
	}
	query += ` ORDER BY slug ASC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}
	defer rows.Close()

	var plans []domain.Plan
	for rows.Next() {
		var p domain.Plan
		var archivedAt sql.NullTime
		if err := rows.Scan(&p.Slug, &p.Title, &p.Status, &archivedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan plan: %w", err)
		}
		if archivedAt.Valid {
			p.ArchivedAt = &archivedAt.Time
		}
		plans = append(plans, p)
	}

	return plans, rows.Err()
}

// ArchivePlan soft-archives or restores a plan. Archived plans keep every row;
// they are only hidden from the default views.
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

// CreatePlan inserts or updates a single plan.
func (r *PlanRepository) CreatePlan(ctx context.Context, plan domain.Plan) error {
	now := time.Now().UTC()
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	plan.UpdatedAt = now
	if plan.Status == "" {
		plan.Status = "ACTIVE"
	}

	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		query := `
		INSERT INTO plans (slug, title, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(slug) DO UPDATE SET
			title = excluded.title,
			status = excluded.status,
			updated_at = excluded.updated_at
		`
		_, err := tx.ExecContext(ctx, query, plan.Slug, plan.Title, plan.Status, plan.CreatedAt, plan.UpdatedAt)
		return err
	})
}

// DeletePlan deletes a plan and cascades to its tasks and artifacts.
func (r *PlanRepository) DeletePlan(ctx context.Context, slug string) error {
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "DELETE FROM plans WHERE slug = ?", slug)
		if err != nil {
			return err
		}
		rows, err := affectedOrErr(res, "delete plan", slug)
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrNotFound
		}
		return nil
	})
}
