package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"zenf/internal/domain"
)

// taskNode is a lightweight topology node used by the DAG transition policy.
type taskNode struct {
	id     string
	status domain.TaskStatus
	deps   []string
}

var activeDependencyStatuses = map[domain.TaskStatus]bool{
	domain.StatusPending: true,
	domain.StatusBlocked: true,
	domain.StatusReady:   true,
}

// containerClause returns the SQL predicate and args for one task container.
// A plan takes precedence; a project task is any task with a project but no
// plan; an orphan is a task with neither.
func containerClause(projectSlug, planSlug string) (string, []any) {
	switch {
	case planSlug != "":
		return "plan_slug = ?", []any{planSlug}
	case projectSlug != "":
		return "project_slug = ? AND plan_slug IS NULL", []any{projectSlug}
	default:
		return "project_slug IS NULL AND plan_slug IS NULL", nil
	}
}

func loadContainerNodes(ctx context.Context, tx *sql.Tx, projectSlug, planSlug string) (map[string]*taskNode, []string, error) {
	where, args := containerClause(projectSlug, planSlug)
	rows, err := tx.QueryContext(ctx, "SELECT id, status, dependencies FROM tasks WHERE "+where, args...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load container tasks: %w", err)
	}
	defer rows.Close()

	snapshot := make(map[string]*taskNode)
	order := []string{}
	for rows.Next() {
		var id, status, depsJSON string
		if err := rows.Scan(&id, &status, &depsJSON); err != nil {
			return nil, nil, fmt.Errorf("failed to scan container task: %w", err)
		}
		node := &taskNode{id: id, status: domain.TaskStatus(status)}
		if activeDependencyStatuses[node.status] {
			if err := json.Unmarshal([]byte(depsJSON), &node.deps); err != nil {
				return nil, nil, fmt.Errorf("%w: task %s has malformed dependencies: %v", ErrInvalidGraph, id, err)
			}
		}
		snapshot[id] = node
		order = append(order, id)
	}
	return snapshot, order, rows.Err()
}

func loadContainerDeps(ctx context.Context, tx *sql.Tx, projectSlug, planSlug string) (map[string][]string, error) {
	where, args := containerClause(projectSlug, planSlug)
	rows, err := tx.QueryContext(ctx, "SELECT id, dependencies FROM tasks WHERE "+where, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to load container dependencies: %w", err)
	}
	defer rows.Close()

	deps := make(map[string][]string)
	for rows.Next() {
		var id, depsJSON string
		if err := rows.Scan(&id, &depsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan container dependency: %w", err)
		}
		list := []string{}
		if depsJSON != "" {
			if err := json.Unmarshal([]byte(depsJSON), &list); err != nil {
				return nil, fmt.Errorf("%w: task %s has malformed dependencies: %v", ErrInvalidGraph, id, err)
			}
		}
		deps[id] = list
	}
	return deps, rows.Err()
}

func loadContainerStatuses(ctx context.Context, tx *sql.Tx, projectSlug, planSlug string) (map[string]domain.TaskStatus, error) {
	where, args := containerClause(projectSlug, planSlug)
	rows, err := tx.QueryContext(ctx, "SELECT id, status FROM tasks WHERE "+where, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to load container statuses: %w", err)
	}
	defer rows.Close()

	statuses := make(map[string]domain.TaskStatus)
	for rows.Next() {
		var id, status string
		if err := rows.Scan(&id, &status); err != nil {
			return nil, fmt.Errorf("failed to scan container status: %w", err)
		}
		statuses[id] = domain.TaskStatus(status)
	}
	return statuses, rows.Err()
}

func validateContainerGraphTx(ctx context.Context, tx *sql.Tx, projectSlug, planSlug string) error {
	deps, err := loadContainerDeps(ctx, tx, projectSlug, planSlug)
	if err != nil {
		return err
	}
	return validateGraph(deps, false)
}

func validateCompleteGraphTx(ctx context.Context, tx *sql.Tx, projectSlug, planSlug string) error {
	deps, err := loadContainerDeps(ctx, tx, projectSlug, planSlug)
	if err != nil {
		return err
	}
	return validateGraph(deps, true)
}

func resyncContainerStatuses(ctx context.Context, tx *sql.Tx, projectSlug, planSlug string, tasks []domain.Task, now time.Time) error {
	statuses, err := loadContainerStatuses(ctx, tx, projectSlug, planSlug)
	if err != nil {
		return err
	}
	for _, task := range tasks {
		current := statuses[task.ID]
		if current != "" && current != domain.StatusReady && current != domain.StatusPending {
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

func unlockDependents(ctx context.Context, tx *sql.Tx, snapshot map[string]*taskNode, order []string, now time.Time) error {
	for _, id := range order {
		node := snapshot[id]
		if node.status != domain.StatusPending && node.status != domain.StatusBlocked {
			continue
		}
		if !allDependenciesCompleted(snapshot, node.deps) {
			continue
		}
		if err := setTaskStatus(ctx, tx, id, domain.StatusReady, now); err != nil {
			return err
		}
		node.status = domain.StatusReady
	}
	return nil
}

func allDependenciesCompleted(snapshot map[string]*taskNode, deps []string) bool {
	for _, depID := range deps {
		dep, ok := snapshot[depID]
		if !ok || dep.status != domain.StatusCompleted {
			return false
		}
	}
	return true
}

func blockDependentsOf(ctx context.Context, tx *sql.Tx, snapshot map[string]*taskNode, order []string, failedID string, now time.Time) error {
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

			node := snapshot[id]
			if node == nil {
				continue
			}
			switch node.status {
			case domain.StatusPending, domain.StatusReady:
				if err := setTaskStatus(ctx, tx, id, domain.StatusBlocked, now); err != nil {
					return err
				}
				node.status = domain.StatusBlocked
				queue = append(queue, id)
			case domain.StatusBlocked:
				queue = append(queue, id)
			}
		}
	}
	return nil
}

func deriveStatus(statuses map[string]domain.TaskStatus, deps []string) domain.TaskStatus {
	if len(deps) == 0 {
		return domain.StatusReady
	}

	allDone := true
	blocked := false
	for _, dep := range deps {
		status, ok := statuses[dep]
		if !ok {
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
				continue
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
