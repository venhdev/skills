package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"

	"zenf/internal/domain"
)

// handleSearch handles GET /api/v1/search?q=&project_slug=&limit=
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	results, err := s.storage.Search(r.Context(), query.Get("q"), query.Get("project_slug"), parseIntDefault(query.Get("limit"), 25))
	if err != nil {
		writeHttpError(w, fmt.Sprintf("failed to search: %v", err), http.StatusInternalServerError)
		return
	}
	if results == nil {
		results = []domain.SearchResult{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
}
