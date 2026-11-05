package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"zenfd/internal/domain"
)

// planTask is a lightweight topology node used by the DAG transition policy.
type planTask struct {
	id     string
	status domain.TaskStatus
	deps   []string
}

// loadPlanTasks reads the full task topology of a plan inside tx. It returns the
// snapshot plus a stable iteration order.
func loadPlanTasks(ctx context.Context, tx *sql.Tx, planID string) (map[string]*planTask, []string, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id, status, dependencies FROM tasks WHERE plan_id = ?", planID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load plan tasks: %w", err)
	}
	defer rows.Close()

	snapshot := make(map[string]*planTask)
	order := make([]string, 0)
	for rows.Next() {
		var id, status, depsJSON string
		if err := rows.Scan(&id, &status, &depsJSON); err != nil {
			return nil, nil, fmt.Errorf("failed to scan plan task: %w", err)
		}

		pt := &planTask{id: id, status: domain.TaskStatus(status)}
		// Only PENDING/BLOCKED nodes participate in dependency evaluation.
		// Decoding every row would let a corrupt dependencies value on an
		// unrelated task (for example a COMPLETED one) fail the transition.
		if pt.status == domain.StatusPending || pt.status == domain.StatusBlocked {
			deps := []string{}
			if depsJSON != "" {
				if err := json.Unmarshal([]byte(depsJSON), &deps); err != nil {
					return nil, nil, fmt.Errorf("failed to decode dependencies for task %s: %w", id, err)
				}
			}
			pt.deps = deps
		}

		snapshot[id] = pt
		order = append(order, id)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("failed to read plan tasks: %w", err)
	}

	return snapshot, order, nil
}

// unlockDependents promotes every PENDING or BLOCKED task whose dependencies are
// all COMPLETED to READY.
func unlockDependents(ctx context.Context, tx *sql.Tx, snapshot map[string]*planTask, order []string, now time.Time) error {
	for _, id := range order {
		pt := snapshot[id]
		if pt.status != domain.StatusPending && pt.status != domain.StatusBlocked {
			continue
		}
		if !allDependenciesCompleted(snapshot, pt.deps) {
			continue
		}
		if err := setTaskStatus(ctx, tx, id, domain.StatusReady, now); err != nil {
			return err
		}
		pt.status = domain.StatusReady
	}
	return nil
}

func allDependenciesCompleted(snapshot map[string]*planTask, deps []string) bool {
	for _, depID := range deps {
		dep, ok := snapshot[depID]
		if !ok || dep.status != domain.StatusCompleted {
			return false
		}
	}
	return true
}

// blockDependentsOf blocks every PENDING task that directly depends on the
// failed task (fast-fail branch).
func blockDependentsOf(ctx context.Context, tx *sql.Tx, snapshot map[string]*planTask, order []string, failedID string, now time.Time) error {
	for _, id := range order {
		pt := snapshot[id]
		if pt.status != domain.StatusPending {
			continue
		}
		if !containsString(pt.deps, failedID) {
			continue
		}
		if err := setTaskStatus(ctx, tx, id, domain.StatusBlocked, now); err != nil {
			return err
		}
		pt.status = domain.StatusBlocked
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func setTaskStatus(ctx context.Context, tx *sql.Tx, taskID string, status domain.TaskStatus, now time.Time) error {
	res, err := tx.ExecContext(ctx, "UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?", string(status), now, taskID)
	if err != nil {
		return fmt.Errorf("failed to set task %s to %s: %w", taskID, status, err)
	}
	rows, err := affectedOrErr(res, "set task status", taskID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("%w: task %s disappeared during status update", ErrNotFound, taskID)
	}
	return nil
}

func planIDOf(ctx context.Context, tx *sql.Tx, taskID string) (string, error) {
	var planID string
	if err := tx.QueryRowContext(ctx, "SELECT plan_id FROM tasks WHERE id = ?", taskID).Scan(&planID); err != nil {
		return "", fmt.Errorf("failed to load plan for task %s: %w", taskID, err)
	}
	return planID, nil
}
