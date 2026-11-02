package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func getFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestCLI_EndToEndLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "zenfd")

	// 1. Build CLI binary
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build zenfd: %v, out: %s", err, string(out))
	}

	port := getFreePort(t)
	dbPath := filepath.Join(tempDir, "e2e_state.db")
	daemonURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// 2. Start Daemon Process
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	daemonCmd := exec.CommandContext(ctx, binPath, "daemon",
		"--port", fmt.Sprintf("%d", port),
		"--db", dbPath,
		"--workers", "2",
		"--tick", "100",
	)
	var daemonOut bytes.Buffer
	daemonCmd.Stdout = &daemonOut
	daemonCmd.Stderr = &daemonOut

	if err := daemonCmd.Start(); err != nil {
		t.Fatalf("failed to start daemon process: %v", err)
	}
	defer func() {
		cancel()
		_ = daemonCmd.Wait()
	}()

	// Wait for daemon to become healthy
	healthy := false
	for i := 0; i < 50; i++ {
		resp, err := http.Get(daemonURL + "/api/v1/tasks")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			healthy = true
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !healthy {
		t.Fatalf("daemon failed to become healthy. Logs:\n%s", daemonOut.String())
	}

	// 3. Test `zenfd status`
	statusCmd := exec.Command(binPath, "status", "--url", daemonURL)
	out, err := statusCmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "healthy and active") {
		t.Fatalf("status command failed: %v, out: %s", err, string(out))
	}

	// 4. Create dummy markdown plan
	planFile := filepath.Join(tempDir, "plan.md")
	mdContent := `
# Plan Dossier: e2e-system · ACTIVE

| Task ID | Description | Dependencies | Status |
| :--- | :--- | :--- | :--- |
| **SLICE-01** | Root Slice | - | **READY** |
| **SLICE-02** | Dependent Slice | SLICE-01 | **PLANNED** |
`
	if err := os.WriteFile(planFile, []byte(mdContent), 0644); err != nil {
		t.Fatalf("failed to write plan file: %v", err)
	}

	// 5. Test `zenfd plan push`
	pushCmd := exec.Command(binPath, "plan", "push", planFile, "--url", daemonURL)
	out, err = pushCmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "Ingested Plan 'e2e-system'") {
		t.Fatalf("plan push failed: %v, out: %s", err, string(out))
	}

	// 6. Test `zenfd plan get e2e-system`
	getCmd := exec.Command(binPath, "plan", "get", "e2e-system", "--url", daemonURL)
	out, err = getCmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "SLICE-01") {
		t.Fatalf("plan get failed: %v, out: %s", err, string(out))
	}

	// 7. Test `zenfd task list`
	listCmd := exec.Command(binPath, "task", "list", "--url", daemonURL)
	out, err = listCmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "SLICE-01") {
		t.Fatalf("task list failed: %v, out: %s", err, string(out))
	}

	t.Log("✓ All Cobra CLI commands (status, plan push, plan get, task list) verified successfully against running daemon")
}
