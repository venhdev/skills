package storage_test

import (
	"testing"

	"zenfd/internal/domain"
)

func TestSearchRepository_CrossEntitySearch(t *testing.T) {
	store, ctx := newTestStore(t)

	if err := store.CreateProject(ctx, domain.Project{Slug: "billing", Name: "Billing Platform"}); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	seedPlan(t, store, ctx, "billing", "billing-plan", []domain.Task{
		{ID: "BILL-1", Title: "Invoice billing flow", Description: "Build the billing pipeline"},
	})
	document := newDocument("billing", "BILLING-ARCH")
	document.Type = domain.DocTypeArchitecture
	document.Title = "Billing ledger architecture"
	document.Content = "The billing ledger reconciles invoice payments."
	if err := store.CreateDocument(ctx, document); err != nil {
		t.Fatalf("CreateDocument failed: %v", err)
	}

	if err := store.CreateProject(ctx, domain.Project{Slug: "other", Name: "Other Platform"}); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}
	seedPlan(t, store, ctx, "other", "other-plan", []domain.Task{
		{ID: "OTHER-1", Title: "Billing mention in another project"},
	})

	results, err := store.Search(ctx, "billing", "billing", 25)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected billing results")
	}
	seen := map[string]bool{}
	for _, result := range results {
		if result.EntityType == "project" {
			continue
		}
		if result.ProjectSlug != "billing" {
			t.Fatalf("project filter leaked foreign result: %+v", result)
		}
		seen[result.EntityType] = true
	}
	for _, entityType := range []string{"plan", "task", "document"} {
		if !seen[entityType] {
			t.Errorf("expected a %s result, got %+v", entityType, results)
		}
	}

	limited, err := store.Search(ctx, "billing", "", 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("expected limit 2, got %d", len(limited))
	}

	empty, err := store.Search(ctx, "   ", "", 25)
	if err != nil {
		t.Fatalf("empty Search failed: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected no results for empty query, got %+v", empty)
	}
}

func TestSearchRepository_EscapesLikeWildcards(t *testing.T) {
	store, ctx := newTestStore(t)
	if err := store.CreateProject(ctx, domain.Project{Slug: "alpha", Name: "Alpha 100% Ready"}); err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	if _, err := store.Search(ctx, "%", "", 25); err != nil {
		t.Fatalf("Search with wildcard failed: %v", err)
	}
	results, err := store.Search(ctx, "100%", "", 25)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 || results[0].EntityType != "project" {
		t.Fatalf("expected the project to match the literal percent, got %+v", results)
	}
}
