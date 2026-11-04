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

	// Launch concurrent writers: each task creates a run and beats 5 times
	var wg sync.WaitGroup
	errCh := make(chan error, numTasks*2)

	for i := 0; i < numTasks; i++ {
		taskID := fmt.Sprintf("task-%02d", i)
		runID := fmt.Sprintf("run-%02d", i)

		wg.Add(1)
		go func(tID, rID string) {
			defer wg.Done()

			// Create Run
			run := domain.Run{
				ID:        rID,
				TaskID:    tID,
				PID:       1000,
				PGID:      1000,
				Status:    domain.StatusRunning,
				LogPath:   "/tmp/log",
				StartedAt: time.Now().UTC(),
			}
			if err := store.CreateRun(ctx, run); err != nil {
				errCh <- fmt.Errorf("CreateRun %s failed: %w", rID, err)
				return
			}

			// Claim Task: READY -> RUNNING
			if err := store.UpdateTaskStatus(ctx, tID, domain.StatusReady, domain.StatusRunning); err != nil {
				errCh <- fmt.Errorf("ClaimTask %s failed: %w", tID, err)
				return
			}

			// Heartbeat 3 times
			for b := 0; b < 3; b++ {
				if err := store.UpdateRunHeartbeat(ctx, rID, time.Now().UTC()); err != nil {
					errCh <- fmt.Errorf("Heartbeat %s failed: %w", rID, err)
					return
				}
				time.Sleep(2 * time.Millisecond)
			}

			// Complete Task: RUNNING -> COMPLETED
			if err := store.UpdateTaskStatus(ctx, tID, domain.StatusRunning, domain.StatusCompleted); err != nil {
				errCh <- fmt.Errorf("CompleteTask %s failed: %w", tID, err)
				return
			}
		}(taskID, runID)
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
	}
}
