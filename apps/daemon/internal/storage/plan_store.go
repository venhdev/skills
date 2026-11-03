package storage

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"zenfd/internal/domain"
)

var (
	ErrNotFound       = errors.New("entity not found")
	ErrInvalidState   = errors.New("invalid state transition")
	ErrConflictStatus = errors.New("task status conflict during transition")
)

var _ domain.StorageEngine = (*Store)(nil)

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

const taskSelectCols = `
id, plan_id, repo_path, title, description, dependencies,
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
		&t.ID, &t.PlanID, &t.RepoPath, &t.Title, &t.Description, &depsJSON,
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
		INSERT INTO tasks (
			id, plan_id, repo_path, title, description, dependencies,
			priority, complexity, risk_level, required_skills, required_tools,
			target_files, acceptance_criteria, verification_commands,
			harness, skill_command, worktree_path, status, claimed_by, claimed_at,
			orchestrator_id, worker_role, pid, orchestrator_pid, notes,
			review_reason, review_details, review_options, review_response, resolved_by,
			conflict_files, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

		for _, task := range tasks {
			if err := task.Validate(); err != nil {
				return err
			}
			if task.CreatedAt.IsZero() {
				task.CreatedAt = now
			}
			task.UpdatedAt = now

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

			_, err = tx.ExecContext(ctx, taskQuery,
				task.ID, plan.Slug, NormalizePath(task.RepoPath), task.Title, task.Description, string(depsJSON),
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

// GetReadyTasks returns tasks whose status is READY, filtered optionally by repoPath.
func (s *Store) GetReadyTasks(ctx context.Context, repoPath string, limit int) ([]domain.Task, error) {
	if limit <= 0 {
		limit = 10
	}

	var query string
	var args []any

	if repoPath != "" {
		query = fmt.Sprintf(`
		SELECT %s
		FROM tasks
		WHERE status = 'READY' AND repo_path = ?
		ORDER BY created_at ASC
		LIMIT ?
		`, taskSelectCols)
		args = []any{repoPath, limit}
	} else {
		query = fmt.Sprintf(`
		SELECT %s
		FROM tasks
		WHERE status = 'READY'
		ORDER BY created_at ASC
		LIMIT ?
		`, taskSelectCols)
		args = []any{limit}
	}

	rows, err := s.db.sqlDB.QueryContext(ctx, query, args...)
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

// ListTasks returns tasks filtered optionally by repoPath and/or status.
func (s *Store) ListTasks(ctx context.Context, repoPath string, status domain.TaskStatus) ([]domain.Task, error) {
	var query string
	var args []any

	if repoPath != "" && status != "" {
		query = fmt.Sprintf(`
		SELECT %s
		FROM tasks
		WHERE repo_path = ? AND status = ?
		ORDER BY created_at ASC
		`, taskSelectCols)
		args = []any{repoPath, string(status)}
	} else if repoPath != "" {
		query = fmt.Sprintf(`
		SELECT %s
		FROM tasks
		WHERE repo_path = ?
		ORDER BY created_at ASC
		`, taskSelectCols)
		args = []any{repoPath}
	} else if status != "" {
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
				INSERT INTO runs (id, task_id, repo_path, pid, pgid, status, log_path, exit_code, started_at, last_beat_at)
				VALUES (?, ?, ?, ?, ?, 'RUNNING', ?, 0, ?, ?)
			`, runID, taskID, t.RepoPath, pid, pid, logPath, now, now)
		}

		packet, err = s.buildContextPacket(ctx, tx, *t)
		return err
	})

	if err != nil {
		return nil, err
	}
	return packet, nil
}

// ReclaimTask re-claims a SUSPENDED or failed task with a new worker and orchestrator.
func (s *Store) ReclaimTask(ctx context.Context, taskID string, newWorkerID string, newOrchestratorID string, newPID int) (*domain.ContextPacket, error) {
	if newWorkerID == "" {
		newWorkerID = fmt.Sprintf("reclaimed-worker-%d", time.Now().UnixNano())
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
			return err
		}

		if t.Status != domain.StatusSuspended && t.Status != domain.StatusReady {
			return fmt.Errorf("%w: task %s is not SUSPENDED or READY (current: %s)", ErrConflictStatus, taskID, t.Status)
		}

		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = 'RUNNING', claimed_by = ?, orchestrator_id = ?, pid = ?, claimed_at = ?, updated_at = ?
			WHERE id = ? AND (status = 'SUSPENDED' OR status = 'READY')
		`, newWorkerID, newOrchestratorID, newPID, now, now, taskID)
		if err != nil {
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: failed to reclaim task %s", ErrConflictStatus, taskID)
		}

		t.Status = domain.StatusRunning
		t.ClaimedBy = newWorkerID
		t.OrchestratorID = newOrchestratorID
		t.PID = newPID
		t.ClaimedAt = &now
		t.UpdatedAt = now

		p, err := s.buildContextPacket(ctx, tx, *t)
		if err != nil {
			return err
		}
		packet = p
		return nil
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

// ResolveReview answers an awaiting review, records resolution, and transitions task back to READY.
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
			SET status = 'READY', review_response = ?, claimed_by = '', pid = 0, resolved_by = 'operator', updated_at = ?
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

// StageMerge marks a worktree task as complete and transitions it to AWAITING_MERGE.
func (s *Store) StageMerge(ctx context.Context, taskID string) error {
	return s.UpdateTaskStatus(ctx, taskID, domain.StatusRunning, domain.StatusAwaitingMerge)
}

// MergeTask executes git merge from worktree to target branch, transitioning to COMPLETED or CONFLICT.
func (s *Store) MergeTask(ctx context.Context, taskID string) (bool, []string, error) {
	t, err := s.GetTask(ctx, taskID)
	if err != nil {
		return false, nil, err
	}

	if t.Status != domain.StatusAwaitingMerge && t.Status != domain.StatusRunning && t.Status != domain.StatusConflict {
		return false, nil, fmt.Errorf("%w: task %s is not ready to merge (current status: %s)", ErrInvalidState, taskID, t.Status)
	}

	// If WorktreePath is specified and exists, test/execute git integration
	if t.WorktreePath != "" {
		if fi, err := os.Stat(t.WorktreePath); err == nil && fi.IsDir() {
			branchCmd := exec.CommandContext(ctx, "git", "-C", t.WorktreePath, "rev-parse", "--abbrev-ref", "HEAD")
			out, err := branchCmd.Output()
			if err == nil {
				branch := strings.TrimSpace(string(out))
				if branch != "" && branch != "HEAD" {
					mergeCmd := exec.CommandContext(ctx, "git", "-C", t.RepoPath, "merge", "--no-ff", "--no-commit", branch)
					if mergeOut, err := mergeCmd.CombinedOutput(); err != nil {
						// Merge conflict detected!
						conflictCmd := exec.CommandContext(ctx, "git", "-C", t.RepoPath, "diff", "--name-only", "--diff-filter=U")
						confOut, _ := conflictCmd.Output()
						_ = exec.CommandContext(ctx, "git", "-C", t.RepoPath, "merge", "--abort").Run()

						var conflictFiles []string
						for _, line := range strings.Split(string(confOut), "\n") {
							line = strings.TrimSpace(line)
							if line != "" {
								conflictFiles = append(conflictFiles, line)
							}
						}
						if len(conflictFiles) == 0 {
							conflictFiles = []string{strings.TrimSpace(string(mergeOut))}
						}

						conflictJSON, _ := json.Marshal(conflictFiles)
						now := time.Now().UTC()
						_ = s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
							_, err := tx.ExecContext(ctx, `
								UPDATE tasks
								SET status = 'CONFLICT', conflict_files = ?, updated_at = ?
								WHERE id = ?
							`, string(conflictJSON), now, taskID)
							return err
						})

						return false, conflictFiles, nil
					} else {
						// Clean merge succeeded
						commitCmd := exec.CommandContext(ctx, "git", "-C", t.RepoPath, "commit", "-m", fmt.Sprintf("zenfd: integrate task %s (%s)", t.ID, t.Title))
						_ = commitCmd.Run()
					}
				}
			}
		}
	}

	// Update task status to COMPLETED (which automatically unlocks downstream DAG in UpdateTaskStatus!)
	err = s.UpdateTaskStatus(ctx, taskID, t.Status, domain.StatusCompleted)
	if err != nil {
		return false, nil, err
	}

	return true, nil, nil
}

// GetTaskContextPacket compiles full context packet (task, target files, skill paths, upstream diffs, media).
func (s *Store) GetTaskContextPacket(ctx context.Context, taskID string) (*domain.ContextPacket, error) {
	t, err := s.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return s.buildContextPacket(ctx, nil, *t)
}

// buildContextPacket generates a complete prompt-ready context packet for an agent.
func (s *Store) buildContextPacket(ctx context.Context, tx *sql.Tx, task domain.Task) (*domain.ContextPacket, error) {
	packet := &domain.ContextPacket{
		Task:                 task,
		TargetFiles:          task.TargetFiles,
		VerificationCommands: task.VerificationCommands,
		SkillPaths:           []string{},
		UpstreamDiffs:        []string{},
		InputAttachments:     []domain.Artifact{},
	}

	// 1. Resolve Skill Paths
	for _, skill := range task.RequiredSkills {
		localPath := filepath.Join(task.RepoPath, ".agents", "skills", skill, "SKILL.md")
		if _, err := os.Stat(localPath); err == nil {
			packet.SkillPaths = append(packet.SkillPaths, localPath)
		} else {
			packet.SkillPaths = append(packet.SkillPaths, skill)
		}
	}

	// 2. Resolve Input Attachments
	attachments, err := s.GetArtifactsByTask(ctx, task.ID, domain.ArtifactRoleInput)
	if err == nil {
		packet.InputAttachments = attachments
	}

	// 3. Resolve Upstream Diffs
	for _, depID := range task.Dependencies {
		diffArtifacts, err := s.GetArtifactsByTask(ctx, depID, domain.ArtifactRoleOutput)
		if err == nil {
			for _, art := range diffArtifacts {
				if art.Type == domain.ArtifactTypeDiff && art.FilePath != "" {
					if content, err := os.ReadFile(art.FilePath); err == nil {
						packet.UpstreamDiffs = append(packet.UpstreamDiffs, string(content))
					}
				}
			}
		}
	}

	// 3.5. Resolve Linked Specs (Zero-Round-Trip Context Injection)
	packet.LinkedSpecs = []domain.CompactSpecPayload{}
	var linkedSlugs []string
	if tx != nil {
		linkedSlugs, _ = s.autoLinkSpecsTx(ctx, tx, task.TargetFiles)
		for _, slug := range linkedSlugs {
			spec, err := s.getSpecTx(ctx, tx, slug)
			if err == nil && spec != nil {
				packet.LinkedSpecs = append(packet.LinkedSpecs, domain.CompactSpecPayload{
					Slug:        spec.Slug,
					Kind:        spec.Kind,
					Title:       spec.Title,
					DiagramCode: spec.DiagramCode,
					Payload:     spec.StructuredPayload,
				})
			}
		}
	} else {
		linkedSlugs, _ = s.AutoLinkSpecs(ctx, task.TargetFiles)
		for _, slug := range linkedSlugs {
			spec, err := s.GetSpec(ctx, slug)
			if err == nil && spec != nil {
				packet.LinkedSpecs = append(packet.LinkedSpecs, domain.CompactSpecPayload{
					Slug:        spec.Slug,
					Kind:        spec.Kind,
					Title:       spec.Title,
					DiagramCode: spec.DiagramCode,
					Payload:     spec.StructuredPayload,
				})
			}
		}
	}

	// 4. Generate Structured Prompt Markdown
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Task Assignment: %s (%s)\n\n", task.Title, task.ID))
	sb.WriteString(fmt.Sprintf("- **Plan**: `%s`\n", task.PlanID))
	sb.WriteString(fmt.Sprintf("- **Priority**: `%s` | **Complexity**: `%s` | **Risk**: `%s`\n", task.Priority, task.Complexity, task.RiskLevel))
	if task.WorktreePath != "" {
		sb.WriteString(fmt.Sprintf("- **Worktree Path**: `%s`\n", task.WorktreePath))
	}
	sb.WriteString("\n## 1. Description & Context\n")
	if task.Description != "" {
		sb.WriteString(task.Description + "\n\n")
	} else {
		sb.WriteString("Execute the implementation according to plan specifications.\n\n")
	}

	if len(packet.LinkedSpecs) > 0 {
		sb.WriteString("## 1.5. Linked Architectural Blueprints & Contracts (SSOT)\n")
		for _, ls := range packet.LinkedSpecs {
			sb.WriteString(fmt.Sprintf("### Spec `%s`: %s (%s)\n", ls.Slug, ls.Title, ls.Kind))
			if ls.DiagramCode != "" {
				sb.WriteString(fmt.Sprintf("```mermaid\n%s\n```\n", ls.DiagramCode))
			}
		}
		sb.WriteString("\n")
	}

	if len(task.AcceptanceCriteria) > 0 {
		sb.WriteString("## 2. Acceptance Criteria\n")
		for _, ac := range task.AcceptanceCriteria {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", ac))
		}
		sb.WriteString("\n")
	}

	if len(task.TargetFiles) > 0 {
		sb.WriteString("## 3. Target Files\n")
		for _, tf := range task.TargetFiles {
			sb.WriteString(fmt.Sprintf("- `%s`\n", tf))
		}
		sb.WriteString("\n")
	}

	if len(task.VerificationCommands) > 0 {
		sb.WriteString("## 4. Verification Commands\n")
		sb.WriteString("Run the following commands to verify implementation correctness before completing:\n```bash\n")
		for _, vc := range task.VerificationCommands {
			sb.WriteString(vc + "\n")
		}
		sb.WriteString("```\n\n")
	}

	sb.WriteString("## 5. MCP Tool Handshake Protocol\n")
	sb.WriteString(fmt.Sprintf("- **Claim**: Call `zenfd_claim_task(task_id=\"%s\", worker_role=\"CODER\", pid=$PID)` before editing files.\n", task.ID))
	sb.WriteString(fmt.Sprintf("- **Architectural Drift / Blocked**: Call `zenfd_yield_review(task_id=\"%s\", reason=\"...\", details=\"...\", options=[\"...\"])` and exit immediately.\n", task.ID))
	sb.WriteString(fmt.Sprintf("- **Stage Merge**: Once all verification commands pass with exit code 0, call `zenfd_stage_merge(task_id=\"%s\")`.\n", task.ID))

	packet.PromptMarkdown = sb.String()
	return packet, nil
}

// UpsertDoc creates or updates a project document or living architecture spec.
func (s *Store) UpsertDoc(ctx context.Context, doc domain.ProjectDoc) error {
	if doc.Slug == "" {
		return fmt.Errorf("doc slug cannot be empty")
	}
	if doc.RepoPath == "" {
		return fmt.Errorf("doc repo_path cannot be empty")
	}
	if doc.Category == "" {
		doc.Category = "ARCHITECTURE"
	}
	now := time.Now().UTC()
	doc.UpdatedAt = now

	diagramsJSON, err := json.Marshal(doc.MermaidDiagrams)
	if err != nil {
		diagramsJSON = []byte("[]")
	}

	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		query := `
		INSERT INTO project_docs (slug, repo_path, git_branch, title, category, content_markdown, mermaid_diagrams, version, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_path, slug) DO UPDATE SET
			git_branch = excluded.git_branch,
			title = excluded.title,
			category = excluded.category,
			content_markdown = excluded.content_markdown,
			mermaid_diagrams = excluded.mermaid_diagrams,
			version = project_docs.version + 1,
			updated_at = excluded.updated_at
		`
		_, err := tx.ExecContext(ctx, query,
			doc.Slug, doc.RepoPath, doc.GitBranch, doc.Title, doc.Category,
			doc.ContentMarkdown, string(diagramsJSON), 1, now,
		)
		return err
	})
}

// GetDoc retrieves a project document by repoPath and slug.
func (s *Store) GetDoc(ctx context.Context, repoPath, slug string) (*domain.ProjectDoc, error) {
	query := `
	SELECT slug, repo_path, git_branch, title, category, content_markdown, mermaid_diagrams, version, updated_at
	FROM project_docs
	WHERE repo_path = ? AND slug = ?
	`
	row := s.db.sqlDB.QueryRowContext(ctx, query, repoPath, slug)
	var doc domain.ProjectDoc
	var diagramsJSON string
	err := row.Scan(&doc.Slug, &doc.RepoPath, &doc.GitBranch, &doc.Title, &doc.Category, &doc.ContentMarkdown, &diagramsJSON, &doc.Version, &doc.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query doc: %w", err)
	}
	_ = json.Unmarshal([]byte(diagramsJSON), &doc.MermaidDiagrams)
	if doc.MermaidDiagrams == nil {
		doc.MermaidDiagrams = []string{}
	}
	return &doc, nil
}

// ListDocs lists project documents filtered optionally by category.
func (s *Store) ListDocs(ctx context.Context, repoPath, category string) ([]domain.ProjectDoc, error) {
	var query string
	var args []any
	if category != "" {
		query = `
		SELECT slug, repo_path, git_branch, title, category, content_markdown, mermaid_diagrams, version, updated_at
		FROM project_docs
		WHERE repo_path = ? AND category = ?
		ORDER BY title ASC
		`
		args = []any{repoPath, category}
	} else {
		query = `
		SELECT slug, repo_path, git_branch, title, category, content_markdown, mermaid_diagrams, version, updated_at
		FROM project_docs
		WHERE repo_path = ?
		ORDER BY title ASC
		`
		args = []any{repoPath}
	}

	rows, err := s.db.sqlDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list docs: %w", err)
	}
	defer rows.Close()

	var docs []domain.ProjectDoc
	for rows.Next() {
		var doc domain.ProjectDoc
		var diagramsJSON string
		err := rows.Scan(&doc.Slug, &doc.RepoPath, &doc.GitBranch, &doc.Title, &doc.Category, &doc.ContentMarkdown, &diagramsJSON, &doc.Version, &doc.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan doc: %w", err)
		}
		_ = json.Unmarshal([]byte(diagramsJSON), &doc.MermaidDiagrams)
		if doc.MermaidDiagrams == nil {
			doc.MermaidDiagrams = []string{}
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

// NormalizePath converts Windows or UNC paths to POSIX Linux paths for WSL compatibility.
func NormalizePath(p string) string {
	if p == "" {
		return ""
	}
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "\\", "/")

	// 1. Check for UNC WSL format: //wsl.localhost/Ubuntu/path or //wsl$/Ubuntu/path
	if strings.HasPrefix(p, "//wsl.localhost/") || strings.HasPrefix(p, "//wsl$/") {
		clean := strings.TrimPrefix(strings.TrimPrefix(p, "//wsl.localhost/"), "//wsl$/")
		idx := strings.Index(clean, "/")
		if idx != -1 {
			return filepath.Clean(clean[idx:])
		}
	}

	// 2. Check for Windows Drive letter: C:/path or D:/path
	if len(p) >= 3 && p[1] == ':' && p[2] == '/' {
		drive := strings.ToLower(string(p[0]))
		return filepath.Clean("/mnt/" + drive + "/" + p[3:])
	}

	return filepath.Clean(p)
}

// ListProjects returns all distinct repo_paths known to the store.
func (s *Store) ListProjects(ctx context.Context) ([]string, error) {
	query := `
	SELECT DISTINCT repo_path FROM plans WHERE repo_path != ''
	UNION
	SELECT DISTINCT repo_path FROM tasks WHERE repo_path != ''
	ORDER BY repo_path ASC
	`
	rows, err := s.db.sqlDB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query projects: %w", err)
	}
	defer rows.Close()

	var projects []string
	seen := make(map[string]bool)
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err == nil && path != "" {
			norm := NormalizePath(path)
			if !seen[norm] {
				seen[norm] = true
				projects = append(projects, norm)
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// If no projects in database yet, automatically discover current daemon working directory
	if len(projects) == 0 {
		if cwd, err := os.Getwd(); err == nil && cwd != "" {
			projects = append(projects, NormalizePath(cwd))
		}
	}

	return projects, nil
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
	plan.RepoPath = NormalizePath(plan.RepoPath)

	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		query := `
		INSERT INTO plans (slug, title, repo_path, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(slug) DO UPDATE SET
			title = excluded.title,
			repo_path = excluded.repo_path,
			status = excluded.status,
			updated_at = excluded.updated_at
		`
		_, err := tx.ExecContext(ctx, query, plan.Slug, plan.Title, plan.RepoPath, plan.Status, plan.CreatedAt, plan.UpdatedAt)
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
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	task.RepoPath = NormalizePath(task.RepoPath)

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
	if task.Status == "" {
		task.Status = domain.StatusPending
		if len(task.Dependencies) == 0 {
			task.Status = domain.StatusReady
		}
	}

	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		query := `
		INSERT INTO tasks (
			id, plan_id, repo_path, title, description, dependencies,
			priority, complexity, risk_level, required_skills, required_tools,
			target_files, acceptance_criteria, verification_commands,
			harness, skill_command, worktree_path, status, claimed_by, claimed_at,
			orchestrator_id, worker_role, pid, orchestrator_pid, notes,
			review_reason, review_details, review_options, review_response, resolved_by,
			conflict_files, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			notes = excluded.notes,
			updated_at = excluded.updated_at
		`
		_, err := tx.ExecContext(ctx, query,
			task.ID, task.PlanID, task.RepoPath, task.Title, task.Description, string(depsJSON),
			priority, complexity, risk, string(skillsJSON), string(toolsJSON),
			string(filesJSON), string(criteriaJSON), string(verifJSON),
			task.Harness, task.SkillCommand, task.WorktreePath, string(task.Status),
			task.ClaimedBy, task.ClaimedAt,
			task.OrchestratorID, task.WorkerRole, task.PID, task.OrchestratorPID, task.Notes,
			task.ReviewReason, task.ReviewDetails, string(optionsJSON), task.ReviewResponse, task.ResolvedBy,
			string(conflictJSON), task.CreatedAt, task.UpdatedAt,
		)
		return err
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

// UpsertSpec saves a technical specification, incrementing version on update.
func (s *Store) UpsertSpec(ctx context.Context, spec domain.TechSpec) (*domain.TechSpec, error) {
	if spec.Slug == "" {
		return nil, fmt.Errorf("spec slug cannot be empty")
	}
	if spec.Title == "" {
		return nil, fmt.Errorf("spec title cannot be empty")
	}
	if spec.Tier < 1 || spec.Tier > 5 {
		spec.Tier = 1
	}
	if spec.StructuredPayload == nil {
		spec.StructuredPayload = make(map[string]any)
	}
	if spec.BoundPaths == nil {
		spec.BoundPaths = []string{}
	}

	payloadBytes, err := json.Marshal(spec.StructuredPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal structured_payload: %w", err)
	}
	boundBytes, err := json.Marshal(spec.BoundPaths)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bound_paths: %w", err)
	}

	now := time.Now().UTC()
	var result *domain.TechSpec

	err = s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		var existingVersion int
		query := `SELECT version FROM tech_specs WHERE slug = ?`
		err := tx.QueryRowContext(ctx, query, spec.Slug).Scan(&existingVersion)

		if errors.Is(err, sql.ErrNoRows) {
			spec.Version = 1
			spec.UpdatedAt = now
			insertSQL := `
			INSERT INTO tech_specs (slug, repo_path, tier, kind, title, diagram_type, diagram_code, structured_payload, notes, bound_paths, version, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
			_, err = tx.ExecContext(ctx, insertSQL,
				spec.Slug, spec.RepoPath, spec.Tier, spec.Kind, spec.Title,
				spec.DiagramType, spec.DiagramCode, string(payloadBytes), spec.Notes,
				string(boundBytes), spec.Version, spec.UpdatedAt)
			if err != nil {
				return err
			}
			result = &spec
			return nil
		} else if err != nil {
			return err
		}

		spec.Version = existingVersion + 1
		spec.UpdatedAt = now
		updateSQL := `
		UPDATE tech_specs 
		SET repo_path=?, tier=?, kind=?, title=?, diagram_type=?, diagram_code=?, structured_payload=?, notes=?, bound_paths=?, version=?, updated_at=?
		WHERE slug=?`
		_, err = tx.ExecContext(ctx, updateSQL,
			spec.RepoPath, spec.Tier, spec.Kind, spec.Title,
			spec.DiagramType, spec.DiagramCode, string(payloadBytes), spec.Notes,
			string(boundBytes), spec.Version, spec.UpdatedAt, spec.Slug)
		if err != nil {
			return err
		}
		result = &spec
		return nil
	})

	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetSpec fetches a spec by slug.
func (s *Store) GetSpec(ctx context.Context, slug string) (*domain.TechSpec, error) {
	var spec *domain.TechSpec
	err := s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		var err error
		spec, err = s.getSpecTx(ctx, tx, slug)
		return err
	})
	if err != nil {
		return nil, err
	}
	return spec, nil
}

func (s *Store) getSpecTx(ctx context.Context, tx *sql.Tx, slug string) (*domain.TechSpec, error) {
	query := `
	SELECT slug, repo_path, tier, kind, title, diagram_type, diagram_code, structured_payload, notes, bound_paths, version, updated_at
	FROM tech_specs WHERE slug = ?`

	var spec domain.TechSpec
	var payloadStr, boundStr string
	err := tx.QueryRowContext(ctx, query, slug).Scan(
		&spec.Slug, &spec.RepoPath, &spec.Tier, &spec.Kind, &spec.Title,
		&spec.DiagramType, &spec.DiagramCode, &payloadStr, &spec.Notes,
		&boundStr, &spec.Version, &spec.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(payloadStr), &spec.StructuredPayload)
	_ = json.Unmarshal([]byte(boundStr), &spec.BoundPaths)
	return &spec, nil
}

// ListSpecs lists specs filtered optionally by repoPath and/or tier.
func (s *Store) ListSpecs(ctx context.Context, repoPath string, tier int) ([]domain.TechSpec, error) {
	query := `
	SELECT slug, repo_path, tier, kind, title, diagram_type, diagram_code, structured_payload, notes, bound_paths, version, updated_at
	FROM tech_specs WHERE 1=1`
	var args []any

	if repoPath != "" {
		query += " AND repo_path = ?"
		args = append(args, repoPath)
	}
	if tier > 0 {
		query += " AND tier = ?"
		args = append(args, tier)
	}
	query += " ORDER BY tier ASC, slug ASC"

	var specs []domain.TechSpec
	err := s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var spec domain.TechSpec
			var payloadStr, boundStr string
			if err := rows.Scan(
				&spec.Slug, &spec.RepoPath, &spec.Tier, &spec.Kind, &spec.Title,
				&spec.DiagramType, &spec.DiagramCode, &payloadStr, &spec.Notes,
				&boundStr, &spec.Version, &spec.UpdatedAt); err != nil {
				return err
			}
			_ = json.Unmarshal([]byte(payloadStr), &spec.StructuredPayload)
			_ = json.Unmarshal([]byte(boundStr), &spec.BoundPaths)
			specs = append(specs, spec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return specs, nil
}

// SearchSpecs executes an FTS5 full-text query across tech_specs_fts.
func (s *Store) SearchSpecs(ctx context.Context, query string) ([]domain.TechSpec, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return s.ListSpecs(ctx, "", 0)
	}

	sqlQuery := `
	SELECT t.slug, t.repo_path, t.tier, t.kind, t.title, t.diagram_type, t.diagram_code, t.structured_payload, t.notes, t.bound_paths, t.version, t.updated_at
	FROM tech_specs_fts f
	JOIN tech_specs t ON f.rowid = t.rowid
	WHERE tech_specs_fts MATCH ?`

	var results []domain.TechSpec
	err := s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, sqlQuery, query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var spec domain.TechSpec
			var payloadStr, boundStr string
			if err := rows.Scan(
				&spec.Slug, &spec.RepoPath, &spec.Tier, &spec.Kind, &spec.Title,
				&spec.DiagramType, &spec.DiagramCode, &payloadStr, &spec.Notes,
				&boundStr, &spec.Version, &spec.UpdatedAt); err != nil {
				return err
			}
			_ = json.Unmarshal([]byte(payloadStr), &spec.StructuredPayload)
			_ = json.Unmarshal([]byte(boundStr), &spec.BoundPaths)
			results = append(results, spec)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// AutoLinkSpecs matches target files against bound paths of all registered specs.
func (s *Store) AutoLinkSpecs(ctx context.Context, targetFiles []string) ([]string, error) {
	var matched []string
	err := s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		var err error
		matched, err = s.autoLinkSpecsTx(ctx, tx, targetFiles)
		return err
	})
	return matched, err
}

func (s *Store) autoLinkSpecsTx(ctx context.Context, tx *sql.Tx, targetFiles []string) ([]string, error) {
	if len(targetFiles) == 0 {
		return nil, nil
	}

	rows, err := tx.QueryContext(ctx, `SELECT slug, bound_paths FROM tech_specs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matchedSlugsMap := make(map[string]bool)
	for rows.Next() {
		var slug, boundStr string
		if err := rows.Scan(&slug, &boundStr); err != nil {
			continue
		}
		var boundPaths []string
		_ = json.Unmarshal([]byte(boundStr), &boundPaths)

		for _, targetFile := range targetFiles {
			for _, bound := range boundPaths {
				if bound != "" && (strings.HasPrefix(targetFile, bound) || strings.Contains(targetFile, bound)) {
					matchedSlugsMap[slug] = true
					break
				}
			}
		}
	}

	var matched []string
	for slug := range matchedSlugsMap {
		matched = append(matched, slug)
	}
	sort.Strings(matched)
	return matched, nil
}

// SavePlanSnapshot saves an atomic directory snapshot for a plan at a given revision.
func (s *Store) SavePlanSnapshot(ctx context.Context, planSlug, revision, author, summary string, files []domain.SnapshotFile) error {
	if planSlug == "" {
		return fmt.Errorf("plan_slug cannot be empty")
	}
	if revision == "" {
		revision = "v1.0"
	}
	now := time.Now().UTC()

	return s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		// 1. Insert or update plan revision record
		_, err := tx.ExecContext(ctx, `
			INSERT INTO plan_revisions (plan_slug, revision, author, summary, created_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(plan_slug, revision) DO UPDATE SET
				author = excluded.author,
				summary = excluded.summary,
				created_at = excluded.created_at
		`, planSlug, revision, author, summary, now)
		if err != nil {
			return fmt.Errorf("failed to save plan revision: %w", err)
		}

		// 2. Clear previous snapshot files for this exact (plan_slug, revision) to support in-place re-push
		_, _ = tx.ExecContext(ctx, `DELETE FROM plan_snapshots WHERE plan_slug = ? AND revision = ?`, planSlug, revision)

		// 3. Insert each file
		insertFileStmt, err := tx.PrepareContext(ctx, `
			INSERT INTO plan_snapshots (plan_slug, revision, file_path, content, byte_size, sha256, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer insertFileStmt.Close()

		for _, f := range files {
			h := sha256.Sum256([]byte(f.Content))
			shaStr := hex.EncodeToString(h[:])
			byteSize := int64(len(f.Content))
			_, err = insertFileStmt.ExecContext(ctx, planSlug, revision, f.FilePath, f.Content, byteSize, shaStr, now)
			if err != nil {
				return fmt.Errorf("failed to save snapshot file %s: %w", f.FilePath, err)
			}
		}

		return nil
	})
}

// GetPlanSnapshot retrieves all snapshot files for a plan at a given revision.
func (s *Store) GetPlanSnapshot(ctx context.Context, planSlug, revision string) ([]domain.SnapshotFile, error) {
	var files []domain.SnapshotFile
	err := s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT plan_slug, revision, file_path, content, byte_size, sha256, created_at
			FROM plan_snapshots
			WHERE plan_slug = ? AND revision = ?
			ORDER BY file_path ASC
		`, planSlug, revision)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var f domain.SnapshotFile
			if err := rows.Scan(&f.PlanSlug, &f.Revision, &f.FilePath, &f.Content, &f.ByteSize, &f.SHA256, &f.CreatedAt); err != nil {
				return err
			}
			files = append(files, f)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// ListPlanRevisions lists all revision records for a given plan.
func (s *Store) ListPlanRevisions(ctx context.Context, planSlug string) ([]domain.PlanRevision, error) {
	var revs []domain.PlanRevision
	err := s.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, `
			SELECT id, plan_slug, revision, author, summary, created_at
			FROM plan_revisions
			WHERE plan_slug = ?
			ORDER BY id ASC
		`, planSlug)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var r domain.PlanRevision
			if err := rows.Scan(&r.ID, &r.PlanSlug, &r.Revision, &r.Author, &r.Summary, &r.CreatedAt); err != nil {
				return err
			}
			revs = append(revs, r)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return revs, nil
}

// GetPlanDiff computes differences between two plan revisions.
func (s *Store) GetPlanDiff(ctx context.Context, planSlug, fromRev, toRev string) ([]domain.FileDiff, error) {
	fromFiles, err := s.GetPlanSnapshot(ctx, planSlug, fromRev)
	if err != nil {
		return nil, err
	}
	toFiles, err := s.GetPlanSnapshot(ctx, planSlug, toRev)
	if err != nil {
		return nil, err
	}

	fromMap := make(map[string]string)
	for _, f := range fromFiles {
		fromMap[f.FilePath] = f.Content
	}

	toMap := make(map[string]string)
	for _, f := range toFiles {
		toMap[f.FilePath] = f.Content
	}

	var diffs []domain.FileDiff

	// Check modified and added files
	for path, newContent := range toMap {
		if oldContent, exists := fromMap[path]; exists {
			if oldContent == newContent {
				diffs = append(diffs, domain.FileDiff{
					FilePath:   path,
					ChangeType: "IDENTICAL",
					OldContent: oldContent,
					NewContent: newContent,
				})
			} else {
				diffs = append(diffs, domain.FileDiff{
					FilePath:   path,
					ChangeType: "MODIFIED",
					OldContent: oldContent,
					NewContent: newContent,
				})
			}
		} else {
			diffs = append(diffs, domain.FileDiff{
				FilePath:   path,
				ChangeType: "ADDED",
				OldContent: "",
				NewContent: newContent,
			})
		}
	}

	// Check deleted files
	for path, oldContent := range fromMap {
		if _, exists := toMap[path]; !exists {
			diffs = append(diffs, domain.FileDiff{
				FilePath:   path,
				ChangeType: "DELETED",
				OldContent: oldContent,
				NewContent: "",
			})
		}
	}

	sort.Slice(diffs, func(i, j int) bool {
		return diffs[i].FilePath < diffs[j].FilePath
	})

	return diffs, nil
}
