package storage_test

import (
	"errors"
	"testing"

	"zenfd/internal/domain"
	"zenfd/internal/storage"
)

func TestAgentStore_CreateDefaultsAndValidation(t *testing.T) {
	store, ctx := newTestStore(t)

	agent := domain.Agent{Slug: "coder", Name: "Coder", Enabled: true}
	if err := store.CreateAgent(ctx, agent); err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}
	created, err := store.GetAgent(ctx, "coder")
	if err != nil {
		t.Fatalf("GetAgent failed: %v", err)
	}
	if created.Persona != domain.PersonaCustom || created.Mode != domain.AgentModeHeadless {
		t.Fatalf("unexpected defaults: %+v", created)
	}

	bad := []domain.Agent{
		{Name: "No slug", Enabled: true},
		{Slug: "no-name", Enabled: true},
		{Slug: "bad-persona", Name: "Bad", Persona: "NOPE", Enabled: true},
		{Slug: "bad-mode", Name: "Bad", Mode: "NOPE", Enabled: true},
	}
	for _, candidate := range bad {
		if err := store.CreateAgent(ctx, candidate); err == nil {
			t.Fatalf("expected validation error for %+v", candidate)
		}
	}

	if err := store.CreateAgent(ctx, domain.Agent{Slug: "coder", Name: "Duplicate", Enabled: true}); !errors.Is(err, storage.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestAgentStore_UpdateListDelete(t *testing.T) {
	store, ctx := newTestStore(t)

	explorer := domain.Agent{Slug: "explorer", Name: "Explorer", Persona: domain.PersonaExplorer, Enabled: true}
	reviewer := domain.Agent{Slug: "reviewer", Name: "Reviewer", Persona: domain.PersonaReviewer, Enabled: false}
	for _, agent := range []domain.Agent{explorer, reviewer} {
		if err := store.CreateAgent(ctx, agent); err != nil {
			t.Fatalf("CreateAgent(%s) failed: %v", agent.Slug, err)
		}
	}

	enabled, err := store.ListAgents(ctx, false)
	if err != nil || len(enabled) != 1 || enabled[0].Slug != "explorer" {
		t.Fatalf("unexpected enabled list: %+v, err=%v", enabled, err)
	}
	all, err := store.ListAgents(ctx, true)
	if err != nil || len(all) != 2 {
		t.Fatalf("unexpected all list: %+v, err=%v", all, err)
	}

	if err := store.UpdateAgent(ctx, "explorer", domain.AgentPatch{
		Name:            ptr("Explorer Updated"),
		Persona:         ptr(domain.PersonaCoder),
		Mode:            ptr(domain.AgentModeInteractive),
		Goal:            ptr("map the codebase"),
		StartConditions: &[]string{"task ready"},
		StopConditions:  &[]string{"task complete"},
		Schedule:        ptr("0 * * * *"),
		Skills:          &[]string{"recon"},
		Model:           ptr("gpt-5"),
		Runner:          ptr("codex"),
		Enabled:         ptr(true),
	}); err != nil {
		t.Fatalf("UpdateAgent failed: %v", err)
	}
	updated, _ := store.GetAgent(ctx, "explorer")
	if updated.Name != "Explorer Updated" || updated.Persona != domain.PersonaCoder ||
		updated.Mode != domain.AgentModeInteractive || updated.Goal != "map the codebase" {
		t.Fatalf("agent patch not applied: %+v", updated)
	}
	if len(updated.StartConditions) != 1 || len(updated.StopConditions) != 1 ||
		len(updated.Skills) != 1 || updated.Schedule != "0 * * * *" {
		t.Fatalf("agent list fields not applied: %+v", updated)
	}

	if err := store.UpdateAgent(ctx, "reviewer", domain.AgentPatch{Enabled: ptr(true)}); err != nil {
		t.Fatalf("enable reviewer failed: %v", err)
	}
	if enabled, _ := store.ListAgents(ctx, false); len(enabled) != 2 {
		t.Fatalf("expected reviewer enabled, got %+v", enabled)
	}

	if err := store.DeleteAgent(ctx, "reviewer"); err != nil {
		t.Fatalf("DeleteAgent failed: %v", err)
	}
	if _, err := store.GetAgent(ctx, "reviewer"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected agent deleted, got %v", err)
	}
	if err := store.DeleteAgent(ctx, "reviewer"); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound on second delete, got %v", err)
	}
}

func TestSessionStore_RegisterListHeartbeatClose(t *testing.T) {
	store, ctx := newTestStore(t)

	if err := store.CreateAgent(ctx, domain.Agent{Slug: "coder", Name: "Coder", Enabled: true}); err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}

	if err := store.CreateSession(ctx, domain.AgentSession{
		ID: "main-1", SessionType: domain.SessionMain, Status: domain.SessionActive,
	}); err != nil {
		t.Fatalf("create MAIN session failed: %v", err)
	}
	if err := store.CreateSession(ctx, domain.AgentSession{
		ID: "sub-1", SessionType: domain.SessionSubagent, ParentID: "main-1", Status: domain.SessionIdle,
	}); err != nil {
		t.Fatalf("create SUBAGENT session failed: %v", err)
	}
	if err := store.CreateSession(ctx, domain.AgentSession{
		ID: "worker-1", AgentSlug: "coder", SessionType: domain.SessionWorker,
		ParentID: "main-1", ExternalRef: "pid-42", Status: domain.SessionActive,
	}); err != nil {
		t.Fatalf("create WORKER session failed: %v", err)
	}

	if err := store.CreateSession(ctx, domain.AgentSession{
		ID: "worker-bad", SessionType: domain.SessionWorker, Status: domain.SessionActive,
	}); err == nil {
		t.Fatal("expected worker session without agent_slug to be rejected")
	}

	active, err := store.ListSessions(ctx, domain.SessionFilter{ActiveOnly: true})
	if err != nil || len(active) != 3 {
		t.Fatalf("unexpected active sessions: %+v, err=%v", active, err)
	}
	workers, err := store.ListSessions(ctx, domain.SessionFilter{Type: domain.SessionWorker})
	if err != nil || len(workers) != 1 || workers[0].ID != "worker-1" {
		t.Fatalf("unexpected worker sessions: %+v, err=%v", workers, err)
	}
	byAgent, err := store.ListSessions(ctx, domain.SessionFilter{AgentSlug: "coder"})
	if err != nil || len(byAgent) != 1 {
		t.Fatalf("unexpected agent sessions: %+v, err=%v", byAgent, err)
	}

	if err := store.HeartbeatSession(ctx, "worker-1", domain.SessionIdle); err != nil {
		t.Fatalf("HeartbeatSession failed: %v", err)
	}
	worker, _ := store.GetSession(ctx, "worker-1")
	if worker.Status != domain.SessionIdle {
		t.Fatalf("status = %s, want IDLE", worker.Status)
	}

	if err := store.CloseSession(ctx, "worker-1"); err != nil {
		t.Fatalf("CloseSession failed: %v", err)
	}
	worker, _ = store.GetSession(ctx, "worker-1")
	if worker.Status != domain.SessionTerminated || worker.EndedAt == nil {
		t.Fatalf("session not terminated: %+v", worker)
	}

	if err := store.HeartbeatSession(ctx, "missing", ""); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing session, got %v", err)
	}
}

func TestSessionStore_AgentDeleteSetsSlugNull(t *testing.T) {
	store, ctx := newTestStore(t)
	if err := store.CreateAgent(ctx, domain.Agent{Slug: "coder", Name: "Coder", Enabled: true}); err != nil {
		t.Fatalf("CreateAgent failed: %v", err)
	}
	if err := store.CreateSession(ctx, domain.AgentSession{
		ID: "worker-1", AgentSlug: "coder", SessionType: domain.SessionWorker, Status: domain.SessionActive,
	}); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if err := store.DeleteAgent(ctx, "coder"); err != nil {
		t.Fatalf("DeleteAgent failed: %v", err)
	}
	worker, err := store.GetSession(ctx, "worker-1")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}
	if worker.AgentSlug != "" {
		t.Fatalf("expected agent_slug to be cleared, got %q", worker.AgentSlug)
	}
}
