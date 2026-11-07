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

const taskSelectColumns = `
id,
COALESCE(project_slug, '') AS project_slug,
COALESCE(plan_slug, '') AS plan_slug,
title, description, dependencies, priority, complexity, risk_level, agent_slug,
required_skills, required_tools, required_docs, target_files, acceptance_criteria,
verification_commands, harness, worktree_path, status, claimed_by, claimed_at,
orchestrator_id, worker_role, claim_session_id, notes, review_reason, review_details,
review_options, review_response, resolved_by, conflict_files, phases, current_phase, created_at, updated_at`

const taskInsertQuery = `
INSERT INTO tasks (
	id, project_slug, plan_slug, title, description, dependencies, priority, complexity,
	risk_level, agent_slug, required_skills, required_tools, required_docs, target_files,
	acceptance_criteria, verification_commands, harness, worktree_path, status, claimed_by,
	claimed_at, orchestrator_id, worker_role, claim_session_id, notes, review_reason,
	review_details, review_options, review_response, resolved_by, conflict_files,
	phases, current_phase, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO NOTHING`

const taskAuthoringUpdateQuery = `
INSERT INTO tasks (
	id, project_slug, plan_slug, title, description, dependencies, priority, complexity,
	risk_level, agent_slug, required_skills, required_tools, required_docs, target_files,
	acceptance_criteria, verification_commands, harness, worktree_path, status, claimed_by,
	claimed_at, orchestrator_id, worker_role, claim_session_id, notes, review_reason,
	review_details, review_options, review_response, resolved_by, conflict_files,
	phases, current_phase, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	title = excluded.title,
	description = excluded.description,
	dependencies = excluded.dependencies,
	priority = excluded.priority,
	complexity = excluded.complexity,
	risk_level = excluded.risk_level,
	agent_slug = excluded.agent_slug,
	required_skills = excluded.required_skills,
	required_tools = excluded.required_tools,
	required_docs = excluded.required_docs,
	target_files = excluded.target_files,
	acceptance_criteria = excluded.acceptance_criteria,
	verification_commands = excluded.verification_commands,
	harness = excluded.harness,
	worktree_path = excluded.worktree_path,
	phases = excluded.phases,
	current_phase = excluded.current_phase,
	updated_at = excluded.updated_at`

func scanTask(s scanner) (*domain.Task, error) {
	var t domain.Task
	var depsJSON, skillsJSON, toolsJSON, docsJSON, filesJSON, criteriaJSON, verifJSON, optionsJSON, conflictJSON, phasesJSON string
	var priorityStr, complexityStr, riskStr, statusStr string
	var claimedAt sql.NullTime

	err := s.Scan(
		&t.ID, &t.ProjectSlug, &t.PlanSlug, &t.Title, &t.Description, &depsJSON,
		&priorityStr, &complexityStr, &riskStr, &t.AgentSlug, &skillsJSON, &toolsJSON, &docsJSON,
		&filesJSON, &criteriaJSON, &verifJSON,
		&t.Harness, &t.WorktreePath, &statusStr, &t.ClaimedBy, &claimedAt,
		&t.OrchestratorID, &t.WorkerRole, &t.ClaimSessionID, &t.Notes,
		&t.ReviewReason, &t.ReviewDetails, &optionsJSON, &t.ReviewResponse, &t.ResolvedBy,
		&conflictJSON, &phasesJSON, &t.CurrentPhase, &t.CreatedAt, &t.UpdatedAt,
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
	t.RequiredDocs = unmarshalStrings(docsJSON)
	t.TargetFiles = unmarshalStrings(filesJSON)
	t.AcceptanceCriteria = unmarshalStrings(criteriaJSON)
	t.VerificationCommands = unmarshalStrings(verifJSON)
	t.ReviewOptions = unmarshalStrings(optionsJSON)
	t.ConflictFiles = unmarshalStrings(conflictJSON)

	if phasesJSON != "" && phasesJSON != "[]" {
		var phases []domain.TaskPhase
		if err := json.Unmarshal([]byte(phasesJSON), &phases); err == nil {
			t.Phases = phases
		}
	}
	if t.Phases == nil {
		t.Phases = []domain.TaskPhase{}
	}

	return &t, nil
}

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

func unmarshalStrings(raw string) []string {
	out, _ := decodeStringArray(raw)
	return out
}

func taskArgs(task *domain.Task) []any {
	depsJSON, _ := json.Marshal(task.Dependencies)
	skillsJSON, _ := json.Marshal(task.RequiredSkills)
	toolsJSON, _ := json.Marshal(task.RequiredTools)
	docsJSON, _ := json.Marshal(task.RequiredDocs)
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
	harness := task.Harness
	if harness == "" {
		harness = "orchestrator"
	}

	phasesJSON, _ := json.Marshal(task.Phases)
	if task.Phases == nil {
		phasesJSON = []byte("[]")
	}

	return []any{
		task.ID, nullableString(task.ProjectSlug), nullableString(task.PlanSlug),
		task.Title, task.Description, string(depsJSON), priority, complexity, risk,
		task.AgentSlug, string(skillsJSON), string(toolsJSON), string(docsJSON),
		string(filesJSON), string(criteriaJSON), string(verifJSON),
		harness, task.WorktreePath, string(task.Status),
		task.ClaimedBy, task.ClaimedAt, task.OrchestratorID, task.WorkerRole,
		task.ClaimSessionID, task.Notes, task.ReviewReason, task.ReviewDetails,
		string(optionsJSON), task.ReviewResponse, task.ResolvedBy,
		string(conflictJSON), string(phasesJSON), task.CurrentPhase, task.CreatedAt, task.UpdatedAt,
	}
}

func normalizeTaskForWrite(task *domain.Task, now time.Time) {
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	if task.Harness == "" {
		task.Harness = "orchestrator"
	}
	if task.Status == "" {
		task.Status = domain.StatusPending
		if len(task.Dependencies) == 0 {
			task.Status = domain.StatusReady
		}
	}
}

// resolveTaskScopeTx fills project_slug from the plan and verifies that an
// explicit project matches. It is a no-op for project tasks and orphans.
func resolveTaskScopeTx(ctx context.Context, tx *sql.Tx, task *domain.Task) error {
	if task.PlanSlug != "" {
		var planProject string
		err := tx.QueryRowContext(ctx, "SELECT project_slug FROM plans WHERE slug = ?", task.PlanSlug).Scan(&planProject)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: plan %s", ErrNotFound, task.PlanSlug)
		}
		if err != nil {
			return fmt.Errorf("failed to resolve plan %s: %w", task.PlanSlug, err)
		}
		if task.ProjectSlug == "" {
			task.ProjectSlug = planProject
		} else if task.ProjectSlug != planProject {
			return fmt.Errorf("%w: plan %s belongs to project %s, not %s", ErrInvalidGraph, task.PlanSlug, planProject, task.ProjectSlug)
		}
	}
	if task.ProjectSlug != "" {
		if _, err := getProjectTx(ctx, tx, task.ProjectSlug); err != nil {
			return err
		}
	}
	return nil
}

func insertTaskTx(ctx context.Context, tx *sql.Tx, task *domain.Task, now time.Time) error {
	if err := resolveTaskScopeTx(ctx, tx, task); err != nil {
		return err
	}
	normalizeTaskForWrite(task, now)
	if err := task.Validate(); err != nil {
		return err
	}
	if err := assertDependenciesInContainer(ctx, tx, task); err != nil {
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

func upsertTaskAuthoringTx(ctx context.Context, tx *sql.Tx, task *domain.Task, now time.Time) error {
	if err := resolveTaskScopeTx(ctx, tx, task); err != nil {
		return err
	}
	normalizeTaskForWrite(task, now)
	if err := task.Validate(); err != nil {
		return err
	}
	if err := assertDependenciesInContainer(ctx, tx, task); err != nil {
		return err
	}

	var existingProject, existingPlan string
	err := tx.QueryRowContext(ctx,
		"SELECT COALESCE(project_slug, ''), COALESCE(plan_slug, '') FROM tasks WHERE id = ?", task.ID).
		Scan(&existingProject, &existingPlan)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to inspect task %s: %w", task.ID, err)
	}
	if err == nil && (existingProject != task.ProjectSlug || existingPlan != task.PlanSlug) {
		return fmt.Errorf("%w: task %s belongs to project %q plan %q", ErrInvalidGraph, task.ID, existingProject, existingPlan)
	}

	if _, err := tx.ExecContext(ctx, taskAuthoringUpdateQuery, taskArgs(task)...); err != nil {
		return fmt.Errorf("failed to upsert task %s: %w", task.ID, err)
	}
	return nil
}

func assertDependenciesInContainer(ctx context.Context, tx *sql.Tx, task *domain.Task) error {
	seen := make(map[string]bool, len(task.Dependencies))
	for _, dep := range task.Dependencies {
		if dep == task.ID {
			return fmt.Errorf("%w: task %s depends on itself", ErrInvalidGraph, task.ID)
		}
		if dep == "" {
			return fmt.Errorf("%w: task %s has an empty dependency", ErrInvalidGraph, task.ID)
		}
		if seen[dep] {
			return fmt.Errorf("%w: task %s repeats dependency %s", ErrInvalidGraph, task.ID, dep)
		}
		seen[dep] = true

		var depProject, depPlan string
		err := tx.QueryRowContext(ctx,
			"SELECT COALESCE(project_slug, ''), COALESCE(plan_slug, '') FROM tasks WHERE id = ?", dep).
			Scan(&depProject, &depPlan)
		if errors.Is(err, sql.ErrNoRows) {
			continue // forward reference; validated once the prerequisite exists
		}
		if err != nil {
			return fmt.Errorf("failed to inspect dependency %s: %w", dep, err)
		}
		if depProject != task.ProjectSlug || depPlan != task.PlanSlug {
			return fmt.Errorf("%w: task %s cannot depend on %s from another container", ErrInvalidGraph, task.ID, dep)
		}
	}
	return nil
}

func (r *TaskRepository) GetTask(ctx context.Context, taskID string) (*domain.Task, error) {
	row := r.db.QueryRow(ctx, "SELECT "+taskSelectColumns+" FROM tasks WHERE id = ?", taskID)
	task, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: task %s", ErrNotFound, taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query task %s: %w", taskID, err)
	}
	return task, nil
}

func (r *TaskRepository) GetTasksByPlan(ctx context.Context, planSlug string) ([]domain.Task, error) {
	rows, err := r.db.Query(ctx,
		"SELECT "+taskSelectColumns+" FROM tasks WHERE plan_slug = ? ORDER BY id ASC", planSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks by plan: %w", err)
	}
	defer rows.Close()

	tasks := []domain.Task{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, *task)
	}
	return tasks, rows.Err()
}

func (r *TaskRepository) ListTasks(ctx context.Context, filter domain.TaskFilter) ([]domain.Task, error) {
	query := "SELECT " + taskSelectColumns + " FROM tasks"
	var conditions []string
	var args []any
	if filter.OrphanOnly {
		conditions = append(conditions, "project_slug IS NULL AND plan_slug IS NULL")
	} else {
		if filter.ProjectSlug != "" {
			conditions = append(conditions, "project_slug = ?")
			args = append(args, filter.ProjectSlug)
		}
		if filter.PlanSlug != "" {
			conditions = append(conditions, "plan_slug = ?")
			args = append(args, filter.PlanSlug)
		}
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, string(filter.Status))
	}
	if filter.AgentSlug != "" {
		conditions = append(conditions, "agent_slug = ?")
		args = append(args, filter.AgentSlug)
	}
	if len(conditions) > 0 {
		query += " WHERE " + joinConditions(conditions)
	}
	query += " ORDER BY created_at ASC, id ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	tasks := []domain.Task{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}
		tasks = append(tasks, *task)
	}
	return tasks, rows.Err()
}

func (r *TaskRepository) CreateTasks(ctx context.Context, tasks []domain.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		// Resolve every scope before comparing containers so a plan task with an
		// empty project_slug can still join the rest of its plan.
		for i := range tasks {
			if err := resolveTaskScopeTx(ctx, tx, &tasks[i]); err != nil {
				return err
			}
			if tasks[i].ProjectSlug != tasks[0].ProjectSlug || tasks[i].PlanSlug != tasks[0].PlanSlug {
				return fmt.Errorf("%w: batch spans containers %s and %s", ErrInvalidGraph,
					tasks[0].ContainerKey(), tasks[i].ContainerKey())
			}
			if err := insertTaskTx(ctx, tx, &tasks[i], now); err != nil {
				return err
			}
		}
		if err := validateContainerGraphTx(ctx, tx, tasks[0].ProjectSlug, tasks[0].PlanSlug); err != nil {
			return err
		}
		return resyncContainerStatuses(ctx, tx, tasks[0].ProjectSlug, tasks[0].PlanSlug, tasks, now)
	})
}

func (r *TaskRepository) UpdateTaskMetadata(ctx context.Context, taskID string, patch domain.TaskPatch) error {
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		task, err := getTaskTx(ctx, tx, taskID)
		if err != nil {
			return err
		}
		if patch.Title != nil || patch.Description != nil || patch.Dependencies != nil {
			if task.Status != domain.StatusReady && task.Status != domain.StatusPending {
				return fmt.Errorf("%w: task %s is %s; metadata can only change while READY or PENDING", ErrInvalidState, taskID, task.Status)
			}
		}

		applyTaskPatch(task, patch)
		if err := resolveTaskScopeTx(ctx, tx, task); err != nil {
			return err
		}
		if err := task.Validate(); err != nil {
			return err
		}
		if err := assertDependenciesInContainer(ctx, tx, task); err != nil {
			return err
		}
		if err := updateTaskAuthoringTx(ctx, tx, task, now); err != nil {
			return err
		}
		if err := validateContainerGraphTx(ctx, tx, task.ProjectSlug, task.PlanSlug); err != nil {
			return err
		}
		return resyncContainerStatuses(ctx, tx, task.ProjectSlug, task.PlanSlug, []domain.Task{*task}, now)
	})
}

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
	if patch.AgentSlug != nil {
		task.AgentSlug = *patch.AgentSlug
	}
	if patch.RequiredSkills != nil {
		task.RequiredSkills = *patch.RequiredSkills
	}
	if patch.RequiredTools != nil {
		task.RequiredTools = *patch.RequiredTools
	}
	if patch.RequiredDocs != nil {
		task.RequiredDocs = *patch.RequiredDocs
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
	if patch.Phases != nil {
		task.Phases = *patch.Phases
	}
	if patch.CurrentPhase != nil {
		task.CurrentPhase = *patch.CurrentPhase
	}
}

func updateTaskAuthoringTx(ctx context.Context, tx *sql.Tx, task *domain.Task, now time.Time) error {
	task.UpdatedAt = now
	depsJSON, _ := json.Marshal(task.Dependencies)
	skillsJSON, _ := json.Marshal(task.RequiredSkills)
	toolsJSON, _ := json.Marshal(task.RequiredTools)
	docsJSON, _ := json.Marshal(task.RequiredDocs)
	filesJSON, _ := json.Marshal(task.TargetFiles)
	criteriaJSON, _ := json.Marshal(task.AcceptanceCriteria)
	verifJSON, _ := json.Marshal(task.VerificationCommands)
	phasesJSON, _ := json.Marshal(task.Phases)
	if task.Phases == nil {
		phasesJSON = []byte("[]")
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE tasks SET
			title = ?, description = ?, dependencies = ?, priority = ?, complexity = ?,
			risk_level = ?, agent_slug = ?, required_skills = ?, required_tools = ?,
			required_docs = ?, target_files = ?, acceptance_criteria = ?,
			verification_commands = ?, harness = ?, worktree_path = ?,
			phases = ?, current_phase = ?, updated_at = ?
		WHERE id = ?`,
		task.Title, task.Description, string(depsJSON), string(task.Priority), string(task.Complexity),
		string(task.RiskLevel), task.AgentSlug, string(skillsJSON), string(toolsJSON),
		string(docsJSON), string(filesJSON), string(criteriaJSON), string(verifJSON),
		task.Harness, task.WorktreePath, string(phasesJSON), task.CurrentPhase, now, task.ID)
	if err != nil {
		return fmt.Errorf("failed to update task %s: %w", task.ID, err)
	}
	rows, err := affectedOrErr(res, "update task", task.ID)
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("%w: task %s", ErrNotFound, task.ID)
	}
	return nil
}

func getTaskTx(ctx context.Context, tx *sql.Tx, taskID string) (*domain.Task, error) {
	row := tx.QueryRowContext(ctx, "SELECT "+taskSelectColumns+" FROM tasks WHERE id = ?", taskID)
	task, err := scanTask(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: task %s", ErrNotFound, taskID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query task %s: %w", taskID, err)
	}
	return task, nil
}

func (r *TaskRepository) UpdateTaskDependencies(ctx context.Context, taskID string, deps []string) error {
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		task, err := getTaskTx(ctx, tx, taskID)
		if err != nil {
			return err
		}
		if task.Status != domain.StatusReady && task.Status != domain.StatusPending && task.Status != domain.StatusBlocked {
			return fmt.Errorf("%w: task %s is %s; dependencies can only change while READY, PENDING, or BLOCKED", ErrInvalidState, taskID, task.Status)
		}
		task.Dependencies = deps
		if err := assertDependenciesInContainer(ctx, tx, task); err != nil {
			return err
		}

		depsJSON, err := json.Marshal(deps)
		if err != nil {
			return fmt.Errorf("failed to encode dependencies for task %s: %w", taskID, err)
		}
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
		if err := validateContainerGraphTx(ctx, tx, task.ProjectSlug, task.PlanSlug); err != nil {
			return err
		}
		return resyncContainerStatuses(ctx, tx, task.ProjectSlug, task.PlanSlug, []domain.Task{*task}, now)
	})
}

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

func (r *TaskRepository) AppendTaskNotes(ctx context.Context, taskID string, note string) error {
	if note == "" {
		return nil
	}
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		return appendTaskNotesTx(ctx, tx, taskID, note, now)
	})
}

func appendTaskNotesTx(ctx context.Context, tx *sql.Tx, taskID string, note string, now time.Time) error {
	res, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET notes = CASE WHEN notes = '' THEN ? ELSE notes || char(10) || ? END, updated_at = ?
		WHERE id = ?`, note, note, now, taskID)
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
