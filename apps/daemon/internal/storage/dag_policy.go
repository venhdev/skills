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

// planTask is a lightweight topology node used by the DAG transition policy.
type planTask struct {
	id     string
	status domain.TaskStatus
	deps   []string
}

// activeDependencyStatuses are the statuses whose dependency list still drives
// the DAG. Terminal and running rows are skipped so a corrupt value on an
// unrelated task cannot fail an in-flight transition.
var activeDependencyStatuses = map[domain.TaskStatus]bool{
	domain.StatusPending: true,
	domain.StatusBlocked: true,
	domain.StatusReady:   true,
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
		if activeDependencyStatuses[pt.status] {
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

// blockDependentsOf blocks every PENDING or READY task that transitively depends
// on the failed task. Direct dependents that were already blocked stay blocked
// and keep propagating the failure to their own dependents, so a failure can
// never leave a downstream task looking claimable.
func blockDependentsOf(ctx context.Context, tx *sql.Tx, snapshot map[string]*planTask, order []string, failedID string, now time.Time) error {
	dependents := make(map[string][]string, len(snapshot))
	for _, id := range order {
		for _, dep := range snapshot[id].deps {
			dependents[dep] = append(dependents[dep], id)
		}
	}

	queue := []string{failedID}
	visited := map[string]bool{failedID: true}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, id := range dependents[current] {
			if visited[id] {
				continue
			}
			visited[id] = true

			pt := snapshot[id]
			if pt == nil {
				continue
			}

			switch pt.status {
			case domain.StatusPending, domain.StatusReady:
				if err := setTaskStatus(ctx, tx, id, domain.StatusBlocked, now); err != nil {
					return err
				}
				pt.status = domain.StatusBlocked
				queue = append(queue, id)
			case domain.StatusBlocked:
				// Already blocked by an earlier failure: keep walking downstream
				// so grandchildren are not left PENDING forever.
				queue = append(queue, id)
			}
		}
	}
	return nil
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

// taskPlanAndDeps returns the owning plan and the raw dependency payload of a
// task, and reports ErrNotFound when the task does not exist.
func taskPlanAndDeps(ctx context.Context, tx *sql.Tx, taskID string) (string, string, error) {
	var planID, depsJSON string
	err := tx.QueryRowContext(ctx, "SELECT plan_id, dependencies FROM tasks WHERE id = ?", taskID).Scan(&planID, &depsJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", fmt.Errorf("%w: task %s", ErrNotFound, taskID)
	}
	if err != nil {
		return "", "", fmt.Errorf("failed to load task %s: %w", taskID, err)
	}
	return planID, depsJSON, nil
}

// loadPlanStatuses reads only the status column of a plan, so callers that need
// topology decisions do not depend on the dependency payload of every row.
func loadPlanStatuses(ctx context.Context, tx *sql.Tx, planID string) (map[string]domain.TaskStatus, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id, status FROM tasks WHERE plan_id = ?", planID)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan statuses: %w", err)
	}
	defer rows.Close()

	statuses := make(map[string]domain.TaskStatus)
	for rows.Next() {
		var id, status string
		if err := rows.Scan(&id, &status); err != nil {
			return nil, fmt.Errorf("failed to scan plan status: %w", err)
		}
		statuses[id] = domain.TaskStatus(status)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read plan statuses: %w", err)
	}
	return statuses, nil
}

// deriveStatus computes the status a task should have given its prerequisites.
// It is used when a task is created or its edges change, so a task added to an
// in-progress plan lands READY/BLOCKED immediately instead of waiting for a
// future completion that will never happen.
func deriveStatus(statuses map[string]domain.TaskStatus, deps []string) domain.TaskStatus {
	if len(deps) == 0 {
		return domain.StatusReady
	}

	allDone := true
	blocked := false
	for _, dep := range deps {
		status, ok := statuses[dep]
		if !ok {
			// Forward reference to a task that does not exist yet.
			allDone = false
			continue
		}
		switch status {
		case domain.StatusCompleted:
		case domain.StatusFailed, domain.StatusCancelled, domain.StatusBlocked:
			blocked = true
			allDone = false
		default:
			allDone = false
		}
	}

	if allDone {
		return domain.StatusReady
	}
	if blocked {
		return domain.StatusBlocked
	}
	return domain.StatusPending
}

func allStatusesCompleted(statuses map[string]domain.TaskStatus, deps []string) bool {
	for _, dep := range deps {
		if statuses[dep] != domain.StatusCompleted {
			return false
		}
	}
	return true
}

// loadPlanDeps reads the dependency map of a plan. Unlike loadPlanTasks it
// decodes every row: graph validation cannot reason about a corrupt edge.
func loadPlanDeps(ctx context.Context, tx *sql.Tx, planID string) (map[string][]string, error) {
	rows, err := tx.QueryContext(ctx, "SELECT id, dependencies FROM tasks WHERE plan_id = ?", planID)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan dependencies: %w", err)
	}
	defer rows.Close()

	deps := make(map[string][]string)
	for rows.Next() {
		var id, depsJSON string
		if err := rows.Scan(&id, &depsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan plan dependencies: %w", err)
		}
		list := []string{}
		if depsJSON != "" {
			if err := json.Unmarshal([]byte(depsJSON), &list); err != nil {
				return nil, fmt.Errorf("%w: task %s has malformed dependencies: %v", ErrInvalidGraph, id, err)
			}
		}
		deps[id] = list
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to read plan dependencies: %w", err)
	}
	return deps, nil
}

// validateTaskGraphTx rejects self-dependencies and cycles in a plan. Missing
// dependencies are allowed here because tasks are authored incrementally (a
// task may be created before the prerequisite it names); a full plan ingestion
// additionally calls validateCompleteGraph.
func validateTaskGraphTx(ctx context.Context, tx *sql.Tx, planID string) error {
	deps, err := loadPlanDeps(ctx, tx, planID)
	if err != nil {
		return err
	}
	return validateGraph(deps, false)
}

// validateCompleteGraph checks a fully specified plan: every dependency must
// exist in the same plan, no task may depend on itself, and the graph must be
// acyclic.
func validateCompleteGraph(deps map[string][]string) error {
	return validateGraph(deps, true)
}

func validateGraph(deps map[string][]string, requireExisting bool) error {
	for id, list := range deps {
		for _, dep := range list {
			if dep == id {
				return fmt.Errorf("%w: task %s depends on itself", ErrInvalidGraph, id)
			}
			if requireExisting {
				if _, ok := deps[dep]; !ok {
					return fmt.Errorf("%w: task %s depends on unknown task %s", ErrInvalidGraph, id, dep)
				}
			}
		}
	}
	if cycle := findCycle(deps); cycle != "" {
		return fmt.Errorf("%w: dependency cycle through task %s", ErrInvalidGraph, cycle)
	}
	return nil
}

// findCycle returns one task id that participates in a dependency cycle, or ""
// when the graph is acyclic. It uses an explicit stack so deep plans cannot
// overflow the goroutine stack.
func findCycle(deps map[string][]string) string {
	const (
		unvisited = 0
		visiting  = 1
		done      = 2
	)
	state := make(map[string]int, len(deps))

	type frame struct {
		id   string
		next int
	}

	for root := range deps {
		if state[root] != unvisited {
			continue
		}

		stack := []frame{{id: root}}
		state[root] = visiting

		for len(stack) > 0 {
			top := &stack[len(stack)-1]
			edges := deps[top.id]

			if top.next >= len(edges) {
				state[top.id] = done
				stack = stack[:len(stack)-1]
				continue
			}

			dep := edges[top.next]
			top.next++

			if _, ok := deps[dep]; !ok {
				continue // forward reference; not a cycle yet
			}

			switch state[dep] {
			case visiting:
				return dep
			case unvisited:
				state[dep] = visiting
				stack = append(stack, frame{id: dep})
			}
		}
	}
	return ""
}
