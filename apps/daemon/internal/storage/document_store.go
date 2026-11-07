package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"zenfd/internal/domain"
)

// DocumentRepository persists the file-canonical document registry. Files stay
// authoritative; the row stores identity, metadata, tags, hash, a searchable
// snapshot, and the FTS index maintained by triggers.
type DocumentRepository struct {
	db *DB
}

const documentSelectColumns = `
id, COALESCE(project_slug, '') AS project_slug, key, slug, type, scope, scope_ref,
title, path, content_hash, byte_size, mime_type, status, summary, content, version,
created_at, updated_at`

// documentSearchSelectColumns qualifies every column because documents_fts
// exposes key, title, summary, and content as well.
const documentSearchSelectColumns = `
documents.id, COALESCE(documents.project_slug, '') AS project_slug, documents.key,
documents.slug, documents.type, documents.scope, documents.scope_ref,
documents.title, documents.path, documents.content_hash, documents.byte_size,
documents.mime_type, documents.status, documents.summary, documents.content,
documents.version, documents.created_at, documents.updated_at`

// newDocumentID returns a time-ordered, collision-resistant identifier. A
// ULID-compatible library is not worth a dependency for the MVP; the 48-bit
// millisecond prefix keeps ids sortable by creation time.
func newDocumentID() string {
	var random [10]byte
	_, _ = rand.Read(random[:])
	return fmt.Sprintf("doc_%012x%s", time.Now().UTC().UnixMilli(), hex.EncodeToString(random[:]))
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]bool, len(tags))
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		out = append(out, tag)
	}
	return out
}

func scanDocument(s scanner) (*domain.Document, error) {
	var d domain.Document
	var typeStr, scopeStr, statusStr string
	if err := s.Scan(
		&d.ID, &d.ProjectSlug, &d.Key, &d.Slug, &typeStr, &scopeStr, &d.ScopeRef,
		&d.Title, &d.Path, &d.ContentHash, &d.ByteSize, &d.MimeType, &statusStr,
		&d.Summary, &d.Content, &d.Version, &d.CreatedAt, &d.UpdatedAt,
	); err != nil {
		return nil, err
	}
	d.Type = domain.DocumentType(typeStr)
	d.Scope = domain.DocumentScope(scopeStr)
	d.Status = domain.DocumentStatus(statusStr)
	return &d, nil
}

func (r *DocumentRepository) loadTags(ctx context.Context, documentID string) ([]string, error) {
	rows, err := r.db.Query(ctx,
		"SELECT tag FROM document_tags WHERE document_id = ? ORDER BY tag ASC", documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tags for document %s: %w", documentID, err)
	}
	defer rows.Close()

	tags := []string{}
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("failed to scan document tag: %w", err)
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func (r *DocumentRepository) CreateDocument(ctx context.Context, document domain.Document) error {
	now := time.Now().UTC()
	if document.ID == "" {
		document.ID = newDocumentID()
	}
	if document.CreatedAt.IsZero() {
		document.CreatedAt = now
	}
	document.UpdatedAt = now
	if document.Status == "" {
		document.Status = domain.DocStatusActive
	}
	if document.Scope == "" {
		document.Scope = domain.DocScopeProject
	}
	if document.Type == "" {
		document.Type = domain.DocTypeOther
	}
	if document.Version == 0 {
		document.Version = 1
	}
	if document.MimeType == "" {
		document.MimeType = "text/markdown"
	}
	if document.Content != "" {
		if document.ContentHash == "" {
			document.ContentHash = domain.ComputeContentHash(document.Content)
		}
		if document.ByteSize == 0 {
			document.ByteSize = int64(len(document.Content))
		}
	}
	if err := document.Validate(); err != nil {
		return err
	}

	tags := normalizeTags(document.Tags)
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		if err := insertDocumentTx(ctx, tx, document); err != nil {
			if isUniqueViolation(err) {
				return fmt.Errorf("%w: document key %s in project %q", ErrAlreadyExists, document.Key, document.ProjectSlug)
			}
			return err
		}
		if err := replaceDocumentTagsTx(ctx, tx, document.ID, tags); err != nil {
			return err
		}
		return nil
	})
}

func insertDocumentTx(ctx context.Context, tx *sql.Tx, document domain.Document) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO documents (
			id, project_slug, key, slug, type, scope, scope_ref, title, path,
			content_hash, byte_size, mime_type, status, summary, content, version,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		document.ID, nullableString(document.ProjectSlug), document.Key, document.Slug,
		string(document.Type), string(document.Scope), document.ScopeRef, document.Title,
		document.Path, document.ContentHash, document.ByteSize, document.MimeType,
		string(document.Status), document.Summary, document.Content, document.Version,
		document.CreatedAt, document.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create document %s: %w", document.Key, err)
	}
	return nil
}

func replaceDocumentTagsTx(ctx context.Context, tx *sql.Tx, documentID string, tags []string) error {
	if _, err := tx.ExecContext(ctx, "DELETE FROM document_tags WHERE document_id = ?", documentID); err != nil {
		return fmt.Errorf("failed to clear tags for document %s: %w", documentID, err)
	}
	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO document_tags (document_id, tag) VALUES (?, ?)", documentID, tag); err != nil {
			return fmt.Errorf("failed to tag document %s with %s: %w", documentID, tag, err)
		}
	}
	return nil
}

func (r *DocumentRepository) finalizeDocument(ctx context.Context, document *domain.Document) error {
	tags, err := r.loadTags(ctx, document.ID)
	if err != nil {
		return err
	}
	document.Tags = tags
	return nil
}

func (r *DocumentRepository) GetDocument(ctx context.Context, id string) (*domain.Document, error) {
	row := r.db.QueryRow(ctx, "SELECT "+documentSelectColumns+" FROM documents WHERE id = ?", id)
	document, err := scanDocument(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: document %s", ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query document %s: %w", id, err)
	}
	if err := r.finalizeDocument(ctx, document); err != nil {
		return nil, err
	}
	return document, nil
}

func (r *DocumentRepository) GetDocumentByKey(ctx context.Context, projectSlug string, key string) (*domain.Document, error) {
	row := r.db.QueryRow(ctx,
		"SELECT "+documentSelectColumns+" FROM documents WHERE IFNULL(project_slug, '') = ? AND key = ?",
		projectSlug, key)
	document, err := scanDocument(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: document %s in project %q", ErrNotFound, key, projectSlug)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query document %s: %w", key, err)
	}
	if err := r.finalizeDocument(ctx, document); err != nil {
		return nil, err
	}
	return document, nil
}

func (r *DocumentRepository) GetDocumentBySlug(ctx context.Context, projectSlug string, slug string) (*domain.Document, error) {
	row := r.db.QueryRow(ctx,
		"SELECT "+documentSelectColumns+" FROM documents WHERE IFNULL(project_slug, '') = ? AND slug = ?",
		projectSlug, slug)
	document, err := scanDocument(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: document slug %s in project %q", ErrNotFound, slug, projectSlug)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query document slug %s: %w", slug, err)
	}
	if err := r.finalizeDocument(ctx, document); err != nil {
		return nil, err
	}
	return document, nil
}

func (r *DocumentRepository) ListDocuments(ctx context.Context, filter domain.DocumentFilter) ([]domain.Document, error) {
	query := "SELECT " + documentSelectColumns + " FROM documents"
	var conditions []string
	var args []any

	if filter.ProjectSlug != "" {
		if filter.IncludeGlobal {
			conditions = append(conditions, "(project_slug = ? OR project_slug IS NULL)")
		} else {
			conditions = append(conditions, "project_slug = ?")
		}
		args = append(args, filter.ProjectSlug)
	}
	if filter.Type != "" {
		conditions = append(conditions, "type = ?")
		args = append(args, string(filter.Type))
	}
	if filter.Scope != "" {
		conditions = append(conditions, "scope = ?")
		args = append(args, string(filter.Scope))
	}
	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, string(filter.Status))
	}
	if filter.Tag != "" {
		conditions = append(conditions,
			"EXISTS (SELECT 1 FROM document_tags dt WHERE dt.document_id = documents.id AND dt.tag = ?)")
		args = append(args, strings.ToLower(strings.TrimSpace(filter.Tag)))
	}
	if len(conditions) > 0 {
		query += " WHERE " + joinConditions(conditions)
	}
	query += " ORDER BY updated_at DESC, id ASC"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}
	defer rows.Close()

	documents := []domain.Document{}
	for rows.Next() {
		document, err := scanDocument(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		if err := r.finalizeDocument(ctx, document); err != nil {
			return nil, err
		}
		documents = append(documents, *document)
	}
	return documents, rows.Err()
}

func (r *DocumentRepository) UpdateDocument(ctx context.Context, id string, patch domain.DocumentPatch) error {
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		document, err := getDocumentTx(ctx, tx, id)
		if err != nil {
			return err
		}
		if patch.Title != nil {
			document.Title = *patch.Title
		}
		if patch.Summary != nil {
			document.Summary = *patch.Summary
		}
		if patch.Status != nil {
			document.Status = *patch.Status
		}
		if patch.Path != nil {
			document.Path = *patch.Path
		}
		if err := document.Validate(); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE documents
			SET title = ?, summary = ?, status = ?, path = ?, updated_at = ?
			WHERE id = ?`,
			document.Title, document.Summary, string(document.Status), document.Path, now, id); err != nil {
			return fmt.Errorf("failed to update document %s: %w", id, err)
		}
		if patch.Tags != nil {
			if err := replaceDocumentTagsTx(ctx, tx, id, normalizeTags(*patch.Tags)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *DocumentRepository) RefreshDocument(ctx context.Context, id string, content string, contentHash string, byteSize int64) error {
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `
			UPDATE documents
			SET content = ?, content_hash = ?, byte_size = ?, version = version + 1, updated_at = ?
			WHERE id = ?`,
			content, contentHash, byteSize, now, id)
		if err != nil {
			return fmt.Errorf("failed to refresh document %s: %w", id, err)
		}
		rows, err := affectedOrErr(res, "refresh document", id)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: document %s", ErrNotFound, id)
		}
		return nil
	})
}

func (r *DocumentRepository) ArchiveDocument(ctx context.Context, id string, archived bool) error {
	status := domain.DocStatusActive
	if archived {
		status = domain.DocStatusArchived
	}
	now := time.Now().UTC()
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "UPDATE documents SET status = ?, updated_at = ? WHERE id = ?",
			string(status), now, id)
		if err != nil {
			return fmt.Errorf("failed to archive document %s: %w", id, err)
		}
		rows, err := affectedOrErr(res, "archive document", id)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: document %s", ErrNotFound, id)
		}
		return nil
	})
}

func (r *DocumentRepository) DeleteDocument(ctx context.Context, id string) error {
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, "DELETE FROM documents WHERE id = ?", id)
		if err != nil {
			return fmt.Errorf("failed to delete document %s: %w", id, err)
		}
		rows, err := affectedOrErr(res, "delete document", id)
		if err != nil {
			return err
		}
		if rows == 0 {
			return fmt.Errorf("%w: document %s", ErrNotFound, id)
		}
		return nil
	})
}

func getDocumentTx(ctx context.Context, tx *sql.Tx, id string) (*domain.Document, error) {
	row := tx.QueryRowContext(ctx, "SELECT "+documentSelectColumns+" FROM documents WHERE id = ?", id)
	document, err := scanDocument(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("%w: document %s", ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query document %s: %w", id, err)
	}
	return document, nil
}

// ftsQuery turns user input into a safe FTS5 conjunction. Quoting every token
// prevents operators such as NEAR/OR from turning a lookup into a syntax error.
func ftsQuery(query string) string {
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return ""
	}
	terms := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.ReplaceAll(field, `"`, `""`)
		terms = append(terms, `"`+field+`"`)
	}
	return strings.Join(terms, " AND ")
}

func (r *DocumentRepository) SearchDocuments(ctx context.Context, query string, filter domain.DocumentFilter) ([]domain.DocumentSearchResult, error) {
	match := ftsQuery(query)
	if match == "" {
		return []domain.DocumentSearchResult{}, nil
	}

	sqlQuery := `
		SELECT ` + documentSearchSelectColumns + `, snippet(documents_fts, -1, '[', ']', '…', 16)
		FROM documents_fts
		JOIN documents ON documents.id = documents_fts.document_id
		WHERE documents_fts MATCH ?`
	args := []any{match}

	if filter.ProjectSlug != "" {
		if filter.IncludeGlobal {
			sqlQuery += " AND (documents.project_slug = ? OR documents.project_slug IS NULL)"
		} else {
			sqlQuery += " AND documents.project_slug = ?"
		}
		args = append(args, filter.ProjectSlug)
	}
	if filter.Type != "" {
		sqlQuery += " AND documents.type = ?"
		args = append(args, string(filter.Type))
	}
	if filter.Scope != "" {
		sqlQuery += " AND documents.scope = ?"
		args = append(args, string(filter.Scope))
	}
	if filter.Status != "" {
		sqlQuery += " AND documents.status = ?"
		args = append(args, string(filter.Status))
	}
	sqlQuery += " ORDER BY rank LIMIT ?"
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	args = append(args, limit)

	rows, err := r.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}
	defer rows.Close()

	results := []domain.DocumentSearchResult{}
	for rows.Next() {
		var d domain.Document
		var typeStr, scopeStr, statusStr, snippet string
		if err := rows.Scan(
			&d.ID, &d.ProjectSlug, &d.Key, &d.Slug, &typeStr, &scopeStr, &d.ScopeRef,
			&d.Title, &d.Path, &d.ContentHash, &d.ByteSize, &d.MimeType, &statusStr,
			&d.Summary, &d.Content, &d.Version, &d.CreatedAt, &d.UpdatedAt, &snippet,
		); err != nil {
			return nil, fmt.Errorf("failed to scan document search hit: %w", err)
		}
		d.Type = domain.DocumentType(typeStr)
		d.Scope = domain.DocumentScope(scopeStr)
		d.Status = domain.DocumentStatus(statusStr)
		tags, err := r.loadTags(ctx, d.ID)
		if err != nil {
			return nil, err
		}
		d.Tags = tags
		results = append(results, domain.DocumentSearchResult{Document: d, Snippet: snippet})
	}
	return results, rows.Err()
}

func (r *DocumentRepository) CreateDocumentLink(ctx context.Context, link domain.DocumentLink) error {
	if link.ToType == "" || link.ToID == "" {
		return fmt.Errorf("document link requires to_type and to_id")
	}
	if link.CreatedAt.IsZero() {
		link.CreatedAt = time.Now().UTC()
	}
	return r.db.ExecImmediate(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx,
			"INSERT OR IGNORE INTO document_links (from_id, to_type, to_id, created_at) VALUES (?, ?, ?, ?)",
			link.FromID, link.ToType, link.ToID, link.CreatedAt); err != nil {
			return fmt.Errorf("failed to create document link: %w", err)
		}
		return nil
	})
}

func (r *DocumentRepository) ListDocumentLinks(ctx context.Context, fromID string) ([]domain.DocumentLink, error) {
	rows, err := r.db.Query(ctx, `
		SELECT from_id, to_type, to_id, created_at
		FROM document_links WHERE from_id = ? ORDER BY created_at ASC, to_type ASC, to_id ASC`, fromID)
	if err != nil {
		return nil, fmt.Errorf("failed to list document links: %w", err)
	}
	defer rows.Close()

	links := []domain.DocumentLink{}
	for rows.Next() {
		var link domain.DocumentLink
		if err := rows.Scan(&link.FromID, &link.ToType, &link.ToID, &link.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan document link: %w", err)
		}
		links = append(links, link)
	}
	return links, rows.Err()
}
