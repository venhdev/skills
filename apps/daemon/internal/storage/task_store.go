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
		&t.Harness, &t.SkillCommand, &t.WorktreePath, &statusStr, &t.ClaimedBy, &claimedAt,
		&t.OrchestratorID, &t.WorkerRole, &t.PID, &t.OrchestratorPID, &t.Notes,
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

	t.Dependencies = unmarshalStrings(depsJSON)
	t.RequiredSkills = unmarshalStrings(skillsJSON)
	t.RequiredTools = unmarshalStrings(toolsJSON)
	t.TargetFiles = unmarshalStrings(filesJSON)
	t.AcceptanceCriteria = unmarshalStrings(criteriaJSON)
	t.VerificationCommands = unmarshalStrings(verifJSON)
	t.ReviewOptions = unmarshalStrings(optionsJSON)
	t.ConflictFiles = unmarshalStrings(conflictJSON)

	return &t, nil
}

// unmarshalStrings decodes a JSON string array and normalizes nil to an empty
// slice so JSON responses never emit null.
func unmarshalStrings(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return []string{}
	}
	return out
}

// taskArgs builds the upsert argument list in taskColumns order.
func taskArgs(task *domain.Task, planID string) []any {
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
		task.ID, planID, task.Title, task.Description, string(depsJSON),
		priority, complexity, risk, string(skillsJSON), string(toolsJSON),
		string(filesJSON), string(criteriaJSON), string(verifJSON),
		task.Harness, task.SkillCommand, task.WorktreePath, string(task.Status),
		task.ClaimedBy, task.ClaimedAt,
		task.OrchestratorID, task.WorkerRole, task.PID, task.OrchestratorPID, task.Notes,
		task.ReviewReason, task.ReviewDetails, string(optionsJSON), task.ReviewResponse, task.ResolvedBy,
		string(conflictJSON), task.CreatedAt, task.UpdatedAt,
	}
}

func upsertTaskTx(ctx context.Context, tx *sql.Tx, task *domain.Task, defaultPlanID string, now time.Time) error {
	if err := task.Validate(); err != nil {
		return err
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now

	planID := task.PlanID
	if planID == "" {
		planID = defaultPlanID
	}

	if task.Status == "" {
		task.Status = domain.StatusPending
		if len(task.Dependencies) == 0 {
			task.Status = domain.StatusReady
		}
	}

	if _, err := tx.ExecContext(ctx, taskUpsertQuery, taskArgs(task, planID)...); err != nil {
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

// GetReadyTasks returns tasks whose status is READY.
func (r *TaskRepository) GetReadyTasks(ctx context.Context, limit int) ([]domain.Task, error) {
	if limit <= 0 {
		limit = 10
	}

	query := fmt.Sprintf(`
	SELECT %s
	FROM tasks
	WHERE status = 'READY'
	ORDER BY created_at ASC
	LIMIT ?
	`, taskSelectCols)

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query ready tasks: %w", err)
	}
	defer rows.Close()

	var tasks []domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ready task: %w", err)
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

// CreateTask inserts a single task.
func (r *TaskRepository) CreateTask(ctx context.Context, task domain.Task) error {
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		return upsertTaskTx(ctx, tx, &task, task.PlanID, now)
	})
}

// UpdateTaskDependencies updates the task dependency list.
func (r *TaskRepository) UpdateTaskDependencies(ctx context.Context, taskID string, deps []string) error {
	now := time.Now().UTC()
	depsJSON, _ := json.Marshal(deps)
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "UPDATE tasks SET dependencies = ?, updated_at = ? WHERE id = ?", string(depsJSON), now, taskID)
		return err
	})
}

// UpdateTaskNotes updates developer supplemental notes.
func (r *TaskRepository) UpdateTaskNotes(ctx context.Context, taskID string, notes string) error {
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "UPDATE tasks SET notes = ?, updated_at = ? WHERE id = ?", notes, now, taskID)
		return err
	})
}

// AppendTaskNotes appends a note to the task's supplemental notes.
func (r *TaskRepository) AppendTaskNotes(ctx context.Context, taskID string, note string) error {
	if note == "" {
		return nil
	}
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET notes = CASE WHEN notes = '' THEN ? ELSE notes || '\n' || ? END, updated_at = ?
			WHERE id = ?
		`, note, note, now, taskID)
		return err
	})
}
