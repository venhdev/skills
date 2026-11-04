package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zenfd/internal/domain"
	"zenfd/internal/domain/prompt"
)

var (
	ErrNotFound       = errors.New("entity not found")
	ErrInvalidState   = errors.New("invalid state transition")
	ErrConflictStatus = errors.New("task status conflict during transition")
)

var _ domain.StorageEngine = (*Store)(nil)

// Store implements domain.StorageEngine using the SQLite WAL DB.
type Store struct {
	db             *DB
	promptCompiler prompt.Compiler
}

// NewStore creates a new domain.StorageEngine backed by the SQLite WAL DB.
func NewStore(db *DB) *Store {
	return &Store{
		db:             db,
		promptCompiler: prompt.NewCompiler(),
	}
}

// StateFingerprint returns a cheap fingerprint of the plans/tasks state. The
// gateway polls it to detect commits made by other processes (for example the
// stdio MCP server) that never pass through the daemon event hub.
func (s *Store) StateFingerprint(ctx context.Context) (string, error) {
	var plansCount, tasksCount int64
	var plansUpdated, tasksUpdated string
	err := s.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM plans),
			(SELECT COALESCE(MAX(updated_at), '') FROM plans),
			(SELECT COUNT(*) FROM tasks),
			(SELECT COALESCE(MAX(updated_at), '') FROM tasks)
	`).Scan(&plansCount, &plansUpdated, &tasksCount, &tasksUpdated)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d|%s|%d|%s", plansCount, plansUpdated, tasksCount, tasksUpdated), nil
}

// SetPromptCompiler sets or overrides the prompt compiler for the store.
func (s *Store) SetPromptCompiler(c prompt.Compiler) {
	s.promptCompiler = c
}

func (s *Store) getPromptCompiler() prompt.Compiler {
	if s.promptCompiler != nil {
		return s.promptCompiler
	}
	return prompt.NewCompiler()
}

// Close safely closes the underlying SQLite database connection pool.
func (s *Store) Close() error {
	return s.db.Close()
}

const taskSelectCols = `
id, plan_id, title, description, dependencies,
priority, complexity, risk_level, required_skills, required_tools,
target_files, acceptance_criteria, verification_commands,
harness, skill_command, worktree_path, status, claimed_by, claimed_at,
orchestrator_id, worker_role, pid, orchestrator_pid, notes,
review_reason, review_details, review_options, review_response, resolved_by,
conflict_files, created_at, updated_at
`

type scanner interface {
	Scan(dest ...any) error
}

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

	_ = json.Unmarshal([]byte(depsJSON), &t.Dependencies)
	if t.Dependencies == nil {
		t.Dependencies = []string{}
	}
	_ = json.Unmarshal([]byte(skillsJSON), &t.RequiredSkills)
	if t.RequiredSkills == nil {
		t.RequiredSkills = []string{}
	}
	_ = json.Unmarshal([]byte(toolsJSON), &t.RequiredTools)
	if t.RequiredTools == nil {
		t.RequiredTools = []string{}
	}
	_ = json.Unmarshal([]byte(filesJSON), &t.TargetFiles)
	if t.TargetFiles == nil {
		t.TargetFiles = []string{}
	}
	_ = json.Unmarshal([]byte(criteriaJSON), &t.AcceptanceCriteria)
	if t.AcceptanceCriteria == nil {
		t.AcceptanceCriteria = []string{}
	}
	_ = json.Unmarshal([]byte(verifJSON), &t.VerificationCommands)
	if t.VerificationCommands == nil {
		t.VerificationCommands = []string{}
	}
	_ = json.Unmarshal([]byte(optionsJSON), &t.ReviewOptions)
	if t.ReviewOptions == nil {
		t.ReviewOptions = []string{}
	}
	_ = json.Unmarshal([]byte(conflictJSON), &t.ConflictFiles)
	if t.ConflictFiles == nil {
		t.ConflictFiles = []string{}
	}

	return &t, nil
}

const taskUpsertQuery = `
INSERT INTO tasks (
	id, plan_id, title, description, dependencies,
	priority, complexity, risk_level, required_skills, required_tools,
	target_files, acceptance_criteria, verification_commands,
	harness, skill_command, worktree_path, status, claimed_by, claimed_at,
	orchestrator_id, worker_role, pid, orchestrator_pid, notes,
	review_reason, review_details, review_options, review_response, resolved_by,
	conflict_files, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	title = excluded.title,
	description = excluded.description,
	dependencies = excluded.dependencies,
	priority = excluded.priority,
	complexity = excluded.complexity,
	risk_level = excluded.risk_level,
	required_skills = excluded.required_skills,
	required_tools = excluded.required_tools,
	target_files = excluded.target_files,
	acceptance_criteria = excluded.acceptance_criteria,
	verification_commands = excluded.verification_commands,
	harness = excluded.harness,
	skill_command = excluded.skill_command,
	worktree_path = excluded.worktree_path,
	status = excluded.status,
	claimed_by = excluded.claimed_by,
	claimed_at = excluded.claimed_at,
	orchestrator_id = excluded.orchestrator_id,
	worker_role = excluded.worker_role,
	pid = excluded.pid,
	orchestrator_pid = excluded.orchestrator_pid,
	notes = excluded.notes,
	review_reason = excluded.review_reason,
	review_details = excluded.review_details,
	review_options = excluded.review_options,
	review_response = excluded.review_response,
	resolved_by = excluded.resolved_by,
	conflict_files = excluded.conflict_files,
	updated_at = excluded.updated_at
`

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

	_, err := tx.ExecContext(ctx, taskUpsertQuery,
		task.ID, planID, task.Title, task.Description, string(depsJSON),
		priority, complexity, risk, string(skillsJSON), string(toolsJSON),
		string(filesJSON), string(criteriaJSON), string(verifJSON),
		task.Harness, task.SkillCommand, task.WorktreePath, string(task.Status),
		task.ClaimedBy, task.ClaimedAt,
		task.OrchestratorID, task.WorkerRole, task.PID, task.OrchestratorPID, task.Notes,
		task.ReviewReason, task.ReviewDetails, string(optionsJSON), task.ReviewResponse, task.ResolvedBy,
		string(conflictJSON), task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert task %s: %w", task.ID, err)
	}
	return nil
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
		INSERT INTO plans (slug, title, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(slug) DO UPDATE SET
			title = excluded.title,
			status = excluded.status,
			updated_at = excluded.updated_at
		`
		_, err := tx.ExecContext(ctx, planQuery, plan.Slug, plan.Title, plan.Status, plan.CreatedAt, plan.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to upsert plan: %w", err)
		}

		// Insert or update tasks via centralized upsertTaskTx
		for _, task := range tasks {
			if err := upsertTaskTx(ctx, tx, &task, plan.Slug, now); err != nil {
				return err
			}
		}

		return nil
	})
}

// GetPlan retrieves a plan by slug.
func (s *Store) GetPlan(ctx context.Context, slug string) (*domain.Plan, error) {
	query := `SELECT slug, title, status, created_at, updated_at FROM plans WHERE slug = ?`
	row := s.db.sqlDB.QueryRowContext(ctx, query, slug)

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
func (s *Store) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	query := `SELECT slug, title, status, created_at, updated_at FROM plans ORDER BY slug ASC`
	rows, err := s.db.sqlDB.QueryContext(ctx, query)
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

// GetTask retrieves a single task by its unique ID.
func (s *Store) GetTask(ctx context.Context, taskID string) (*domain.Task, error) {
	query := fmt.Sprintf("SELECT %s FROM tasks WHERE id = ?", taskSelectCols)
	row := s.db.sqlDB.QueryRowContext(ctx, query, taskID)
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
func (s *Store) GetTasksByPlan(ctx context.Context, planSlug string) ([]domain.Task, error) {
	query := fmt.Sprintf(`
	SELECT %s
	FROM tasks
	WHERE plan_id = ?
	ORDER BY id ASC
	`, taskSelectCols)

	rows, err := s.db.sqlDB.QueryContext(ctx, query, planSlug)
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
func (s *Store) GetReadyTasks(ctx context.Context, limit int) ([]domain.Task, error) {
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

	rows, err := s.db.sqlDB.QueryContext(ctx, query, limit)
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
func (s *Store) ListTasks(ctx context.Context, status domain.TaskStatus) ([]domain.Task, error) {
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

	rows, err := s.db.sqlDB.QueryContext(ctx, query, args...)
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

			// Query all PENDING or BLOCKED downstream tasks in this plan
			rows, err := tx.QueryContext(ctx, "SELECT id, dependencies FROM tasks WHERE plan_id = ? AND status IN ('PENDING', 'BLOCKED')", planID)
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

			if len(pendingList) > 0 {
				// Query all task statuses in this plan in a single query (eliminating N+1 loop)
				statusRows, err := tx.QueryContext(ctx, "SELECT id, status FROM tasks WHERE plan_id = ?", planID)
				if err != nil {
					return err
				}
				defer statusRows.Close()

				taskStatuses := make(map[string]domain.TaskStatus)
				for statusRows.Next() {
					var id, st string
					if err := statusRows.Scan(&id, &st); err == nil {
						taskStatuses[id] = domain.TaskStatus(st)
					}
				}
				statusRows.Close()

				// For each pending task, verify if ALL dependencies are COMPLETED in memory
				for _, pt := range pendingList {
					allCompleted := true
					for _, depID := range pt.deps {
						if taskStatuses[depID] != domain.StatusCompleted {
							allCompleted = false
							break
						}
					}

					if allCompleted {
						_, err := tx.ExecContext(ctx, "UPDATE tasks SET status = 'READY', updated_at = ? WHERE id = ?", now, pt.id)
						if err != nil {
							return fmt.Errorf("failed to unlock ready task %s: %w", pt.id, err)
						}
						taskStatuses[pt.id] = domain.StatusReady
					}
				}
			}
		}

		// If task has FAILED, cascade block downstream dependent tasks (Fast-Fail branch)
		if to == domain.StatusFailed {
			var planID string
			_ = tx.QueryRowContext(ctx, "SELECT plan_id FROM tasks WHERE id = ?", taskID).Scan(&planID)
			if planID != "" {
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

// ClaimTask atomically verifies readiness, sets claim lock (workerID & pid), transitions to RUNNING, and returns ContextPacket.
func (s *Store) ClaimTask(ctx context.Context, taskID string, workerID string, pid int) (*domain.ContextPacket, error) {
	return s.ClaimTaskWithMeta(ctx, taskID, workerID, "", "", pid, 0)
}

// ClaimTaskWithMeta atomically claims a ready task with full metadata.
func (s *Store) ClaimTaskWithMeta(ctx context.Context, taskID string, workerID string, orchestratorID string, workerRole string, pid int, orchestratorPID int) (*domain.ContextPacket, error) {
	if workerID == "" {
		workerID = fmt.Sprintf("worker-%d", time.Now().UnixNano())
	}
	now := time.Now().UTC()

	var packet *domain.ContextPacket
	err := s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		query := fmt.Sprintf("SELECT %s FROM tasks WHERE id = ?", taskSelectCols)
		row := tx.QueryRowContext(ctx, query, taskID)
		t, err := scanTask(row)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("failed to load task %s: %w", taskID, err)
		}

		if t.Status != domain.StatusReady {
			return fmt.Errorf("%w: task %s is not in READY status (current: %s)", ErrConflictStatus, taskID, t.Status)
		}

		// Update to RUNNING with claim lock and metadata
		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = 'RUNNING', claimed_by = ?, orchestrator_id = ?, worker_role = ?, pid = ?, orchestrator_pid = ?, claimed_at = ?, updated_at = ?
			WHERE id = ? AND status = 'READY'
		`, workerID, orchestratorID, workerRole, pid, orchestratorPID, now, now, taskID)
		if err != nil {
			return fmt.Errorf("failed to claim task: %w", err)
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: task %s was already claimed by another worker", ErrConflictStatus, taskID)
		}

		t.Status = domain.StatusRunning
		t.ClaimedBy = workerID
		t.OrchestratorID = orchestratorID
		t.WorkerRole = workerRole
		t.PID = pid
		t.OrchestratorPID = orchestratorPID
		t.ClaimedAt = &now
		t.UpdatedAt = now

		// If pid > 0, register Run attempt
		if pid > 0 {
			runID := fmt.Sprintf("run-%s-%d", taskID, now.Unix())
			logPath := filepath.Join(os.TempDir(), fmt.Sprintf("%s.log", runID))
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO runs (id, task_id, pid, pgid, status, log_path, exit_code, started_at, last_beat_at)
				VALUES (?, ?, ?, ?, 'RUNNING', ?, 0, ?, ?)
			`, runID, taskID, pid, pid, logPath, now, now)
		}

		packet, err = s.buildContextPacket(ctx, *t)
		return err
	})

	if err != nil {
		return nil, err
	}
	return packet, nil
}

// EvictTask evicts a running or suspended task, resetting it to READY or cancelling it.
func (s *Store) EvictTask(ctx context.Context, taskID string, reason string, action string) error {
	now := time.Now().UTC()
	targetStatus := domain.StatusReady
	if action == "CANCEL" {
		targetStatus = domain.StatusCancelled
	}

	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = ?, claimed_by = '', pid = 0, notes = CASE WHEN notes = '' THEN ? ELSE notes || '\n' || ? END, updated_at = ?
			WHERE id = ? AND (status = 'RUNNING' OR status = 'SUSPENDED')
		`, string(targetStatus), "Eviction: "+reason, "Eviction: "+reason, now, taskID)
		if err != nil {
			return err
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return fmt.Errorf("%w: task %s is not in RUNNING or SUSPENDED status", ErrConflictStatus, taskID)
		}
		return nil
	})
}

// RequestReview pauses execution, records review reasons/options, and transitions task to AWAITING_REVIEW.
func (s *Store) RequestReview(ctx context.Context, taskID string, reason string, details string, options []string) error {
	now := time.Now().UTC()
	optionsJSON, err := json.Marshal(options)
	if err != nil {
		optionsJSON = []byte("[]")
	}

	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
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
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return fmt.Errorf("%w: failed to transition task %s to AWAITING_REVIEW", ErrConflictStatus, taskID)
		}
		return nil
	})
}

// ResolveReview records the human guidance relayed by the main agent and transitions task back to READY.
func (s *Store) ResolveReview(ctx context.Context, taskID string, response string) error {
	now := time.Now().UTC()
	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
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
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return fmt.Errorf("%w: failed to resolve review for task %s", ErrConflictStatus, taskID)
		}
		return nil
	})
}

// GetTaskContextPacket compiles full context packet (task, target files, skill paths, media).
func (s *Store) GetTaskContextPacket(ctx context.Context, taskID string) (*domain.ContextPacket, error) {
	t, err := s.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return s.buildContextPacket(ctx, *t)
}

// buildContextPacket generates a complete prompt-ready context packet for an agent.
func (s *Store) buildContextPacket(ctx context.Context, task domain.Task) (*domain.ContextPacket, error) {
	skillPaths := []string{}

	// 1. Resolve Skill Paths relative to the task worktree when one is configured.
	// Without a repo binding, the daemon cannot discover a repository root; agents
	// resolve bare skill names through their own skill registry.
	if task.WorktreePath != "" {
		for _, skill := range task.RequiredSkills {
			localPath := filepath.Join(task.WorktreePath, ".agents", "skills", skill, "SKILL.md")
			if _, err := os.Stat(localPath); err == nil {
				skillPaths = append(skillPaths, localPath)
			} else {
				skillPaths = append(skillPaths, skill)
			}
		}
	} else {
		skillPaths = append(skillPaths, task.RequiredSkills...)
	}

	// 2. Resolve Input Attachments
	attachments, err := s.GetArtifactsByTask(ctx, task.ID, domain.ArtifactRoleInput)
	if err != nil {
		attachments = []domain.Artifact{}
	}

	// 3. Delegate Prompt Context Compilation to domain Compiler
	packet, err := s.getPromptCompiler().Compile(ctx, task, attachments)
	if err != nil {
		return nil, err
	}
	packet.SkillPaths = skillPaths
	return packet, nil
}

// CreatePlan inserts a single plan.
func (s *Store) CreatePlan(ctx context.Context, plan domain.Plan) error {
	now := time.Now().UTC()
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	plan.UpdatedAt = now
	if plan.Status == "" {
		plan.Status = "ACTIVE"
	}

	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
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

// DeletePlan deletes a plan and cascades to its tasks, runs, artifacts, and docs.
func (s *Store) DeletePlan(ctx context.Context, slug string) error {
	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "DELETE FROM plans WHERE slug = ?", slug)
		if err != nil {
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return ErrNotFound
		}
		return nil
	})
}

// CreateTask inserts a single task.
func (s *Store) CreateTask(ctx context.Context, task domain.Task) error {
	now := time.Now().UTC()
	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		return upsertTaskTx(ctx, tx, &task, task.PlanID, now)
	})
}

// UpdateTaskDependencies updates the task dependency list.
func (s *Store) UpdateTaskDependencies(ctx context.Context, taskID string, deps []string) error {
	now := time.Now().UTC()
	depsJSON, _ := json.Marshal(deps)
	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "UPDATE tasks SET dependencies = ?, updated_at = ? WHERE id = ?", string(depsJSON), now, taskID)
		return err
	})
}

// UpdateTaskNotes updates developer supplemental notes.
func (s *Store) UpdateTaskNotes(ctx context.Context, taskID string, notes string) error {
	now := time.Now().UTC()
	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, "UPDATE tasks SET notes = ?, updated_at = ? WHERE id = ?", notes, now, taskID)
		return err
	})
}

// AppendTaskNotes appends a note to the task's supplemental notes.
func (s *Store) AppendTaskNotes(ctx context.Context, taskID string, note string) error {
	if note == "" {
		return nil
	}
	now := time.Now().UTC()
	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET notes = CASE WHEN notes = '' THEN ? ELSE notes || '\n' || ? END, updated_at = ?
			WHERE id = ?
		`, note, note, now, taskID)
		return err
	})
}

// AddInputAttachment adds an input artifact (media, screenshot, or spec) to a task.
func (s *Store) AddInputAttachment(ctx context.Context, taskID string, name string, filePath string, mimeType string, byteSize int64, sha256 string) (*domain.Artifact, error) {
	now := time.Now().UTC()
	artID := fmt.Sprintf("art-%s-%d", taskID, now.UnixNano())
	artType := domain.ArtifactTypeImage
	if !strings.HasPrefix(mimeType, "image/") {
		artType = domain.ArtifactTypeDoc
	}

	art := domain.Artifact{
		ID:        artID,
		TaskID:    taskID,
		Role:      domain.ArtifactRoleInput,
		Name:      name,
		Type:      artType,
		FilePath:  filePath,
		MimeType:  mimeType,
		ByteSize:  byteSize,
		Sha256:    sha256,
		CreatedAt: now,
	}

	err := s.RecordArtifact(ctx, art)
	if err != nil {
		return nil, err
	}
	return &art, nil
}
