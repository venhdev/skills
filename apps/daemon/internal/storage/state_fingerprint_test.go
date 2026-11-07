package storage_test

import (
	"strconv"
	"testing"

	"zenf/internal/domain"
)

func TestStateFingerprint_TracksEveryMutation(t *testing.T) {
	store, ctx := newTestStore(t)

	initial, err := store.StateFingerprint(ctx)
	if err != nil {
		t.Fatalf("StateFingerprint failed: %v", err)
	}
	if initial != "0" {
		t.Fatalf("initial fingerprint = %s, want 0", initial)
	}

	read := func() int64 {
		t.Helper()
		value, err := store.StateFingerprint(ctx)
		if err != nil {
			t.Fatalf("StateFingerprint failed: %v", err)
		}
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			t.Fatalf("fingerprint %q is not numeric: %v", value, err)
		}
		return parsed
	}

	seedProject(t, store, ctx, "alpha")
	afterProject := read()
	if afterProject <= 0 {
		t.Fatalf("expected fingerprint to advance after project create, got %d", afterProject)
	}

	seedPlan(t, store, ctx, "alpha", "p1", []domain.Task{{ID: "A", Title: "A"}})
	afterPlan := read()
	if afterPlan <= afterProject {
		t.Fatalf("expected fingerprint to advance after plan create, got %d <= %d", afterPlan, afterProject)
	}

	if err := store.UpdateTaskStatus(ctx, "A", domain.StatusReady, domain.StatusRunning); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}
	afterTransition := read()
	if afterTransition <= afterPlan {
		t.Fatalf("expected fingerprint to advance after task transition, got %d <= %d", afterTransition, afterPlan)
	}
}
