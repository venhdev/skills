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

		for _, task := range tasks {
			if err := upsertTaskTx(ctx, tx, &task, plan.Slug, now); err != nil {
				return err
			}
		}

		return nil
	})
}

// GetPlan retrieves a plan by slug.
func (r *PlanRepository) GetPlan(ctx context.Context, slug string) (*domain.Plan, error) {
	query := `SELECT slug, title, status, created_at, updated_at FROM plans WHERE slug = ?`
	row := r.db.QueryRow(ctx, query, slug)

	var p domain.Plan
	err := row.Scan(&p.Slug, &p.Title, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query plan: %w", err)
	}

	return &p, nil
}

// ListPlans returns every plan in the plans table ordered by slug. Unlike
// deriving plans from tasks, this includes plans that have no tasks yet.
func (r *PlanRepository) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	query := `SELECT slug, title, status, created_at, updated_at FROM plans ORDER BY slug ASC`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}
	defer rows.Close()

	var plans []domain.Plan
	for rows.Next() {
		var p domain.Plan
		if err := rows.Scan(&p.Slug, &p.Title, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan plan: %w", err)
		}
		plans = append(plans, p)
	}

	return plans, rows.Err()
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

// DeletePlan deletes a plan and cascades to its tasks, runs, and artifacts.
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
