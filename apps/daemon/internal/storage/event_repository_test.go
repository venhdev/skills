package storage_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"zenfd/internal/domain"
	"zenfd/internal/storage"
)

func newEventStore(t *testing.T) (*storage.Store, context.Context) {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "events.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return storage.NewStore(db), ctx
}

func TestEvents_TriggersRecordEveryWriter(t *testing.T) {
	store, ctx := newEventStore(t)

	if err := store.SavePlanWithTasks(ctx, domain.Plan{Slug: "log-plan", Title: "Log", Status: "ACTIVE"}, []domain.Task{
		{ID: "A", PlanID: "log-plan", Title: "A", Status: domain.StatusReady},
		{ID: "B", PlanID: "log-plan", Title: "B", Status: domain.StatusPending, Dependencies: []string{"A"}},
	}); err != nil {
		t.Fatalf("SavePlanWithTasks failed: %v", err)
	}

	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusReady, domain.StatusRunning); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}
	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusRunning, domain.StatusCompleted); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	events, err := store.ListEvents(ctx, 0, 200)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected events")
	}

	var sawCreated, sawStatus, sawUnlock bool
	for _, evt := range events {
		switch evt.Type {
		case "task.created":
			if evt.TaskID == "A" && evt.PlanID == "log-plan" {
				sawCreated = true
			}
		case "task.updated":
			var payload struct {
				Kind string `json:"kind"`
				From string `json:"from"`
				To   string `json:"to"`
			}
			if err := json.Unmarshal([]byte(evt.Payload), &payload); err != nil {
				t.Fatalf("update payload is not JSON: %v", err)
			}
			if evt.TaskID == "A" && payload.Kind == "status" && payload.From == "RUNNING" && payload.To == "COMPLETED" {
				sawStatus = true
			}
			// Completing A unlocks B; that transition is an UPDATE too.
			if evt.TaskID == "B" && payload.To == "READY" {
				sawUnlock = true
			}
		}
	}
	if !sawCreated {
		t.Error("missing task.created event")
	}
	if !sawStatus {
		t.Error("missing status update event with from/to")
	}
	if !sawUnlock {
		t.Error("missing dependency-unlock update event")
	}
}

func TestEvents_ListWindowAndCursor(t *testing.T) {
	store, ctx := newEventStore(t)

	if err := store.SavePlanWithTasks(ctx, domain.Plan{Slug: "window-plan", Title: "Window", Status: "ACTIVE"}, []domain.Task{
		{ID: "A", PlanID: "window-plan", Title: "A", Status: domain.StatusReady},
		{ID: "B", PlanID: "window-plan", Title: "B", Status: domain.StatusReady},
		{ID: "C", PlanID: "window-plan", Title: "C", Status: domain.StatusReady},
	}); err != nil {
		t.Fatalf("SavePlanWithTasks failed: %v", err)
	}

	all, err := store.ListEvents(ctx, 0, 100)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	if len(all) < 4 {
		t.Fatalf("expected at least 4 events, got %d", len(all))
	}

	window, err := store.ListEvents(ctx, 0, 2)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	if len(window) != 2 {
		t.Fatalf("expected 2 events, got %d", len(window))
	}
	if window[0].ID >= window[1].ID {
		t.Errorf("window is not ascending: %d, %d", window[0].ID, window[1].ID)
	}
	if window[1].ID != all[len(all)-1].ID {
		t.Errorf("window should end at the newest event")
	}

	cursor, err := store.ListEvents(ctx, window[0].ID, 100)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	if len(cursor) == 0 || cursor[0].ID <= window[0].ID {
		t.Errorf("cursor did not advance: %+v", cursor)
	}
}
