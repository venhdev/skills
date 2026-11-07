package storage_test

import (
	"errors"
	"testing"

	"zenf/internal/domain"
	"zenf/internal/storage"
)

func newDocument(project, key string) domain.Document {
	return domain.Document{
		ProjectSlug: project,
		Key:         key,
		Slug:        key,
		Type:        domain.DocTypeArchitecture,
		Scope:       domain.DocScopeProject,
		Title:       "Title " + key,
		Path:        "/tmp/docs/" + key + ".md",
		Content:     "architecture and design notes for " + key,
		Summary:     "summary for " + key,
		Tags:        []string{"Architecture", " core ", "architecture"},
	}
}

func TestDocumentStore_CreateDefaultsAndTags(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")

	document := newDocument("alpha", "ARCH")
	if err := store.CreateDocument(ctx, document); err != nil {
		t.Fatalf("CreateDocument failed: %v", err)
	}

	created, err := store.GetDocumentByKey(ctx, "alpha", "ARCH")
	if err != nil {
		t.Fatalf("GetDocumentByKey failed: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected generated document id")
	}
	if created.Status != domain.DocStatusActive || created.Version != 1 {
		t.Fatalf("unexpected defaults: %+v", created)
	}
	if created.ContentHash == "" || created.ByteSize == 0 {
		t.Fatalf("expected content hash and byte size to be provided: %+v", created)
	}
	if len(created.Tags) != 2 || created.Tags[0] != "architecture" || created.Tags[1] != "core" {
		t.Fatalf("tags not normalized: %+v", created.Tags)
	}

	bySlug, err := store.GetDocumentBySlug(ctx, "alpha", "ARCH")
	if err != nil || bySlug.ID != created.ID {
		t.Fatalf("GetDocumentBySlug mismatch: %+v, err=%v", bySlug, err)
	}
}

func TestDocumentStore_UniquenessPerProjectAndGlobal(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedProject(t, store, ctx, "beta")

	if err := store.CreateDocument(ctx, newDocument("alpha", "ARCH")); err != nil {
		t.Fatalf("create alpha doc failed: %v", err)
	}
	if err := store.CreateDocument(ctx, newDocument("alpha", "ARCH")); !errors.Is(err, storage.ErrAlreadyExists) {
		t.Fatalf("expected duplicate key in same project to fail, got %v", err)
	}
	if err := store.CreateDocument(ctx, newDocument("beta", "ARCH")); err != nil {
		t.Fatalf("same key in another project should be allowed: %v", err)
	}

	global := newDocument("", "RULES")
	global.Scope = domain.DocScopeGlobal
	if err := store.CreateDocument(ctx, global); err != nil {
		t.Fatalf("create global doc failed: %v", err)
	}
	if err := store.CreateDocument(ctx, global); !errors.Is(err, storage.ErrAlreadyExists) {
		t.Fatalf("expected duplicate global key to fail, got %v", err)
	}
}

func TestDocumentStore_Validation(t *testing.T) {
	store, ctx := newTestStore(t)

	cases := []domain.Document{
		{Key: "NO-PATH", Slug: "no-path", Type: domain.DocTypeSpec, Scope: domain.DocScopeGlobal, Title: "No path"},
		{Key: "NO-TITLE", Slug: "no-title", Type: domain.DocTypeSpec, Scope: domain.DocScopeGlobal, Path: "/tmp/x.md"},
		{Key: "BAD-TYPE", Slug: "bad", Type: "NOPE", Scope: domain.DocScopeGlobal, Title: "x", Path: "/tmp/x.md"},
		{
			Key: "PLAN-SCOPE", Slug: "plan", Type: domain.DocTypeSpec,
			Scope: domain.DocScopePlan, Title: "x", Path: "/tmp/x.md",
		},
	}
	for _, document := range cases {
		if err := store.CreateDocument(ctx, document); err == nil {
			t.Fatalf("expected validation error for %+v", document)
		}
	}
}

func TestDocumentStore_ListFilters(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedProject(t, store, ctx, "beta")

	arch := newDocument("alpha", "ARCH")
	global := newDocument("", "RULES")
	global.Scope = domain.DocScopeGlobal
	global.Type = domain.DocTypeRule
	global.Tags = []string{"rule"}
	betaDoc := newDocument("beta", "SPEC")
	betaDoc.Type = domain.DocTypeSpec
	betaDoc.Tags = []string{"spec"}

	for _, document := range []domain.Document{arch, global, betaDoc} {
		if err := store.CreateDocument(ctx, document); err != nil {
			t.Fatalf("seed document failed: %v", err)
		}
	}

	projectOnly, err := store.ListDocuments(ctx, domain.DocumentFilter{ProjectSlug: "alpha"})
	if err != nil || len(projectOnly) != 1 || projectOnly[0].Key != "ARCH" {
		t.Fatalf("unexpected project-only list: %+v, err=%v", projectOnly, err)
	}
	withGlobal, err := store.ListDocuments(ctx, domain.DocumentFilter{ProjectSlug: "alpha", IncludeGlobal: true})
	if err != nil || len(withGlobal) != 2 {
		t.Fatalf("unexpected include-global list: %+v, err=%v", withGlobal, err)
	}
	byType, err := store.ListDocuments(ctx, domain.DocumentFilter{Type: domain.DocTypeRule})
	if err != nil || len(byType) != 1 || byType[0].Key != "RULES" {
		t.Fatalf("unexpected type filter: %+v, err=%v", byType, err)
	}
	byTag, err := store.ListDocuments(ctx, domain.DocumentFilter{Tag: "spec"})
	if err != nil || len(byTag) != 1 || byTag[0].Key != "SPEC" {
		t.Fatalf("unexpected tag filter: %+v, err=%v", byTag, err)
	}
	limited, err := store.ListDocuments(ctx, domain.DocumentFilter{Limit: 1})
	if err != nil || len(limited) != 1 {
		t.Fatalf("unexpected limit result: %+v, err=%v", limited, err)
	}
}

func TestDocumentStore_UpdateRefreshArchiveDelete(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	if err := store.CreateDocument(ctx, newDocument("alpha", "ARCH")); err != nil {
		t.Fatalf("create document failed: %v", err)
	}
	document, _ := store.GetDocumentByKey(ctx, "alpha", "ARCH")

	if err := store.UpdateDocument(ctx, document.ID, domain.DocumentPatch{
		Title:   ptr("Renamed"),
		Summary: ptr("new summary"),
		Path:    ptr("/tmp/docs/renamed.md"),
		Tags:    &[]string{"new-tag"},
	}); err != nil {
		t.Fatalf("UpdateDocument failed: %v", err)
	}
	updated, _ := store.GetDocument(ctx, document.ID)
	if updated.Title != "Renamed" || updated.Summary != "new summary" || updated.Path != "/tmp/docs/renamed.md" {
		t.Fatalf("metadata patch not applied: %+v", updated)
	}
	if len(updated.Tags) != 1 || updated.Tags[0] != "new-tag" {
		t.Fatalf("tags not replaced: %+v", updated.Tags)
	}

	if err := store.RefreshDocument(ctx, document.ID, "fresh content about widgets", "hash-fresh", 26); err != nil {
		t.Fatalf("RefreshDocument failed: %v", err)
	}
	refreshed, _ := store.GetDocument(ctx, document.ID)
	if refreshed.Version != document.Version+1 || refreshed.Content != "fresh content about widgets" ||
		refreshed.ContentHash != "hash-fresh" || refreshed.ByteSize != 26 {
		t.Fatalf("refresh not applied: %+v", refreshed)
	}

	if err := store.ArchiveDocument(ctx, document.ID, true); err != nil {
		t.Fatalf("ArchiveDocument failed: %v", err)
	}
	archived, _ := store.GetDocument(ctx, document.ID)
	if archived.Status != domain.DocStatusArchived {
		t.Fatalf("status = %s, want ARCHIVED", archived.Status)
	}

	if err := store.DeleteDocument(ctx, document.ID); err != nil {
		t.Fatalf("DeleteDocument failed: %v", err)
	}
	if _, err := store.GetDocument(ctx, document.ID); !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected document deleted, got %v", err)
	}
}

func TestDocumentStore_SearchUsesFTSTriggers(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	seedProject(t, store, ctx, "beta")

	arch := newDocument("alpha", "ARCH")
	arch.Content = "The claim handshake uses an atomic database transaction."
	if err := store.CreateDocument(ctx, arch); err != nil {
		t.Fatalf("create alpha doc failed: %v", err)
	}
	global := newDocument("", "GLOBAL-RULES")
	global.Scope = domain.DocScopeGlobal
	global.Content = "Global rule about atomic transactions and locking."
	if err := store.CreateDocument(ctx, global); err != nil {
		t.Fatalf("create global doc failed: %v", err)
	}
	beta := newDocument("beta", "BETA-SPEC")
	beta.Content = "Beta mentions transactions differently."
	if err := store.CreateDocument(ctx, beta); err != nil {
		t.Fatalf("create beta doc failed: %v", err)
	}

	results, err := store.SearchDocuments(ctx, "atomic transactions",
		domain.DocumentFilter{ProjectSlug: "alpha", IncludeGlobal: true})
	if err != nil {
		t.Fatalf("SearchDocuments failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 hits for alpha+global, got %d: %+v", len(results), results)
	}
	for _, result := range results {
		if result.Document.ProjectSlug == "beta" {
			t.Fatalf("project filter leaked beta hit: %+v", result)
		}
		if result.Snippet == "" {
			t.Fatalf("expected a snippet, got none for %+v", result.Document.Key)
		}
	}

	if _, err := store.SearchDocuments(ctx, "", domain.DocumentFilter{}); err != nil {
		t.Fatalf("empty query should not fail: %v", err)
	}
}

func TestDocumentStore_Links(t *testing.T) {
	store, ctx := newTestStore(t)
	seedProject(t, store, ctx, "alpha")
	if err := store.CreateDocument(ctx, newDocument("alpha", "ARCH")); err != nil {
		t.Fatalf("create document failed: %v", err)
	}
	if err := store.CreateDocument(ctx, newDocument("alpha", "SPEC")); err != nil {
		t.Fatalf("create document failed: %v", err)
	}
	arch, _ := store.GetDocumentByKey(ctx, "alpha", "ARCH")
	spec, _ := store.GetDocumentByKey(ctx, "alpha", "SPEC")

	if err := store.CreateDocumentLink(ctx, domain.DocumentLink{
		FromID: arch.ID, ToType: "document", ToID: spec.ID,
	}); err != nil {
		t.Fatalf("CreateDocumentLink failed: %v", err)
	}
	// Duplicate link creation is idempotent.
	if err := store.CreateDocumentLink(ctx, domain.DocumentLink{
		FromID: arch.ID, ToType: "document", ToID: spec.ID,
	}); err != nil {
		t.Fatalf("duplicate CreateDocumentLink failed: %v", err)
	}

	links, err := store.ListDocumentLinks(ctx, arch.ID)
	if err != nil {
		t.Fatalf("ListDocumentLinks failed: %v", err)
	}
	if len(links) != 1 || links[0].ToID != spec.ID {
		t.Fatalf("unexpected links: %+v", links)
	}
}
