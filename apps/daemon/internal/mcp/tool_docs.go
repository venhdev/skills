package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"zenfd/internal/domain"
)

func (s *Server) registerDocumentTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("docs",
			mcp.WithDescription("Manage SSOT documents. Files stay canonical; the registry stores identity, metadata, tags, hash, and a searchable snapshot. Also supports FTS search."),
			mcp.WithString("action", mcp.Required(), mcp.Description("create | get | list | update | refresh | archive | unarchive | delete | search")),
			mcp.WithString("id", mcp.Description("Document id")),
			mcp.WithString("project_slug", mcp.Description("Project scope. Omit for a global document or an unscoped list.")),
			mcp.WithString("key", mcp.Description("Stable document key, unique per project")),
			mcp.WithString("slug", mcp.Description("Human-readable slug, unique per project (defaults to key)")),
			mcp.WithString("type", mcp.Description("ARCHITECTURE | ADR | SPEC | RULE | GUIDE | REFERENCE | MEMORY | OTHER")),
			mcp.WithString("scope", mcp.Description("GLOBAL | PROJECT | PLAN | TASK")),
			mcp.WithString("scope_ref", mcp.Description("Plan slug or task id when scope is PLAN/TASK")),
			mcp.WithString("title", mcp.Description("Document title")),
			mcp.WithString("path", mcp.Description("Absolute or repository-relative path of the canonical file")),
			mcp.WithString("summary", mcp.Description("Short summary used by search results")),
			mcp.WithString("content", mcp.Description("Content snapshot. When omitted on create/refresh, the file at path is read.")),
			mcp.WithString("content_hash", mcp.Description("SHA-256 of the content. Computed when omitted.")),
			mcp.WithNumber("byte_size", mcp.Description("Content size in bytes. Computed when omitted.")),
			mcp.WithString("mime_type", mcp.Description("MIME type (default text/markdown)")),
			mcp.WithString("status", mcp.Description("DRAFT | ACTIVE | SUPERSEDED | ARCHIVED")),
			mcp.WithArray("tags", mcp.Description("Normalized tags"), mcp.WithStringItems()),
			mcp.WithBoolean("include_global", mcp.Description("For project lists: include globally scoped documents")),
			mcp.WithString("tag", mcp.Description("Filter by one tag")),
			mcp.WithNumber("limit", mcp.Description("Result limit (default 50, max 100)")),
			mcp.WithString("query", mcp.Description("Full-text query for action=search")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := request.GetArguments()
			action := request.GetString("action", "")
			projectSlug := request.GetString("project_slug", "")

			switch action {
			case "create":
				document, err := s.documentFromRequest(ctx, request)
				if err != nil {
					return toolError("%v", err), nil
				}
				if err := s.store.CreateDocument(ctx, document); err != nil {
					return toolError("failed to create document: %v", err), nil
				}
				created, err := s.store.GetDocumentByKey(ctx, document.ProjectSlug, document.Key)
				if err != nil {
					return toolError("failed to load document %s: %v", document.ID, err), nil
				}
				return marshalToolResult(created), nil

			case "get":
				document, err := s.lookupDocument(ctx, request)
				if err != nil {
					return toolError("%v", err), nil
				}
				return marshalToolResult(document), nil

			case "list":
				filter := domain.DocumentFilter{
					ProjectSlug:   projectSlug,
					IncludeGlobal: request.GetBool("include_global", false),
					Type:          domain.DocumentType(request.GetString("type", "")),
					Scope:         domain.DocumentScope(request.GetString("scope", "")),
					Status:        domain.DocumentStatus(request.GetString("status", "")),
					Tag:           request.GetString("tag", ""),
					Limit:         request.GetInt("limit", 50),
				}
				documents, err := s.store.ListDocuments(ctx, filter)
				if err != nil {
					return toolError("failed to list documents: %v", err), nil
				}
				if documents == nil {
					documents = []domain.Document{}
				}
				return marshalToolResult(documents), nil

			case "update":
				id := request.GetString("id", "")
				if id == "" {
					return toolError("id is required for update"), nil
				}
				patch, err := parseDocumentPatch(args)
				if err != nil {
					return toolError("invalid update payload: %v", err), nil
				}
				if err := s.store.UpdateDocument(ctx, id, patch); err != nil {
					return toolError("failed to update document %s: %v", id, err), nil
				}
				updated, err := s.store.GetDocument(ctx, id)
				if err != nil {
					return toolError("failed to load document %s: %v", id, err), nil
				}
				return marshalToolResult(updated), nil

			case "refresh":
				id := request.GetString("id", "")
				if id == "" {
					return toolError("id is required for refresh"), nil
				}
				content := request.GetString("content", "")
				if content == "" {
					document, err := s.store.GetDocument(ctx, id)
					if err != nil {
						return toolError("failed to load document %s: %v", id, err), nil
					}
					content, err = readDocumentContent(document.Path)
					if err != nil {
						return toolError("failed to read %s: %v", document.Path, err), nil
					}
				}
				hash := request.GetString("content_hash", "")
				size := int64(request.GetInt("byte_size", len(content)))
				if content != "" {
					if hash == "" {
						hash = contentHash(content)
					}
				} else if hash == "" {
					size = 0
				}
				if err := s.store.RefreshDocument(ctx, id, content, hash, size); err != nil {
					return toolError("failed to refresh document %s: %v", id, err), nil
				}
				refreshed, err := s.store.GetDocument(ctx, id)
				if err != nil {
					return toolError("failed to load document %s: %v", id, err), nil
				}
				return marshalToolResult(refreshed), nil

			case "archive", "unarchive":
				id := request.GetString("id", "")
				if id == "" {
					return toolError("id is required for %s", action), nil
				}
				archived := action == "archive"
				if err := s.store.ArchiveDocument(ctx, id, archived); err != nil {
					return toolError("failed to %s document %s: %v", action, id, err), nil
				}
				return marshalToolResult(map[string]any{"id": id, "archived": archived}), nil

			case "delete":
				id := request.GetString("id", "")
				if id == "" {
					return toolError("id is required for delete"), nil
				}
				if err := s.store.DeleteDocument(ctx, id); err != nil {
					return toolError("failed to delete document %s: %v", id, err), nil
				}
				return marshalToolResult(map[string]any{"id": id, "deleted": true}), nil

			case "search":
				query := request.GetString("query", "")
				if query == "" {
					return toolError("query is required for search"), nil
				}
				filter := domain.DocumentFilter{
					ProjectSlug:   projectSlug,
					IncludeGlobal: request.GetBool("include_global", true),
					Type:          domain.DocumentType(request.GetString("type", "")),
					Scope:         domain.DocumentScope(request.GetString("scope", "")),
					Status:        domain.DocumentStatus(request.GetString("status", "")),
					Limit:         request.GetInt("limit", 25),
				}
				results, err := s.store.SearchDocuments(ctx, query, filter)
				if err != nil {
					return toolError("failed to search documents: %v", err), nil
				}
				if results == nil {
					results = []domain.DocumentSearchResult{}
				}
				return marshalToolResult(results), nil

			default:
				return toolError("unknown action (expected create, get, list, update, refresh, archive, unarchive, delete, or search)"), nil
			}
		},
	)
}

func (s *Server) documentFromRequest(ctx context.Context, request mcp.CallToolRequest) (domain.Document, error) {
	projectSlug := request.GetString("project_slug", "")
	key := request.GetString("key", "")
	slug := request.GetString("slug", "")
	if key == "" && slug == "" {
		return domain.Document{}, fmt.Errorf("key or slug is required")
	}
	if key == "" {
		key = slug
	}
	if slug == "" {
		slug = key
	}
	scope := domain.DocumentScope(request.GetString("scope", ""))
	if scope == "" {
		if projectSlug == "" {
			scope = domain.DocScopeGlobal
		} else {
			scope = domain.DocScopeProject
		}
	}
	if scope == domain.DocScopeGlobal {
		projectSlug = ""
	}

	docType := domain.DocumentType(strings.ToUpper(request.GetString("type", "")))
	if docType == "" {
		docType = domain.DocTypeOther
	}
	status := domain.DocumentStatus(strings.ToUpper(request.GetString("status", "")))
	if status == "" {
		status = domain.DocStatusActive
	}
	path := request.GetString("path", "")
	if path == "" {
		return domain.Document{}, fmt.Errorf("path is required")
	}

	content := request.GetString("content", "")
	if content == "" {
		read, err := readDocumentContent(path)
		if err == nil {
			content = read
		}
	}
	hash := request.GetString("content_hash", "")
	if hash == "" && content != "" {
		hash = contentHash(content)
	}
	size := int64(request.GetInt("byte_size", len(content)))
	mimeType := request.GetString("mime_type", "text/markdown")

	return domain.Document{
		ProjectSlug: projectSlug,
		Key:         key,
		Slug:        slug,
		Type:        docType,
		Scope:       scope,
		ScopeRef:    request.GetString("scope_ref", ""),
		Title:       request.GetString("title", slug),
		Path:        path,
		ContentHash: hash,
		ByteSize:    size,
		MimeType:    mimeType,
		Status:      status,
		Summary:     request.GetString("summary", ""),
		Content:     content,
		Version:     1,
		Tags:        request.GetStringSlice("tags", nil),
	}, nil
}

func (s *Server) lookupDocument(ctx context.Context, request mcp.CallToolRequest) (*domain.Document, error) {
	if id := request.GetString("id", ""); id != "" {
		return s.store.GetDocument(ctx, id)
	}
	projectSlug := request.GetString("project_slug", "")
	if key := request.GetString("key", ""); key != "" {
		return s.store.GetDocumentByKey(ctx, projectSlug, key)
	}
	if slug := request.GetString("slug", ""); slug != "" {
		return s.store.GetDocumentBySlug(ctx, projectSlug, slug)
	}
	return nil, fmt.Errorf("id, key, or slug is required")
}

func parseDocumentPatch(args map[string]any) (domain.DocumentPatch, error) {
	encoded, err := json.Marshal(args)
	if err != nil {
		return domain.DocumentPatch{}, err
	}
	var payload struct {
		Title   *string   `json:"title"`
		Summary *string   `json:"summary"`
		Status  *string   `json:"status"`
		Path    *string   `json:"path"`
		Tags    *[]string `json:"tags"`
	}
	if err := json.Unmarshal(encoded, &payload); err != nil {
		return domain.DocumentPatch{}, err
	}
	patch := domain.DocumentPatch{
		Title:   payload.Title,
		Summary: payload.Summary,
		Path:    payload.Path,
		Tags:    payload.Tags,
	}
	if payload.Status != nil {
		status := domain.DocumentStatus(*payload.Status)
		patch.Status = &status
	}
	return patch, nil
}

func readDocumentContent(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", path)
	}
	if info.Size() > 4<<20 {
		return "", fmt.Errorf("%s is larger than 4 MiB; register it without a content snapshot", path)
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".txt", ".json", ".yaml", ".yml", ".xml", ".csv",
		".toml", ".sql", ".go", ".ts", ".tsx", ".js", ".jsx", ".css", ".html":
	default:
		// Binary/media documents stay path-only; never index their bytes.
		return "", nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
