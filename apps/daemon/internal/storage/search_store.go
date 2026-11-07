package storage

import (
	"context"
	"fmt"
	"strings"

	"zenf/internal/domain"
)

// SearchRepository provides cross-entity search over projects, plans, tasks,
// and the document FTS index.
type SearchRepository struct {
	db *DB
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

func (r *SearchRepository) Search(ctx context.Context, query string, projectSlug string, limit int) ([]domain.SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []domain.SearchResult{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	pattern := "%" + escapeLike(query) + "%"
	results := []domain.SearchResult{}

	appendRows := func(sqlQuery string, args ...any) error {
		rows, err := r.db.Query(ctx, sqlQuery, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var hit domain.SearchResult
			if err := rows.Scan(&hit.EntityType, &hit.EntityID, &hit.Title, &hit.ProjectSlug); err != nil {
				return err
			}
			results = append(results, hit)
			if len(results) >= limit {
				return nil
			}
		}
		return rows.Err()
	}

	projectSQL := `
		SELECT 'project', slug, name, slug
		FROM projects
		WHERE slug LIKE ? ESCAPE '\' OR name LIKE ? ESCAPE '\' OR description LIKE ? ESCAPE '\'
		ORDER BY slug ASC LIMIT ?`
	if projectSlug != "" {
		projectSQL = `
			SELECT 'project', slug, name, slug
			FROM projects
			WHERE slug = ? AND (slug LIKE ? ESCAPE '\' OR name LIKE ? ESCAPE '\' OR description LIKE ? ESCAPE '\')
			ORDER BY slug ASC LIMIT ?`
		if err := appendRows(projectSQL, projectSlug, pattern, pattern, pattern, limit); err != nil {
			return nil, fmt.Errorf("failed to search projects: %w", err)
		}
	} else if err := appendRows(projectSQL, pattern, pattern, pattern, limit); err != nil {
		return nil, fmt.Errorf("failed to search projects: %w", err)
	}

	if len(results) < limit {
		planSQL := `
			SELECT 'plan', slug, title, project_slug
			FROM plans
			WHERE (slug LIKE ? ESCAPE '\' OR title LIKE ? ESCAPE '\')`
		args := []any{pattern, pattern}
		if projectSlug != "" {
			planSQL += " AND project_slug = ?"
			args = append(args, projectSlug)
		}
		planSQL += " ORDER BY slug ASC LIMIT ?"
		args = append(args, limit-len(results))
		if err := appendRows(planSQL, args...); err != nil {
			return nil, fmt.Errorf("failed to search plans: %w", err)
		}
	}

	if len(results) < limit {
		taskSQL := `
			SELECT 'task', id, title, COALESCE(project_slug, '')
			FROM tasks
			WHERE (id LIKE ? ESCAPE '\' OR title LIKE ? ESCAPE '\' OR description LIKE ? ESCAPE '\')`
		args := []any{pattern, pattern, pattern}
		if projectSlug != "" {
			taskSQL += " AND project_slug = ?"
			args = append(args, projectSlug)
		}
		taskSQL += " ORDER BY updated_at DESC LIMIT ?"
		args = append(args, limit-len(results))
		if err := appendRows(taskSQL, args...); err != nil {
			return nil, fmt.Errorf("failed to search tasks: %w", err)
		}
	}

	if len(results) < limit {
		match := ftsQuery(query)
		if match != "" {
			docSQL := `
				SELECT 'document', documents.id, documents.title,
				       COALESCE(documents.project_slug, '')
				FROM documents_fts
				JOIN documents ON documents.id = documents_fts.document_id
				WHERE documents_fts MATCH ?`
			args := []any{match}
			if projectSlug != "" {
				docSQL += " AND (documents.project_slug = ? OR documents.project_slug IS NULL)"
				args = append(args, projectSlug)
			}
			docSQL += " ORDER BY rank LIMIT ?"
			args = append(args, limit-len(results))
			if err := appendRows(docSQL, args...); err != nil {
				return nil, fmt.Errorf("failed to search documents: %w", err)
			}
		}
	}

	return results, nil
}
