package domain

import (
	"fmt"
	"strings"
	"time"
)

type DocumentType string

const (
	DocTypeArchitecture DocumentType = "ARCHITECTURE"
	DocTypeADR          DocumentType = "ADR"
	DocTypeSpec         DocumentType = "SPEC"
	DocTypeRule         DocumentType = "RULE"
	DocTypeGuide        DocumentType = "GUIDE"
	DocTypeReference    DocumentType = "REFERENCE"
	DocTypeMemory       DocumentType = "MEMORY"
	DocTypeOther        DocumentType = "OTHER"
)

func (t DocumentType) IsValid() bool {
	switch t {
	case DocTypeArchitecture, DocTypeADR, DocTypeSpec, DocTypeRule,
		DocTypeGuide, DocTypeReference, DocTypeMemory, DocTypeOther:
		return true
	default:
		return false
	}
}

type DocumentScope string

const (
	DocScopeGlobal  DocumentScope = "GLOBAL"
	DocScopeProject DocumentScope = "PROJECT"
	DocScopePlan    DocumentScope = "PLAN"
	DocScopeTask    DocumentScope = "TASK"
)

func (s DocumentScope) IsValid() bool {
	switch s {
	case DocScopeGlobal, DocScopeProject, DocScopePlan, DocScopeTask:
		return true
	default:
		return false
	}
}

type DocumentStatus string

const (
	DocStatusDraft      DocumentStatus = "DRAFT"
	DocStatusActive     DocumentStatus = "ACTIVE"
	DocStatusSuperseded DocumentStatus = "SUPERSEDED"
	DocStatusArchived   DocumentStatus = "ARCHIVED"
)

func (s DocumentStatus) IsValid() bool {
	switch s {
	case DocStatusDraft, DocStatusActive, DocStatusSuperseded, DocStatusArchived:
		return true
	default:
		return false
	}
}

// Document is a registry record for a file-canonical document. The file at Path
// is the source of truth; Content is a searchable snapshot captured at
// registration or refresh time.
type Document struct {
	ID          string         `json:"id"`
	ProjectSlug string         `json:"project_slug,omitempty"`
	Key         string         `json:"key"`
	Slug        string         `json:"slug"`
	Type        DocumentType   `json:"type"`
	Scope       DocumentScope  `json:"scope"`
	ScopeRef    string         `json:"scope_ref,omitempty"`
	Title       string         `json:"title"`
	Path        string         `json:"path"`
	ContentHash string         `json:"content_hash"`
	ByteSize    int64          `json:"byte_size"`
	MimeType    string         `json:"mime_type"`
	Status      DocumentStatus `json:"status"`
	Summary     string         `json:"summary,omitempty"`
	Content     string         `json:"content,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Version     int            `json:"version"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// DocumentBrief is the compact reference embedded in a task context packet.
type DocumentBrief struct {
	ID      string        `json:"id"`
	Key     string        `json:"key"`
	Slug    string        `json:"slug"`
	Title   string        `json:"title"`
	Type    DocumentType  `json:"type"`
	Scope   DocumentScope `json:"scope"`
	Path    string        `json:"path"`
	Summary string        `json:"summary,omitempty"`
}

func (d *Document) Brief() DocumentBrief {
	return DocumentBrief{
		ID: d.ID, Key: d.Key, Slug: d.Slug, Title: d.Title,
		Type: d.Type, Scope: d.Scope, Path: d.Path, Summary: d.Summary,
	}
}

func (d *Document) Validate() error {
	if d.ID == "" {
		return fmt.Errorf("document id cannot be empty")
	}
	if d.Key == "" {
		return fmt.Errorf("document key cannot be empty")
	}
	if strings.ContainsAny(d.Key, " \t\n") {
		return fmt.Errorf("document key %q cannot contain whitespace", d.Key)
	}
	if d.Slug == "" {
		return fmt.Errorf("document slug cannot be empty")
	}
	if d.Title == "" {
		return fmt.Errorf("document title cannot be empty")
	}
	if !d.Type.IsValid() {
		return fmt.Errorf("invalid document type: %s", d.Type)
	}
	if !d.Scope.IsValid() {
		return fmt.Errorf("invalid document scope: %s", d.Scope)
	}
	if !d.Status.IsValid() {
		return fmt.Errorf("invalid document status: %s", d.Status)
	}
	if d.Path == "" {
		return fmt.Errorf("document path cannot be empty")
	}
	if (d.Scope == DocScopePlan || d.Scope == DocScopeTask) && d.ScopeRef == "" {
		return fmt.Errorf("document scope %s requires scope_ref", d.Scope)
	}
	return nil
}

// DocumentPatch carries partial metadata updates. Content is only replaced by
// the explicit refresh operation.
type DocumentPatch struct {
	Title   *string
	Summary *string
	Status  *DocumentStatus
	Path    *string
	Tags    *[]string
}

// DocumentFilter narrows a document query. Empty fields are not applied.
type DocumentFilter struct {
	ProjectSlug   string
	IncludeGlobal bool
	Type          DocumentType
	Scope         DocumentScope
	Status        DocumentStatus
	Tag           string
	Limit         int
}

// DocumentSearchResult is one FTS hit with a short snippet.
type DocumentSearchResult struct {
	Document Document `json:"document"`
	Snippet  string   `json:"snippet"`
}

// DocumentLink records an outbound reference from a document to another
// document or execution entity.
type DocumentLink struct {
	FromID    string    `json:"from_id"`
	ToType    string    `json:"to_type"`
	ToID      string    `json:"to_id"`
	CreatedAt time.Time `json:"created_at"`
}

// SearchResult is one cross-entity search hit.
type SearchResult struct {
	EntityType  string `json:"entity_type"`
	EntityID    string `json:"entity_id"`
	Title       string `json:"title"`
	Snippet     string `json:"snippet,omitempty"`
	ProjectSlug string `json:"project_slug,omitempty"`
}
