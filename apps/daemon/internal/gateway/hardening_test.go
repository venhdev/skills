package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"zenf/internal/storage"
)

func TestGateway_MutationRejectsForeignOrigin(t *testing.T) {
	_, baseURL, _, cleanup := setupTestGateway(t)
	defer cleanup()

	body := strings.NewReader(`{"slug":"x","project_slug":"test-project","title":"x"}`)

	foreign, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/plans", body)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	foreign.Header.Set("Content-Type", "application/json")
	foreign.Header.Set("Origin", "https://evil.example")
	resp, err := http.DefaultClient.Do(foreign)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign origin: expected 403, got %d", resp.StatusCode)
	}

	loopback, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/plans", strings.NewReader(`{"slug":"x","project_slug":"test-project","title":"x"}`))
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	loopback.Header.Set("Content-Type", "application/json")
	loopback.Header.Set("Origin", "http://localhost:5173")
	respLoop, err := http.DefaultClient.Do(loopback)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer respLoop.Body.Close()
	if respLoop.StatusCode != http.StatusCreated {
		t.Fatalf("loopback origin: expected 201, got %d", respLoop.StatusCode)
	}
	if got := respLoop.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("expected echoed origin, got %q", got)
	}
}

func TestGateway_ShutdownDoesNotWaitForSSEClients(t *testing.T) {
	tempDir := t.TempDir()
	db, err := storage.Open(tempDir + "/shutdown.db")
	if err != nil {
		t.Fatalf("failed to open storage: %v", err)
	}
	defer db.Close()

	srv := NewServer(storage.NewStore(db))
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	serveDone := make(chan struct{})
	go func() {
		_ = srv.Serve(ln)
		close(serveDone)
	}()

	resp, err := http.Get(fmt.Sprintf("http://%s/api/v1/events/stream", ln.Addr().String()))
	if err != nil {
		t.Fatalf("failed to open SSE stream: %v", err)
	}
	defer resp.Body.Close()

	// Wait for the connect acknowledgement so the handler is definitely live.
	reader := bufio.NewReader(resp.Body)
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatalf("failed to read SSE ack: %v", err)
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 1500*time.Millisecond {
		t.Fatalf("Shutdown waited %v for an open SSE client", elapsed)
	}

	select {
	case <-serveDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Serve did not return after Shutdown")
	}
}

func TestEventPayloadEscapesQuotes(t *testing.T) {
	payload := eventPayload(map[string]string{"slug": `a"b`})
	var decoded map[string]string
	if err := json.Unmarshal([]byte(payload), &decoded); err != nil {
		t.Fatalf("payload is not valid JSON (%q): %v", payload, err)
	}
	if decoded["slug"] != `a"b` {
		t.Errorf("round trip changed slug: %q", decoded["slug"])
	}
}
