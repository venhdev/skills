package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"zenf/internal/domain"
)
func (s *Server) Broadcast(evt domain.EventMessage) {
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}

	// Distribute non-blocking to active SSE clients
	s.sseMu.RLock()
	defer s.sseMu.RUnlock()

	for ch := range s.sseClients {
		select {
		case ch <- evt:
		default:
			// Slow consumer, skip to prevent stalling control plane
		}
	}
}

// eventPayload encodes a broadcast payload as JSON. Building payloads with
// fmt.Sprintf produced invalid JSON for identifiers containing quotes.
func eventPayload(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(encoded)
}


// handleListEvents handles GET /api/v1/events?after_id=&limit=
func (s *Server) handleListEvents(w http.ResponseWriter, r *http.Request) {
	afterID := int64(0)
	if raw := r.URL.Query().Get("after_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 0 {
			writeHttpError(w, "invalid after_id", http.StatusBadRequest)
			return
		}
		afterID = parsed
	}

	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeHttpError(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}

	events, err := s.storage.ListEvents(r.Context(), afterID, limit)
	if err != nil {
		writeHttpError(w, fmt.Sprintf("failed to list events: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"events": events})
}

// handleEventStream handles GET /api/v1/events/stream
func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeHttpError(w, "streaming unsupported by client", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientCh := make(chan domain.EventMessage, 32)

	s.sseMu.Lock()
	s.sseClients[clientCh] = struct{}{}
	s.sseMu.Unlock()

	defer func() {
		s.sseMu.Lock()
		delete(s.sseClients, clientCh)
		s.sseMu.Unlock()
	}()

	// Send initial connect acknowledgement
	fmt.Fprintf(w, "event: ping\ndata: connected\n\n")
	flusher.Flush()

	// Idle SSE connections are closed by proxies and browsers; a periodic
	// comment keeps the stream alive without inventing domain events.
	keepAlive := time.NewTicker(25 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-s.done:
			return
		case <-keepAlive.C:
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case evt, ok := <-clientCh:
			if !ok {
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evt.Type, string(data))
			flusher.Flush()
		}
	}
}

