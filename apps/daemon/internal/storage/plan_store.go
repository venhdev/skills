package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"zenfd/internal/domain"
)

var (
	ErrNotFound       = errors.New("entity not found")
	ErrInvalidState   = errors.New("invalid state transition")
	ErrConflictStatus = errors.New("task status conflict during transition")
)

// Store implements domain.StorageEngine using the SQLite WAL DB.
type Store struct {
	db *DB
}

// NewStore creates a new domain.StorageEngine backed by the SQLite WAL DB.
func NewStore(db *DB) *Store {
	return &Store{db: db}
}

// Close safely closes the underlying SQLite database connection pool.
func (s *Store) Close() error {
	return s.db.Close()
}

// SavePlanWithTasks writes a plan and its tasks in a single atomic transaction.
func (s *Store) SavePlanWithTasks(ctx context.Context, plan domain.Plan, tasks []domain.Task) error {
	if plan.Slug == "" {
		return fmt.Errorf("plan slug cannot be empty")
	}

	now := time.Now().UTC()
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	plan.UpdatedAt = now

	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		// Upsert plan
		planQuery := `
		INSERT INTO plans (slug, title, repo_path, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(slug) DO UPDATE SET
			title = excluded.title,
			repo_path = excluded.repo_path,
			status = excluded.status,
			updated_at = excluded.updated_at
		`
		_, err := tx.ExecContext(ctx, planQuery, plan.Slug, plan.Title, plan.RepoPath, plan.Status, plan.CreatedAt, plan.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to upsert plan: %w", err)
		}

		// Insert or update tasks
		taskQuery := `
		INSERT INTO tasks (id, plan_id, repo_path, title, dependencies, harness, skill_command, worktree_path, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			dependencies = excluded.dependencies,
			harness = excluded.harness,
			skill_command = excluded.skill_command,
			worktree_path = excluded.worktree_path,
			status = excluded.status,
			updated_at = excluded.updated_at
		`

		for _, task := range tasks {
			if err := task.Validate(); err != nil {
				return err
			}
			if task.CreatedAt.IsZero() {
				task.CreatedAt = now
			}
			task.UpdatedAt = now

			depsJSON, err := json.Marshal(task.Dependencies)
			if err != nil {
				return fmt.Errorf("failed to marshal dependencies for task %s: %w", task.ID, err)
			}

			_, err = tx.ExecContext(ctx, taskQuery,
				task.ID, plan.Slug, task.RepoPath, task.Title, string(depsJSON),
				task.Harness, task.SkillCommand, task.WorktreePath, string(task.Status),
				task.CreatedAt, task.UpdatedAt)
			if err != nil {
				return fmt.Errorf("failed to upsert task %s: %w", task.ID, err)
			}
		}

		return nil
	})
}

// GetPlan retrieves a plan by slug.
func (s *Store) GetPlan(ctx context.Context, slug string) (*domain.Plan, error) {
	query := `SELECT slug, title, repo_path, status, created_at, updated_at FROM plans WHERE slug = ?`
	row := s.db.sqlDB.QueryRowContext(ctx, query, slug)

	var p domain.Plan
	err := row.Scan(&p.Slug, &p.Title, &p.RepoPath, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query plan: %w", err)
	}

	return &p, nil
}

// GetTasksByPlan retrieves all tasks belonging to a plan.
func (s *Store) GetTasksByPlan(ctx context.Context, planSlug string) ([]domain.Task, error) {
	query := `
	SELECT id, plan_id, repo_path, title, dependencies, harness, skill_command, worktree_path, status, created_at, updated_at
	FROM tasks
	WHERE plan_id = ?
	ORDER BY id ASC
	`
	rows, err := s.db.sqlDB.QueryContext(ctx, query, planSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by plan: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		var t domain.Task
		var depsJSON string
		var statusStr string

		err := rows.Scan(
			&t.ID, &t.PlanID, &t.RepoPath, &t.Title, &depsJSON,
			&t.Harness, &t.SkillCommand, &t.WorktreePath, &statusStr,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		t.Status = domain.TaskStatus(statusStr)
		if err := json.Unmarshal([]byte(depsJSON), &t.Dependencies); err != nil {
			t.Dependencies = []string{}
		}

		tasks = append(tasks, t)
	}

	return tasks, rows.Err()
}

// GetReadyTasks returns tasks whose status is READY, filtered optionally by repoPath.
func (s *Store) GetReadyTasks(ctx context.Context, repoPath string, limit int) ([]domain.Task, error) {
	if limit <= 0 {
		limit = 10
	}

	var query string
	var args []any

	if repoPath != "" {
		query = `
		SELECT id, plan_id, repo_path, title, dependencies, harness, skill_command, worktree_path, status, created_at, updated_at
		FROM tasks
		WHERE status = 'READY' AND repo_path = ?
		ORDER BY created_at ASC
		LIMIT ?
		`
		args = []any{repoPath, limit}
	} else {
		query = `
		SELECT id, plan_id, repo_path, title, dependencies, harness, skill_command, worktree_path, status, created_at, updated_at
		FROM tasks
		WHERE status = 'READY'
		ORDER BY created_at ASC
		LIMIT ?
		`
		args = []any{limit}
	}

	rows, err := s.db.sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query ready tasks: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		var t domain.Task
		var depsJSON string
		var statusStr string

		err := rows.Scan(
			&t.ID, &t.PlanID, &t.RepoPath, &t.Title, &depsJSON,
			&t.Harness, &t.SkillCommand, &t.WorktreePath, &statusStr,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ready task: %w", err)
		}

		t.Status = domain.TaskStatus(statusStr)
		_ = json.Unmarshal([]byte(depsJSON), &t.Dependencies)
		tasks = append(tasks, t)
	}

	return tasks, rows.Err()
}

// ListTasks returns tasks filtered optionally by repoPath and/or status.
func (s *Store) ListTasks(ctx context.Context, repoPath string, status domain.TaskStatus) ([]domain.Task, error) {
	var query string
	var args []any

	if repoPath != "" && status != "" {
		query = `
		SELECT id, plan_id, repo_path, title, dependencies, harness, skill_command, worktree_path, status, created_at, updated_at
		FROM tasks
		WHERE repo_path = ? AND status = ?
		ORDER BY created_at ASC
		`
		args = []any{repoPath, string(status)}
	} else if repoPath != "" {
		query = `
		SELECT id, plan_id, repo_path, title, dependencies, harness, skill_command, worktree_path, status, created_at, updated_at
		FROM tasks
		WHERE repo_path = ?
		ORDER BY created_at ASC
		`
		args = []any{repoPath}
	} else if status != "" {
		query = `
		SELECT id, plan_id, repo_path, title, dependencies, harness, skill_command, worktree_path, status, created_at, updated_at
		FROM tasks
		WHERE status = ?
		ORDER BY created_at ASC
		`
		args = []any{string(status)}
	} else {
		query = `
		SELECT id, plan_id, repo_path, title, dependencies, harness, skill_command, worktree_path, status, created_at, updated_at
		FROM tasks
		ORDER BY created_at ASC
		`
	}

	rows, err := s.db.sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		var t domain.Task
		var depsJSON string
		var statusStr string

		err := rows.Scan(
			&t.ID, &t.PlanID, &t.RepoPath, &t.Title, &depsJSON,
			&t.Harness, &t.SkillCommand, &t.WorktreePath, &statusStr,
			&t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		t.Status = domain.TaskStatus(statusStr)
		_ = json.Unmarshal([]byte(depsJSON), &t.Dependencies)
		tasks = append(tasks, t)
	}

	return tasks, rows.Err()
}

// UpdateTaskStatus atomically transitions task status and automatically resolves DAG dependencies.
func (s *Store) UpdateTaskStatus(ctx context.Context, taskID string, from domain.TaskStatus, to domain.TaskStatus) error {
	if !from.CanTransitionTo(to) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidState, from, to)
	}

	now := time.Now().UTC()

	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		// Update target task status
		res, err := tx.ExecContext(ctx, "UPDATE tasks SET status = ?, updated_at = ? WHERE id = ? AND status = ?",
			string(to), now, taskID, string(from))
		if err != nil {
			return fmt.Errorf("failed to update task status: %w", err)
		}

		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: task %s is not in status %s", ErrConflictStatus, taskID, from)
		}

		// If task has COMPLETED, check downstream tasks in the same plan and unlock them if all dependencies are satisfied!
		if to == domain.StatusCompleted {
			var planID string
			err := tx.QueryRowContext(ctx, "SELECT plan_id FROM tasks WHERE id = ?", taskID).Scan(&planID)
			if err != nil {
				return err
			}

			// Query all PENDING tasks in this plan
			rows, err := tx.QueryContext(ctx, "SELECT id, dependencies FROM tasks WHERE plan_id = ? AND status = 'PENDING'", planID)
			if err != nil {
				return err
			}
			defer rows.Close()

			type pendingTask struct {
				id   string
				deps []string
			}
			var pendingList []pendingTask

			for rows.Next() {
				var pt pendingTask
				var depsJSON string
				if err := rows.Scan(&pt.id, &depsJSON); err == nil {
					_ = json.Unmarshal([]byte(depsJSON), &pt.deps)
					pendingList = append(pendingList, pt)
				}
			}
			rows.Close()

			// For each pending task, verify if ALL dependencies are COMPLETED
			for _, pt := range pendingList {
				allCompleted := true
				for _, depID := range pt.deps {
					var depStatus string
					err := tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", depID).Scan(&depStatus)
					if err != nil || domain.TaskStatus(depStatus) != domain.StatusCompleted {
						allCompleted = false
						break
					}
				}

				if allCompleted {
					_, err := tx.ExecContext(ctx, "UPDATE tasks SET status = 'READY', updated_at = ? WHERE id = ?", now, pt.id)
					if err != nil {
						return fmt.Errorf("failed to unlock ready task %s: %w", pt.id, err)
					}
				}
			}
		}

		// If task has FAILED, cascade block downstream dependent tasks (Fast-Fail branch)
		if to == domain.StatusFailed {
			var planID string
			_ = tx.QueryRowContext(ctx, "SELECT plan_id FROM tasks WHERE id = ?", taskID).Scan(&planID)
			if planID != "" {
				// Query all PENDING tasks in this plan and block those that depend on this failed task
				rows, err := tx.QueryContext(ctx, "SELECT id, dependencies FROM tasks WHERE plan_id = ? AND status = 'PENDING'", planID)
				if err == nil {
					defer rows.Close()
					var toBlock []string
					for rows.Next() {
						var id string
						var depsJSON string
						if err := rows.Scan(&id, &depsJSON); err == nil {
							var deps []string
							_ = json.Unmarshal([]byte(depsJSON), &deps)
							for _, dep := range deps {
								if dep == taskID {
									toBlock = append(toBlock, id)
									break
								}
							}
						}
					}
					rows.Close()

					for _, blockID := range toBlock {
						_, _ = tx.ExecContext(ctx, "UPDATE tasks SET status = 'BLOCKED', updated_at = ? WHERE id = ?", now, blockID)
					}
				}
			}
		}

		return nil
	})
}
