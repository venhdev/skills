package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"zenfd/internal/domain"
)

// TaskLifecycle owns the task state machine, claim mutual exclusion, and the
// review handshake. It composes the task read projection and the context packet
// builder instead of performing filesystem or prompt work itself.
type TaskLifecycle struct {
	db      *DB
	tasks   *TaskRepository
	packets domain.ContextPacketBuilder
}

// UpdateTaskStatus atomically transitions task status and resolves DAG
// dependencies. Every failure aborts the transaction; nothing is swallowed.
func (l *TaskLifecycle) UpdateTaskStatus(ctx context.Context, taskID string, from domain.TaskStatus, to domain.TaskStatus) error {
	if !from.CanTransitionTo(to) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidState, from, to)
	}

	now := time.Now().UTC()

	return l.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "UPDATE tasks SET status = ?, updated_at = ? WHERE id = ? AND status = ?",
			string(to), now, taskID, string(from))
		if err != nil {
			return fmt.Errorf("failed to update task status: %w", err)
		}

		rows, err := affectedOrErr(res, "update task status", taskID)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: task %s is not in status %s", ErrConflictStatus, taskID, from)
		}

		if to != domain.StatusCompleted && to != domain.StatusFailed {
			return nil
		}

		planID, err := planIDOf(ctx, tx, taskID)
		if err != nil {
			return err
		}
		snapshot, order, err := loadPlanTasks(ctx, tx, planID)
		if err != nil {
			return err
		}

		switch to {
		case domain.StatusCompleted:
			return unlockDependents(ctx, tx, snapshot, order, now)
		case domain.StatusFailed:
			return blockDependentsOf(ctx, tx, snapshot, order, taskID, now)
		}
		return nil
	})
}

// ClaimTask atomically verifies readiness, sets claim lock (workerID & pid), transitions to RUNNING, and returns ContextPacket.
func (l *TaskLifecycle) ClaimTask(ctx context.Context, taskID string, workerID string, pid int) (*domain.ContextPacket, error) {
	return l.ClaimTaskWithMeta(ctx, taskID, workerID, "", "", pid, 0)
}

// ClaimTaskWithMeta atomically claims a ready task with full metadata.
func (l *TaskLifecycle) ClaimTaskWithMeta(ctx context.Context, taskID string, workerID string, orchestratorID string, workerRole string, pid int, orchestratorPID int) (*domain.ContextPacket, error) {
	if workerID == "" {
		workerID = fmt.Sprintf("worker-%d", time.Now().UnixNano())
	}
	now := time.Now().UTC()

	// Preflight outside the write transaction. Building the packet performs
	// filesystem checks, artifact reads, and prompt compilation; keeping that
	// work out of ExecImmediate prevents it from blocking every other writer.
	task, err := l.tasks.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task.Status != domain.StatusReady {
		return nil, fmt.Errorf("%w: task %s is not in READY status (current: %s)", ErrConflictStatus, taskID, task.Status)
	}

	packet, err := l.packets.Build(ctx, *task)
	if err != nil {
		return nil, err
	}

	// Commit only the claim transition and run registration while holding the
	// write lock. The UPDATE re-checks READY so a concurrent claim still loses.
	err = l.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = 'RUNNING', claimed_by = ?, orchestrator_id = ?, worker_role = ?, pid = ?, orchestrator_pid = ?, claimed_at = ?, updated_at = ?
			WHERE id = ? AND status = 'READY'
		`, workerID, orchestratorID, workerRole, pid, orchestratorPID, now, now, taskID)
		if err != nil {
			return fmt.Errorf("failed to claim task: %w", err)
		}
		rows, err := affectedOrErr(res, "claim task", taskID)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: task %s was already claimed by another worker", ErrConflictStatus, taskID)
		}

		// If pid > 0, register Run attempt as part of the same atomic claim.
		if pid > 0 {
			runID := domain.NewRunID(taskID)
			return insertRunTx(ctx, tx, domain.Run{
				ID:         runID,
				TaskID:     taskID,
				PID:        pid,
				PGID:       pid,
				Status:     domain.StatusRunning,
				LogPath:    filepath.Join(os.TempDir(), fmt.Sprintf("%s.log", runID)),
				StartedAt:  now,
				LastBeatAt: now,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// The prompt compiler does not depend on status or claim metadata, so the
	// preflight packet is valid. Apply the committed claim fields before return.
	applyClaim(task, workerID, orchestratorID, workerRole, pid, orchestratorPID, now)
	packet.Task = *task
	return packet, nil
}

// applyClaim mirrors the columns written by the claim UPDATE.
func applyClaim(task *domain.Task, workerID, orchestratorID, workerRole string, pid, orchestratorPID int, now time.Time) {
	task.Status = domain.StatusRunning
	task.ClaimedBy = workerID
	task.OrchestratorID = orchestratorID
	task.WorkerRole = workerRole
	task.PID = pid
	task.OrchestratorPID = orchestratorPID
	task.ClaimedAt = &now
	task.UpdatedAt = now
}

// EvictTask evicts a running or suspended task, resetting it to READY or cancelling it.
func (l *TaskLifecycle) EvictTask(ctx context.Context, taskID string, reason string, action string) error {
	now := time.Now().UTC()
	targetStatus := domain.StatusReady
	if action == "CANCEL" {
		targetStatus = domain.StatusCancelled
	}

	return l.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = ?, claimed_by = '', pid = 0, notes = CASE WHEN notes = '' THEN ? ELSE notes || '\n' || ? END, updated_at = ?
			WHERE id = ? AND (status = 'RUNNING' OR status = 'SUSPENDED')
		`, string(targetStatus), "Eviction: "+reason, "Eviction: "+reason, now, taskID)
		if err != nil {
			return err
		}
		rows, err := affectedOrErr(res, "evict task", taskID)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: task %s is not in RUNNING or SUSPENDED status", ErrConflictStatus, taskID)
		}
		return nil
	})
}

// RequestReview pauses execution, records review reasons/options, and transitions task to AWAITING_REVIEW.
func (l *TaskLifecycle) RequestReview(ctx context.Context, taskID string, reason string, details string, options []string) error {
	now := time.Now().UTC()
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		optionsJSON = []byte("[]")
	}

	return l.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		var currentStatus string
		err := tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		if domain.TaskStatus(currentStatus) != domain.StatusRunning {
			return fmt.Errorf("%w: task %s must be RUNNING to request review (currently %s)", ErrInvalidState, taskID, currentStatus)
		}

		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = 'AWAITING_REVIEW', review_reason = ?, review_details = ?, review_options = ?, review_response = '', updated_at = ?
			WHERE id = ? AND status = 'RUNNING'
		`, reason, details, string(optionsJSON), now, taskID)
		if err != nil {
			return err
		}
		rows, err := affectedOrErr(res, "request review", taskID)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: failed to transition task %s to AWAITING_REVIEW", ErrConflictStatus, taskID)
		}
		return nil
	})
}

// ResolveReview records the human guidance relayed by the main agent and transitions task back to READY.
func (l *TaskLifecycle) ResolveReview(ctx context.Context, taskID string, response string) error {
	now := time.Now().UTC()
	return l.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		var currentStatus string
		err := tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}

		if domain.TaskStatus(currentStatus) != domain.StatusAwaitingReview {
			return fmt.Errorf("%w: task %s is not AWAITING_REVIEW (currently %s)", ErrInvalidState, taskID, currentStatus)
		}

		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = 'READY', review_response = ?, claimed_by = '', pid = 0, resolved_by = 'main-agent', updated_at = ?
			WHERE id = ? AND status = 'AWAITING_REVIEW'
		`, response, now, taskID)
		if err != nil {
			return err
		}
		rows, err := affectedOrErr(res, "resolve review", taskID)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: failed to resolve review for task %s", ErrConflictStatus, taskID)
		}
		return nil
	})
}

// GetTaskContextPacket compiles full context packet (task, target files, skill paths, media).
func (l *TaskLifecycle) GetTaskContextPacket(ctx context.Context, taskID string) (*domain.ContextPacket, error) {
	t, err := l.tasks.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return l.packets.Build(ctx, *t)
}
