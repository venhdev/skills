package storage_test

import (
	"encoding/json"
	"testing"

	"zenf/internal/domain"
)

func eventTypes(events []domain.EventMessage) map[string]int {
	counts := make(map[string]int, len(events))
	for _, event := range events {
		counts[event.Type]++
	}
	return counts
}

func TestEventRepository_TriggersRecordEveryMutation(t *testing.T) {
	store, ctx := newTestStore(t)

	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{
		{ID: "A", Title: "A", Status: domain.StatusReady},
		{ID: "B", Title: "B", Dependencies: []string{"A"}},
	})
	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusReady, domain.StatusRunning); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}
	if err := store.CreateDocument(ctx, newDocument("alpha", "ARCH")); err != nil {
		t.Fatalf("CreateDocument failed: %v", err)
	}
	if err := store.CreateAgent(ctx, domain.Agent{Slug: "coder", Name: "Coder", Enabled: true}); err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}
	if err := store.CreateSession(ctx, domain.AgentSession{
		ID: "main-1", SessionType: domain.SessionMain, Status: domain.SessionActive,
	}); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if err := store.RecordArtifact(ctx, domain.Artifact{
		ID: "out-1", TaskID: "A", Type: domain.ArtifactTypeFile, FilePath: "/tmp/out",
	}); err != nil {
		t.Fatalf("RecordArtifact failed: %v", err)
	}

	events, err := store.ListEvents(ctx, 0, 500)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	counts := eventTypes(events)
	for _, want := range []string{
		"project.created", "plan.created", "task.created", "task.updated",
		"document.created", "document.tagged", "agent.created",
		"session.registered", "artifact.added",
	} {
		if counts[want] == 0 {
			t.Errorf("expected at least one %s event, got counts %+v", want, counts)
		}
	}

	// Events must be ordered ascending by id.
	for i := 1; i < len(events); i++ {
		if events[i].ID <= events[i-1].ID {
			t.Fatalf("events not ascending at index %d: %d <= %d", i, events[i].ID, events[i-1].ID)
		}
	}
}

func TestEventRepository_CarriesEntityContextAndPayload(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{{ID: "A", Title: "A", Status: domain.StatusReady}})
	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusReady, domain.StatusRunning); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	events, err := store.ListEvents(ctx, 0, 100)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}

	var statusEvent *domain.EventMessage
	for i := range events {
		event := &events[i]
		if event.Type == "task.updated" {
			var payload map[string]any
			if err := json.Unmarshal([]byte(event.Payload), &payload); err != nil {
				t.Fatalf("event payload is not JSON: %v", err)
			}
			if payload["kind"] == "status" {
				statusEvent = event
				break
			}
		}
	}
	if statusEvent == nil {
		t.Fatal("expected a task.updated status event")
	}
	if statusEvent.ProjectSlug != "alpha" || statusEvent.PlanSlug != "p1" ||
		statusEvent.TaskID != "A" || statusEvent.EntityType != "task" || statusEvent.EntityID != "A" {
		t.Fatalf("event context mismatch: %+v", statusEvent)
	}

	var payload map[string]any
	_ = json.Unmarshal([]byte(statusEvent.Payload), &payload)
	if payload["from"] != "READY" || payload["to"] != "RUNNING" {
		t.Fatalf("unexpected status payload: %+v", payload)
	}
}

func TestEventRepository_WindowAndLatestID(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")

	first, err := store.LatestEventID(ctx)
	if err != nil {
		t.Fatalf("LatestEventID failed: %v", err)
	}
	if first == 0 {
		t.Fatal("expected project.created event")
	}

	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{{ID: "A", Title: "A"}})
	latest, err := store.LatestEventID(ctx)
	if err != nil {
		t.Fatalf("LatestEventID failed: %v", err)
	}
	if latest <= first {
		t.Fatalf("expected latest id to advance, got %d <= %d", latest, first)
	}

	window, err := store.ListEvents(ctx, first, 100)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}
	if len(window) == 0 {
		t.Fatal("expected events after the first project event")
	}
	for _, event := range window {
		if event.ID <= first {
			t.Fatalf("window leaked event %d <= afterID %d", event.ID, first)
		}
		if event.Type == "project.created" {
			t.Fatalf("window should exclude the first project event: %+v", event)
		}
	}

	clamped, err := store.ListEvents(ctx, 0, 1)
	if err != nil {
		t.Fatalf("ListEvents(limit=1) failed: %v", err)
	}
	if len(clamped) != 1 {
		t.Fatalf("expected one event, got %d", len(clamped))
	}
}
