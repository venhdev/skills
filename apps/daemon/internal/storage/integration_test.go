package storage_test

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"zenfd/internal/domain"
	"zenfd/internal/storage"
)

func TestStorage_HighConcurrencyWrites(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "concurrent.db")

	db, err := storage.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	store := storage.NewStore(db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Seed root plan
	plan := domain.Plan{
		Slug:   "stress-plan",
		Title:  "Stress Test Plan",
		Status: "ACTIVE",
	}

	const numTasks = 20
	var tasks []domain.Task
	for i := 0; i < numTasks; i++ {
		tasks = append(tasks, domain.Task{
			ID:     fmt.Sprintf("task-%02d", i),
			PlanID: plan.Slug,
			Title:  fmt.Sprintf("Task %02d", i),
			Status: domain.StatusReady,
		})
	}

	if err := store.SavePlanWithTasks(ctx, plan, tasks); err != nil {
		t.Fatalf("SavePlanWithTasks failed: %v", err)
	}

	// Launch concurrent writers: each task is claimed, annotated, audited, and
	// completed through the same serialized write path.
	var wg sync.WaitGroup
	errCh := make(chan error, numTasks*3)

	for i := 0; i < numTasks; i++ {
		taskID := fmt.Sprintf("task-%02d", i)

		wg.Add(1)
		go func(tID string) {
			defer wg.Done()

			// Claim Task: READY -> RUNNING
			if err := store.UpdateTaskStatus(ctx, tID, domain.StatusReady, domain.StatusRunning); err != nil {
				errCh <- fmt.Errorf("ClaimTask %s failed: %w", tID, err)
				return
			}

			if err := store.AppendTaskNotes(ctx, tID, "worker started"); err != nil {
				errCh <- fmt.Errorf("AppendTaskNotes %s failed: %w", tID, err)
				return
			}

			if err := store.AppendEvent(ctx, domain.EventMessage{
				TaskID:  tID,
				Type:    "task:status",
				Payload: `{"status":"RUNNING"}`,
			}); err != nil {
				errCh <- fmt.Errorf("AppendEvent %s failed: %w", tID, err)
				return
			}

			// Complete Task: RUNNING -> COMPLETED
			if err := store.UpdateTaskStatus(ctx, tID, domain.StatusRunning, domain.StatusCompleted); err != nil {
				errCh <- fmt.Errorf("CompleteTask %s failed: %w", tID, err)
				return
			}
		}(taskID)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("Concurrent write error: %v", err)
	}

	// Verify all tasks are completed
	allTasks, err := store.GetTasksByPlan(ctx, plan.Slug)
	if err != nil {
		t.Fatalf("GetTasksByPlan failed: %v", err)
	}
	if len(allTasks) != numTasks {
		t.Fatalf("Expected %d tasks, got %d", numTasks, len(allTasks))
	}

	for _, task := range allTasks {
		if task.Status != domain.StatusCompleted {
			t.Errorf("Task %s is not COMPLETED, got: %s", task.ID, task.Status)
		}
		if task.Notes != "worker started" {
			t.Errorf("Task %s notes = %q, want %q", task.ID, task.Notes, "worker started")
		}
	}
}
