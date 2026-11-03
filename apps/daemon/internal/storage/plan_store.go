package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
review_reason, review_details, review_options, review_response,
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
		&t.ReviewReason, &t.ReviewDetails, &optionsJSON, &t.ReviewResponse,
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
			review_reason, review_details, review_options, review_response,
			conflict_files, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			review_reason = excluded.review_reason,
			review_details = excluded.review_details,
			review_options = excluded.review_options,
			review_response = excluded.review_response,
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
				task.ID, plan.Slug, task.RepoPath, task.Title, task.Description, string(depsJSON),
				priority, complexity, risk, string(skillsJSON), string(toolsJSON),
				string(filesJSON), string(criteriaJSON), string(verifJSON),
				task.Harness, task.SkillCommand, task.WorktreePath, string(task.Status),
				task.ClaimedBy, task.ClaimedAt,
				task.ReviewReason, task.ReviewDetails, string(optionsJSON), task.ReviewResponse,
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

		// Update to RUNNING with claim lock
		res, err := tx.ExecContext(ctx, `
			UPDATE tasks
			SET status = 'RUNNING', claimed_by = ?, claimed_at = ?, updated_at = ?
			WHERE id = ? AND status = 'READY'
		`, workerID, now, now, taskID)
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

// ResolveReview answers an awaiting review and resumes task back to RUNNING.
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
			SET status = 'RUNNING', review_response = ?, updated_at = ?
			WHERE id = ? AND status = 'AWAITING_REVIEW'
		`, response, now, taskID)
		if err != nil {
			return err
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			return fmt.Errorf("%w: failed to transition task %s to RUNNING", ErrConflictStatus, taskID)
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

	sb.WriteString("## 5. Execution Protocol\n")
	sb.WriteString("- If scope or architectural drift is encountered, invoke `zenfd_request_review` with your questions and options.\n")
	sb.WriteString("- Once all acceptance criteria are met and verification commands pass, invoke `zenfd_complete_task`.\n")

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
