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

func (c *blockingCompiler) Compile(ctx context.Context, task domain.Task, attachments []domain.Artifact) (*domain.ContextPacket, error) {
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
		Task:             task,
		InputAttachments: attachments,
	}, nil
}

func (c *blockingCompiler) Release() {
	c.releaseOnce.Do(func() { close(c.release) })
}

func seedClaimTasks(t *testing.T, store *Store, ids ...string) {
	t.Helper()
	tasks := make([]domain.Task, len(ids))
	for i, id := range ids {
		tasks[i] = domain.Task{
			ID:           id,
			PlanID:       "claim-preflight",
			Title:        id,
			Status:       domain.StatusReady,
			Dependencies: []string{},
		}
	}
	if err := store.SavePlanWithTasks(context.Background(), domain.Plan{
		Slug:   "claim-preflight",
		Title:  "Claim Preflight",
		Status: "ACTIVE",
	}, tasks); err != nil {
		t.Fatalf("SavePlanWithTasks failed: %v", err)
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

func TestClaimTaskWithMeta_BuildsPacketOutsideWriteLock(t *testing.T) {
	compiler := &blockingCompiler{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	store, ctx := newClaimTestStore(t, compiler)
	seedClaimTasks(t, store, "T-CLAIM-A", "T-WRITE-B")
	defer compiler.Release()

	claimErr := make(chan error, 1)
	go func() {
		_, err := store.ClaimTaskWithMeta(ctx, "T-CLAIM-A", "worker-a", "", "CODER")
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
		t.Fatalf("ClaimTaskWithMeta failed: %v", err)
	}

	task, err := store.GetTask(ctx, "T-CLAIM-A")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if task.Status != domain.StatusRunning {
		t.Fatalf("expected claimed task RUNNING, got %s", task.Status)
	}
}

func TestClaimTaskWithMeta_PacketFailureLeavesTaskReady(t *testing.T) {
	compiler := &blockingCompiler{
		started: make(chan struct{}),
		release: make(chan struct{}),
		err:     errors.New("prompt compilation failed"),
	}
	compiler.Release()
	store, ctx := newClaimTestStore(t, compiler)
	seedClaimTasks(t, store, "T-FAIL")

	if _, err := store.ClaimTaskWithMeta(ctx, "T-FAIL", "worker-fail", "", "CODER"); err == nil {
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

func TestClaimTaskWithMeta_ConcurrentSameTask(t *testing.T) {
	store, ctx := newClaimTestStore(t, nil)
	seedClaimTasks(t, store, "T-RACE")

	const workers = 8
	results := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, err := store.ClaimTaskWithMeta(ctx, "T-RACE", "worker-race", "", "CODER")
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
