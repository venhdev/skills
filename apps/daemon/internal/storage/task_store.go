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

// TaskRepository persists task records and their read projections.
type TaskRepository struct {
	db *DB
}

type scanner interface {
	Scan(dest ...any) error
}

// scanTask maps a tasks row to a domain.Task. The destination order must match
// taskColumns.
func scanTask(s scanner) (*domain.Task, error) {
	var t domain.Task
	var depsJSON, skillsJSON, toolsJSON, filesJSON, criteriaJSON, verifJSON, optionsJSON, conflictJSON string
	var priorityStr, complexityStr, riskStr, statusStr string
	var claimedAt sql.NullTime

	err := s.Scan(
		&t.ID, &t.PlanID, &t.Title, &t.Description, &depsJSON,
		&priorityStr, &complexityStr, &riskStr, &skillsJSON, &toolsJSON,
		&filesJSON, &criteriaJSON, &verifJSON,
		&t.Harness, &t.WorktreePath, &statusStr, &t.ClaimedBy, &claimedAt,
		&t.OrchestratorID, &t.WorkerRole, &t.Notes,
		&t.ReviewReason, &t.ReviewDetails, &optionsJSON, &t.ReviewResponse, &t.ResolvedBy,
		&conflictJSON, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	t.Priority = domain.TaskPriority(priorityStr)
	if t.Priority == "" {
		t.Priority = domain.PriorityP2
	}
	t.Complexity = domain.TaskComplexity(complexityStr)
	if t.Complexity == "" {
		t.Complexity = domain.ComplexityMedium
	}
	t.RiskLevel = domain.TaskRisk(riskStr)
	if t.RiskLevel == "" {
		t.RiskLevel = domain.RiskLow
	}
	t.Status = domain.TaskStatus(statusStr)

	if claimedAt.Valid {
		t.ClaimedAt = &claimedAt.Time
	}

	deps, depsCorrupt := decodeStringArray(depsJSON)
	t.Dependencies = deps
	t.DependenciesCorrupt = depsCorrupt
	t.RequiredSkills = unmarshalStrings(skillsJSON)
	t.RequiredTools = unmarshalStrings(toolsJSON)
	t.TargetFiles = unmarshalStrings(filesJSON)
	t.AcceptanceCriteria = unmarshalStrings(criteriaJSON)
	t.VerificationCommands = unmarshalStrings(verifJSON)
	t.ReviewOptions = unmarshalStrings(optionsJSON)
	t.ConflictFiles = unmarshalStrings(conflictJSON)

	return &t, nil
}

// decodeStringArray decodes a JSON string array and reports whether the stored
// value was malformed. Dependencies are DAG-critical, so callers that act on
// topology (claim, transitions) must refuse a corrupt value instead of silently
// treating the task as dependency-free.
func decodeStringArray(raw string) ([]string, bool) {
	if raw == "" {
		return []string{}, false
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}, true
	}
	if out == nil {
		return []string{}, false
	}
	return out, false
}

// unmarshalStrings decodes a JSON string array and normalizes nil to an empty
// slice so JSON responses never emit null. It is used for presentation fields
// where a corrupt value must not break reads.
func unmarshalStrings(raw string) []string {
	out, _ := decodeStringArray(raw)
	return out
}

// taskArgs builds the insert argument list in taskColumns order.
func taskArgs(task *domain.Task) []any {
	depsJSON, _ := json.Marshal(task.Dependencies)
	skillsJSON, _ := json.Marshal(task.RequiredSkills)
	toolsJSON, _ := json.Marshal(task.RequiredTools)
	filesJSON, _ := json.Marshal(task.TargetFiles)
	criteriaJSON, _ := json.Marshal(task.AcceptanceCriteria)
	verifJSON, _ := json.Marshal(task.VerificationCommands)
	optionsJSON, _ := json.Marshal(task.ReviewOptions)
	conflictJSON, _ := json.Marshal(task.ConflictFiles)

	priority := string(task.Priority)
	if priority == "" {
		priority = string(domain.PriorityP2)
	}
	complexity := string(task.Complexity)
	if complexity == "" {
		complexity = string(domain.ComplexityMedium)
	}
	risk := string(task.RiskLevel)
	if risk == "" {
		risk = string(domain.RiskLow)
	}

	return []any{
		task.ID, task.PlanID, task.Title, task.Description, string(depsJSON),
		priority, complexity, risk, string(skillsJSON), string(toolsJSON),
		string(filesJSON), string(criteriaJSON), string(verifJSON),
		task.Harness, task.WorktreePath, string(task.Status),
		task.ClaimedBy, task.ClaimedAt,
		task.OrchestratorID, task.WorkerRole, task.Notes,
		task.ReviewReason, task.ReviewDetails, string(optionsJSON), task.ReviewResponse, task.ResolvedBy,
		string(conflictJSON), task.CreatedAt, task.UpdatedAt,
	}
}

// normalizeTaskForWrite applies the create-time defaults shared by both write
// paths. Lifecycle fields are only defaulted for a new row; callers that patch
// an existing row never reach this with a meaningful status.
func normalizeTaskForWrite(task *domain.Task, now time.Time) {
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now

	if task.Status == "" {
		task.Status = domain.StatusPending
		if len(task.Dependencies) == 0 {
			task.Status = domain.StatusReady
		}
	}
}

// insertTaskTx creates a task. It never overwrites an existing row, so a
// duplicate ID surfaces as ErrAlreadyExists instead of silently resetting the
// task's lifecycle state.
func insertTaskTx(ctx context.Context, tx *sql.Tx, task *domain.Task, now time.Time) error {
	normalizeTaskForWrite(task, now)
	if err := task.Validate(); err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, taskInsertQuery, taskArgs(task)...)
	if err != nil {
		return fmt.Errorf("failed to insert task %s: %w", task.ID, err)
	}
	rows, err := affectedOrErr(res, "insert task", task.ID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("%w: task %s", ErrAlreadyExists, task.ID)
	}
	return nil
}

// patchTaskTx creates a task when it is missing and otherwise updates only the
// authoring metadata. Status, claim, notes, review, and created_at stay under
// the control of the task state machine.
func patchTaskTx(ctx context.Context, tx *sql.Tx, task *domain.Task, now time.Time) error {
	normalizeTaskForWrite(task, now)
	if err := task.Validate(); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, taskPatchQuery, taskArgs(task)...); err != nil {
		return fmt.Errorf("failed to upsert task %s: %w", task.ID, err)
	}
	return nil
}

// GetTask retrieves a single task by its unique ID.
func (r *TaskRepository) GetTask(ctx context.Context, taskID string) (*domain.Task, error) {
	query := fmt.Sprintf("SELECT %s FROM tasks WHERE id = ?", taskSelectCols)
	row := r.db.QueryRow(ctx, query, taskID)
	t, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query task %s: %w", taskID, err)
	}
	return t, nil
}

// GetTasksByPlan retrieves all tasks belonging to a plan.
func (r *TaskRepository) GetTasksByPlan(ctx context.Context, planSlug string) ([]domain.Task, error) {
	query := fmt.Sprintf(`
	SELECT %s
	FROM tasks
	WHERE plan_id = ?
	ORDER BY id ASC
	`, taskSelectCols)

	rows, err := r.db.Query(ctx, query, planSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by plan: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, *t)
	}

	return tasks, rows.Err()
}

// ListTasks returns tasks filtered optionally by status.
func (r *TaskRepository) ListTasks(ctx context.Context, status domain.TaskStatus) ([]domain.Task, error) {
	var query string
	var args []any

	if status != "" {
		query = fmt.Sprintf(`
		SELECT %s
		FROM tasks
		WHERE status = ?
		ORDER BY created_at ASC
		`, taskSelectCols)
		args = []any{string(status)}
	} else {
		query = fmt.Sprintf(`
		SELECT %s
		FROM tasks
		ORDER BY created_at ASC
		`, taskSelectCols)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, *t)
	}

	return tasks, rows.Err()
}

// UpdateTaskMetadata applies a partial metadata patch to a READY or PENDING
// task. Dependency changes revalidate the plan graph and re-derive the status.
func (r *TaskRepository) UpdateTaskMetadata(ctx context.Context, taskID string, patch domain.TaskPatch) error {
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		task, err := getTaskTx(ctx, tx, taskID)
		if err != nil {
			return err
		}
		if task.Status != domain.StatusReady && task.Status != domain.StatusPending {
			return fmt.Errorf("%w: task %s is %s; metadata can only change while READY or PENDING", ErrInvalidState, taskID, task.Status)
		}

		applyTaskPatch(task, patch)
		if err := patchTaskTx(ctx, tx, task, now); err != nil {
			return err
		}
		if err := validateTaskGraphTx(ctx, tx, task.PlanID); err != nil {
			return err
		}
		return resyncTaskStatuses(ctx, tx, task.PlanID, []domain.Task{*task}, now)
	})
}

// applyTaskPatch copies non-nil patch fields onto the task.
func applyTaskPatch(task *domain.Task, patch domain.TaskPatch) {
	if patch.Title != nil {
		task.Title = *patch.Title
	}
	if patch.Description != nil {
		task.Description = *patch.Description
	}
	if patch.Dependencies != nil {
		task.Dependencies = *patch.Dependencies
	}
	if patch.Priority != nil {
		task.Priority = *patch.Priority
	}
	if patch.Complexity != nil {
		task.Complexity = *patch.Complexity
	}
	if patch.RiskLevel != nil {
		task.RiskLevel = *patch.RiskLevel
	}
	if patch.RequiredSkills != nil {
		task.RequiredSkills = *patch.RequiredSkills
	}
	if patch.RequiredTools != nil {
		task.RequiredTools = *patch.RequiredTools
	}
	if patch.TargetFiles != nil {
		task.TargetFiles = *patch.TargetFiles
	}
	if patch.AcceptanceCriteria != nil {
		task.AcceptanceCriteria = *patch.AcceptanceCriteria
	}
	if patch.VerificationCommands != nil {
		task.VerificationCommands = *patch.VerificationCommands
	}
	if patch.Harness != nil {
		task.Harness = *patch.Harness
	}
	if patch.WorktreePath != nil {
		task.WorktreePath = *patch.WorktreePath
	}
}

// getTaskTx reads a task inside an existing transaction.
func getTaskTx(ctx context.Context, tx *sql.Tx, taskID string) (*domain.Task, error) {
	query := fmt.Sprintf("SELECT %s FROM tasks WHERE id = ?", taskSelectCols)
	task, err := scanTask(tx.QueryRowContext(ctx, query, taskID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: task %s", ErrNotFound, taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query task %s: %w", taskID, err)
	}
	return task, nil
}

// CreateTask inserts a single task. It fails with ErrAlreadyExists when the ID
// is taken; callers that mean to edit authoring metadata must not use it.
func (r *TaskRepository) CreateTask(ctx context.Context, task domain.Task) error {
	return r.CreateTasks(ctx, []domain.Task{task})
}

// CreateTasks inserts a batch of tasks in one transaction. The batch is atomic:
// a duplicate ID or an invalid graph rolls the whole call back, and forward
// references inside the batch are validated after every row is written.
func (r *TaskRepository) CreateTasks(ctx context.Context, tasks []domain.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	planID := tasks[0].PlanID
	if planID == "" {
		return fmt.Errorf("task plan_id cannot be empty")
	}

	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		for i := range tasks {
			if tasks[i].PlanID == "" {
				tasks[i].PlanID = planID
			}
			if tasks[i].PlanID != planID {
				return fmt.Errorf("%w: batch spans plans %s and %s", ErrInvalidGraph, planID, tasks[i].PlanID)
			}
			if err := insertTaskTx(ctx, tx, &tasks[i], now); err != nil {
				return err
			}
		}
		if err := validateTaskGraphTx(ctx, tx, planID); err != nil {
			return err
		}
		return resyncTaskStatuses(ctx, tx, planID, tasks, now)
	})
}

// resyncTaskStatuses recomputes the initial status of newly inserted tasks from
// the prerequisites that already exist in the plan. Without this, a task added
// after its dependencies completed would stay PENDING forever.
func resyncTaskStatuses(ctx context.Context, tx *sql.Tx, planID string, tasks []domain.Task, now time.Time) error {
	statuses, err := loadPlanStatuses(ctx, tx, planID)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		current := statuses[task.ID]
		// Explicit lifecycle states (RUNNING, terminal, BLOCKED) are preserved;
		// only the derived initial states are recomputed.
		if current != domain.StatusReady && current != domain.StatusPending {
			continue
		}
		target := deriveStatus(statuses, task.Dependencies)
		if current == target {
			continue
		}
		if err := setTaskStatus(ctx, tx, task.ID, target, now); err != nil {
			return err
		}
		statuses[task.ID] = target
	}
	return nil
}

// UpdateTaskDependencies updates the task dependency list.
func (r *TaskRepository) UpdateTaskDependencies(ctx context.Context, taskID string, deps []string) error {
	now := time.Now().UTC()
	depsJSON, err := json.Marshal(deps)
	if err != nil {
		return fmt.Errorf("failed to encode dependencies for task %s: %w", taskID, err)
	}
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "UPDATE tasks SET dependencies = ?, updated_at = ? WHERE id = ?", string(depsJSON), now, taskID)
		if err != nil {
			return err
		}
		rows, err := affectedOrErr(res, "update task dependencies", taskID)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: task %s", ErrNotFound, taskID)
		}

		planID, err := planIDOf(ctx, tx, taskID)
		if err != nil {
			return err
		}
		if err := validateTaskGraphTx(ctx, tx, planID); err != nil {
			return err
		}

		// Recompute this task's readiness after the edge change, but never move
		// a task that is already executing or finished.
		var status string
		if err := tx.QueryRowContext(ctx, "SELECT status FROM tasks WHERE id = ?", taskID).Scan(&status); err != nil {
			return err
		}
		if current := domain.TaskStatus(status); current == domain.StatusPending ||
			current == domain.StatusBlocked || current == domain.StatusReady {
			statuses, err := loadPlanStatuses(ctx, tx, planID)
			if err != nil {
				return err
			}
			target := deriveStatus(statuses, deps)
			if target != current {
				if err := setTaskStatus(ctx, tx, taskID, target, now); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// UpdateTaskNotes updates developer supplemental notes.
func (r *TaskRepository) UpdateTaskNotes(ctx context.Context, taskID string, notes string) error {
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "UPDATE tasks SET notes = ?, updated_at = ? WHERE id = ?", notes, now, taskID)
		if err != nil {
			return err
		}
		rows, err := affectedOrErr(res, "update task notes", taskID)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: task %s", ErrNotFound, taskID)
		}
		return nil
	})
}

// AppendTaskNotes appends a note to the task's supplemental notes.
func (r *TaskRepository) AppendTaskNotes(ctx context.Context, taskID string, note string) error {
	if note == "" {
		return nil
	}
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		return appendTaskNotesTx(ctx, tx, taskID, note, now)
	})
}

// appendTaskNotesTx appends a note inside an existing transaction.
func appendTaskNotesTx(ctx context.Context, tx *sql.Tx, taskID string, note string, now time.Time) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET notes = CASE WHEN notes = '' THEN ? ELSE notes || '\n' || ? END, updated_at = ?
		WHERE id = ?
	`, note, note, now, taskID)
	if err != nil {
		return err
	}
	rows, err := affectedOrErr(res, "append task notes", taskID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("%w: task %s", ErrNotFound, taskID)
	}
	return nil
}
