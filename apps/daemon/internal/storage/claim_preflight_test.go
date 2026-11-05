package storage

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"zenfd/internal/contextpacket"
	"zenfd/internal/domain"
	"zenfd/internal/domain/prompt"
)

// blockingCompiler lets a test hold prompt compilation open so it can prove
// that claim preflight does not hold the database write lock.
type blockingCompiler struct {
	started     chan struct{}
	release     chan struct{}
	startOnce   sync.Once
	releaseOnce sync.Once
	err         error
}

var _ prompt.Compiler = (*blockingCompiler)(nil)

func (c *blockingCompiler) Compile(ctx context.Context, input prompt.Input) (*domain.ContextPacket, error) {
	c.startOnce.Do(func() { close(c.started) })
	select {
	case <-c.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if c.err != nil {
		return nil, c.err
	}
	return &domain.ContextPacket{
		Task:             input.Task,
		InputAttachments: input.InputAttachments,
	}, nil
}

func (c *blockingCompiler) Release() {
	c.releaseOnce.Do(func() { close(c.release) })
}

func seedClaimTasks(t *testing.T, store *Store, ids ...string) {
	t.Helper()
	ctx := context.Background()
	if err := store.CreateProject(ctx, domain.Project{
		Slug: "claim-project", Name: "Claim Project", RootPath: t.TempDir(),
	}); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	tasks := make([]domain.Task, len(ids))
	for i, id := range ids {
		tasks[i] = domain.Task{
			ID:           id,
			PlanSlug:     "claim-preflight",
			Title:        id,
			Status:       domain.StatusReady,
			Dependencies: []string{},
		}
	}
	if err := store.CreatePlanWithTasks(ctx, domain.Plan{
		Slug:        "claim-preflight",
		ProjectSlug: "claim-project",
		Title:       "Claim Preflight",
		Status:      domain.PlanStatusActive,
	}, tasks); err != nil {
		t.Fatalf("CreatePlanWithTasks failed: %v", err)
	}
}

func newClaimTestStore(t *testing.T, compiler prompt.Compiler) (*Store, context.Context) {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "claim-preflight.db"))
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store := NewStore(db)
	if compiler != nil {
		store.TaskLifecycle.packets = contextpacket.New(store.ArtifactRepository, compiler)
	}
	return store, context.Background()
}

func TestClaimTask_BuildsPacketOutsideWriteLock(t *testing.T) {
	compiler := &blockingCompiler{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	store, ctx := newClaimTestStore(t, compiler)
	seedClaimTasks(t, store, "T-CLAIM-A", "T-WRITE-B")
	defer compiler.Release()

	claimErr := make(chan error, 1)
	go func() {
		_, err := store.ClaimTask(ctx, "T-CLAIM-A", domain.ClaimOptions{WorkerID: "worker-a", WorkerRole: "CODER"})
		claimErr <- err
	}()

	select {
	case <-compiler.started:
	case <-time.After(2 * time.Second):
		t.Fatal("claim did not reach prompt compilation")
	}

	// While compilation is blocked, an unrelated write must still proceed.
	// The old implementation held writeMu here and this write would stall.
	writeCtx, cancel := context.WithTimeout(ctx, 750*time.Millisecond)
	defer cancel()
	writeErr := make(chan error, 1)
	go func() {
		writeErr <- store.UpdateTaskNotes(writeCtx, "T-WRITE-B", "preflight write")
	}()

	select {
	case err := <-writeErr:
		if err != nil {
			t.Fatalf("unrelated write failed: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("unrelated write blocked behind claim preflight")
	}

	compiler.Release()
	if err := <-claimErr; err != nil {
		t.Fatalf("ClaimTask failed: %v", err)
	}

	task, err := store.GetTask(ctx, "T-CLAIM-A")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.Status != domain.StatusRunning {
		t.Fatalf("expected claimed task RUNNING, got %s", task.Status)
	}
}

func TestClaimTask_PacketFailureLeavesTaskReady(t *testing.T) {
	compiler := &blockingCompiler{
		started: make(chan struct{}),
		release: make(chan struct{}),
		err:     errors.New("prompt compilation failed"),
	}
	compiler.Release()
	store, ctx := newClaimTestStore(t, compiler)
	seedClaimTasks(t, store, "T-FAIL")

	if _, err := store.ClaimTask(ctx, "T-FAIL", domain.ClaimOptions{WorkerID: "worker-fail", WorkerRole: "CODER"}); err == nil {
		t.Fatal("expected claim to fail when packet compilation fails")
	}

	task, err := store.GetTask(ctx, "T-FAIL")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.Status != domain.StatusReady {
		t.Fatalf("expected task to remain READY, got %s", task.Status)
	}
}

func TestClaimTask_ConcurrentSameTask(t *testing.T) {
	store, ctx := newClaimTestStore(t, nil)
	seedClaimTasks(t, store, "T-RACE")

	const workers = 8
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := store.ClaimTask(ctx, "T-RACE", domain.ClaimOptions{WorkerID: "worker-race", WorkerRole: "CODER"})
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)

	successes, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrConflictStatus):
			conflicts++
		default:
			t.Errorf("unexpected claim error: %v", err)
		}
	}
	if successes != 1 || conflicts != workers-1 {
		t.Fatalf("expected 1 success and %d conflicts, got %d/%d", workers-1, successes, conflicts)
	}

	task, err := store.GetTask(ctx, "T-RACE")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.Status != domain.StatusRunning {
		t.Fatalf("expected task RUNNING, got %s", task.Status)
	}
}

func TestClaimTask_RecordsClaimMetadata(t *testing.T) {
	store, ctx := newClaimTestStore(t, nil)
	seedClaimTasks(t, store, "T-META")

	packet, err := store.ClaimTask(ctx, "T-META", domain.ClaimOptions{
		WorkerID:       "worker-meta",
		OrchestratorID: "main-session",
		WorkerRole:     "CODER",
		SessionID:      "sess-meta",
	})
	if err != nil {
		t.Fatalf("ClaimTask failed: %v", err)
	}
	if packet.Task.Status != domain.StatusRunning {
		t.Fatalf("packet task status = %s, want RUNNING", packet.Task.Status)
	}

	task, err := store.GetTask(ctx, "T-META")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.ClaimedBy != "worker-meta" || task.OrchestratorID != "main-session" ||
		task.WorkerRole != "CODER" || task.ClaimSessionID != "sess-meta" {
		t.Fatalf("claim metadata not persisted: %+v", task)
	}
	if task.ClaimedAt == nil {
		t.Fatal("expected claimed_at to be set")
	}
}
