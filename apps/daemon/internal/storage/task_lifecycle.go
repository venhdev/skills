package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"zenf/internal/domain"
)

// TaskLifecycle owns the task state machine, claim mutual exclusion, and the
// review handshake. It composes the task read projection and the context packet
// builder instead of performing filesystem or prompt work itself.
type TaskLifecycle struct {
	db      *DB
	tasks   *TaskRepository
	packets domain.ContextPacketBuilder
}

// SetPacketBuilder sets the context packet builder.
func (l *TaskLifecycle) SetPacketBuilder(b domain.ContextPacketBuilder) {
	l.packets = b
}

// UpdateTaskStatus atomically transitions task status and resolves DAG
// dependencies. Every failure aborts the transaction; nothing is swallowed.
func (l *TaskLifecycle) UpdateTaskStatus(ctx context.Context, taskID string, from domain.TaskStatus, to domain.TaskStatus) error {
	return l.updateTaskStatus(ctx, taskID, from, to, "")
}

// UpdateTaskStatusWithNote applies a transition and appends an audit note in the
// same transaction, so a recorded transition can never lose its explanation.
func (l *TaskLifecycle) UpdateTaskStatusWithNote(ctx context.Context, taskID string, from domain.TaskStatus, to domain.TaskStatus, note string) error {
	return l.updateTaskStatus(ctx, taskID, from, to, note)
}

func (l *TaskLifecycle) updateTaskStatus(ctx context.Context, taskID string, from domain.TaskStatus, to domain.TaskStatus, note string) error {
	if !from.CanTransitionTo(to) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidState, from, to)
	}

	now := time.Now().UTC()

	return l.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		task, err := getTaskTx(ctx, tx, taskID)
		if err != nil {
			return err
		}
		if task.Status != from {
			return fmt.Errorf("%w: task %s is not in status %s", ErrConflictStatus, taskID, from)
		}
		if task.DependenciesCorrupt && (to == domain.StatusCompleted || to == domain.StatusFailed) {
			return fmt.Errorf("%w: task %s has malformed dependencies; repair the row before completing it", ErrInvalidGraph, taskID)
		}

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

		if note != "" {
			if err := appendTaskNotesTx(ctx, tx, taskID, note, now); err != nil {
				return err
			}
		}

		if to != domain.StatusCompleted && to != domain.StatusFailed {
			return nil
		}

		snapshot, order, err := loadContainerNodes(ctx, tx, task.ProjectSlug, task.PlanSlug)
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

// ClaimTask atomically verifies readiness, sets the claim lock, transitions to
// RUNNING, and returns the ContextPacket.
func (l *TaskLifecycle) ClaimTask(ctx context.Context, taskID string, opts domain.ClaimOptions) (*domain.ContextPacket, error) {
	workerID := opts.WorkerID
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
	if task.DependenciesCorrupt {
		return nil, fmt.Errorf("%w: task %s has malformed dependencies; repair the row before claiming", ErrInvalidGraph, taskID)
	}

	var packet *domain.ContextPacket
	if l.packets != nil {
		p, err := l.packets.Build(ctx, *task)
		if err != nil {
			return nil, err
		}
		packet = p
	} else {
		packet = &domain.ContextPacket{
			Task:                 *task,
			TargetFiles:          task.TargetFiles,
			VerificationCommands: task.VerificationCommands,
		}
	}

	// Commit only the claim transition while holding the write lock. The UPDATE
	// re-checks READY so a concurrent claim still loses.
	err = l.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = 'RUNNING', claimed_by = ?, claimed_at = ?,
			    orchestrator_id = ?, worker_role = ?, claim_session_id = ?, updated_at = ?
			WHERE id = ? AND status = 'READY'
		`, workerID, now, opts.OrchestratorID, opts.WorkerRole, opts.SessionID, now, taskID)
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
		return nil
	})
	if err != nil {
		return nil, err
	}

	// The prompt compiler does not depend on status or claim metadata, so the
	// preflight packet is valid. Apply the committed claim fields before return.
	applyClaim(task, workerID, opts, now)
	packet.Task = *task
	return packet, nil
}

// applyClaim mirrors the columns written by the claim UPDATE.
func applyClaim(task *domain.Task, workerID string, opts domain.ClaimOptions, now time.Time) {
	task.Status = domain.StatusRunning
	task.ClaimedBy = workerID
	task.OrchestratorID = opts.OrchestratorID
	task.WorkerRole = opts.WorkerRole
	task.ClaimSessionID = opts.SessionID
	task.ClaimedAt = &now
	task.UpdatedAt = now
}

// EvictTask evicts a running task, resetting it to READY or cancelling it. A
// reset task whose prerequisites are not all completed lands in BLOCKED instead
// of becoming claimable behind a failed upstream. Claim metadata is cleared so a
// later claim starts from a clean slate. The returned status is the status
// actually written.
func (l *TaskLifecycle) EvictTask(ctx context.Context, taskID string, reason string, action string) (domain.TaskStatus, error) {
	now := time.Now().UTC()
	targetStatus := domain.StatusReady
	if action == "CANCEL" {
		targetStatus = domain.StatusCancelled
	}

	err := l.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		task, err := getTaskTx(ctx, tx, taskID)
		if err != nil {
			return err
		}
		if task.Status != domain.StatusRunning {
			return fmt.Errorf("%w: task %s is not RUNNING", ErrConflictStatus, taskID)
		}
		if targetStatus == domain.StatusReady {
			if task.DependenciesCorrupt {
				return fmt.Errorf("%w: task %s has malformed dependencies", ErrInvalidGraph, taskID)
			}
			statuses, err := loadContainerStatuses(ctx, tx, task.ProjectSlug, task.PlanSlug)
			if err != nil {
				return err
			}
			if !allStatusesCompleted(statuses, task.Dependencies) {
				targetStatus = domain.StatusBlocked
			}
		}

		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = ?, claimed_by = '', claimed_at = NULL,
			    orchestrator_id = '', worker_role = '', claim_session_id = '',
			    notes = CASE WHEN notes = '' THEN ? ELSE notes || char(10) || ? END,
			    updated_at = ?
			WHERE id = ? AND status = 'RUNNING'
		`, string(targetStatus), "Eviction: "+reason, "Eviction: "+reason, now, taskID)
		if err != nil {
			return err
		}
		rows, err := affectedOrErr(res, "evict task", taskID)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: task %s is not RUNNING", ErrConflictStatus, taskID)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return targetStatus, nil
}

func allStatusesCompleted(statuses map[string]domain.TaskStatus, deps []string) bool {
	for _, dep := range deps {
		if statuses[dep] != domain.StatusCompleted {
			return false
		}
	}
	return true
}

// RequestReview pauses execution, records review reasons/options, and
// transitions the task to AWAITING_REVIEW.
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
			return fmt.Errorf("%w: task %s", ErrNotFound, taskID)
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

// ResolveReview records the human guidance relayed by the main agent and
// transitions the task back to READY with a clean claim slate.
func (l *TaskLifecycle) ResolveReview(ctx context.Context, taskID string, response string) error {
	now := time.Now().UTC()
	return l.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		var currentStatus string
		err := tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&currentStatus)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: task %s", ErrNotFound, taskID)
		}
		if err != nil {
			return err
		}

		if domain.TaskStatus(currentStatus) != domain.StatusAwaitingReview {
			return fmt.Errorf("%w: task %s is not AWAITING_REVIEW (currently %s)", ErrInvalidState, taskID, currentStatus)
		}

		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = 'READY', review_response = ?, claimed_by = '', claimed_at = NULL,
			    orchestrator_id = '', worker_role = '', claim_session_id = '',
			    resolved_by = 'main-agent', updated_at = ?
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

// GetTaskContextPacket compiles the full context packet: task, project, plan,
// agent persona, linked documents, target files, skill paths, and media.
func (l *TaskLifecycle) GetTaskContextPacket(ctx context.Context, taskID string) (*domain.ContextPacket, error) {
	task, err := l.tasks.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if l.packets != nil {
		return l.packets.Build(ctx, *task)
	}
	return &domain.ContextPacket{
		Task:                 *task,
		TargetFiles:          task.TargetFiles,
		VerificationCommands: task.VerificationCommands,
	}, nil
}
