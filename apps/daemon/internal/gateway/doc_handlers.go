package gateway

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"zenfd/internal/domain"
)
func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := domain.DocumentFilter{
		ProjectSlug:   query.Get("project_slug"),
		IncludeGlobal: query.Get("include_global") == "true",
		Type:          domain.DocumentType(strings.ToUpper(query.Get("type"))),
		Scope:         domain.DocumentScope(strings.ToUpper(query.Get("scope"))),
		Status:        domain.DocumentStatus(strings.ToUpper(query.Get("status"))),
		Tag:           query.Get("tag"),
		Limit:         parseIntDefault(query.Get("limit"), 50),
	}
	documents, err := s.storage.ListDocuments(r.Context(), filter)
	if err != nil {
		writeHttpError(w, fmt.Sprintf("failed to list documents: %v", err), http.StatusInternalServerError)
		return
	}
	if documents == nil {
		documents = []domain.Document{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"documents": documents})
}

// readTextSnapshot captures a searchable snapshot for text-like documents. It
// deliberately skips binary/media files, which must be registered by path only.
func readTextSnapshot(path string) (string, bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() > 4<<20 {
		return "", false
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".markdown", ".txt", ".json", ".yaml", ".yml", ".xml", ".csv",
		".toml", ".sql", ".go", ".ts", ".tsx", ".js", ".jsx", ".css", ".html":
	default:
		return "", false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(raw), true
}

// POST /api/v1/docs
func (s *Server) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
	var document domain.Document
	if err := json.NewDecoder(r.Body).Decode(&document); err != nil {
		writeHttpError(w, fmt.Sprintf("invalid document payload: %v", err), http.StatusBadRequest)
		return
	}
	if document.Content == "" && document.Path != "" {
		if raw, ok := readTextSnapshot(document.Path); ok {
			document.Content = raw
		}
	}
	if document.Content != "" {
		if document.ContentHash == "" {
			sum := sha256.Sum256([]byte(document.Content))
			document.ContentHash = hex.EncodeToString(sum[:])
		}
		if document.ByteSize == 0 {
			document.ByteSize = int64(len(document.Content))
		}
	}
	if err := s.storage.CreateDocument(r.Context(), document); err != nil {
		writeStorageError(w, "create document", err)
		return
	}
	created, err := s.storage.GetDocumentByKey(r.Context(), document.ProjectSlug, document.Key)
	if err != nil {
		writeStorageError(w, "load document", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(created)
}

// GET /api/v1/docs/search?query=&project_slug=&include_global=true&type=&scope=&status=&limit=
func (s *Server) handleSearchDocuments(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get("query") == "" {
		writeHttpError(w, "query is required", http.StatusBadRequest)
		return
	}
	filter := domain.DocumentFilter{
		ProjectSlug:   query.Get("project_slug"),
		IncludeGlobal: query.Get("include_global") == "true",
		Type:          domain.DocumentType(strings.ToUpper(query.Get("type"))),
		Scope:         domain.DocumentScope(strings.ToUpper(query.Get("scope"))),
		Status:        domain.DocumentStatus(strings.ToUpper(query.Get("status"))),
		Limit:         parseIntDefault(query.Get("limit"), 25),
	}
	results, err := s.storage.SearchDocuments(r.Context(), query.Get("query"), filter)
	if err != nil {
		writeHttpError(w, fmt.Sprintf("failed to search documents: %v", err), http.StatusInternalServerError)
		return
	}
	if results == nil {
		results = []domain.DocumentSearchResult{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
}

// GET /api/v1/docs/{id}
func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	document, err := s.storage.GetDocument(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStorageError(w, "load document", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(document)
}

// PATCH /api/v1/docs/{id}
func (s *Server) handleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title   *string   `json:"title"`
		Summary *string   `json:"summary"`
		Status  *string   `json:"status"`
		Path    *string   `json:"path"`
		Tags    *[]string `json:"tags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHttpError(w, fmt.Sprintf("invalid payload: %v", err), http.StatusBadRequest)
		return
	}
	patch := domain.DocumentPatch{Title: req.Title, Summary: req.Summary, Path: req.Path, Tags: req.Tags}
	if req.Status != nil {
		status := domain.DocumentStatus(strings.ToUpper(*req.Status))
		patch.Status = &status
	}
	id := r.PathValue("id")
	if err := s.storage.UpdateDocument(r.Context(), id, patch); err != nil {
		writeStorageError(w, "update document", err)
		return
	}
	document, err := s.storage.GetDocument(r.Context(), id)
	if err != nil {
		writeStorageError(w, "load document", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(document)
}

// POST /api/v1/docs/{id}/refresh
func (s *Server) handleRefreshDocument(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Content     string `json:"content"`
		ContentHash string `json:"content_hash"`
		ByteSize    int64  `json:"byte_size"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	content := req.Content
	if content == "" {
		document, err := s.storage.GetDocument(r.Context(), id)
		if err != nil {
			writeStorageError(w, "load document", err)
			return
		}
		info, err := os.Stat(document.Path)
		if err != nil {
			writeHttpError(w, fmt.Sprintf("failed to read %s: %v", document.Path, err), http.StatusBadRequest)
			return
		}
		if info.IsDir() {
			writeHttpError(w, fmt.Sprintf("failed to read %s: path is a directory", document.Path), http.StatusBadRequest)
			return
		}
		if raw, ok := readTextSnapshot(document.Path); ok {
			content = raw
		}
	}
	hash := req.ContentHash
	size := req.ByteSize
	if content != "" {
		if hash == "" {
			sum := sha256.Sum256([]byte(content))
			hash = hex.EncodeToString(sum[:])
		}
		if size == 0 {
			size = int64(len(content))
		}
	}
	if err := s.storage.RefreshDocument(r.Context(), id, content, hash, size); err != nil {
		writeStorageError(w, "refresh document", err)
		return
	}
	document, err := s.storage.GetDocument(r.Context(), id)
	if err != nil {
		writeStorageError(w, "load document", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(document)
}

// DELETE /api/v1/docs/{id}
func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	if err := s.storage.DeleteDocument(r.Context(), r.PathValue("id")); err != nil {
		writeStorageError(w, "delete document", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Agents and sessions
// ---------------------------------------------------------------------------

// GET /api/v1/agents?include_disabled=true
