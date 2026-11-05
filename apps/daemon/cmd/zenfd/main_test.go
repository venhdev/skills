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

	// 4. Register the owning project
	projectPayload := fmt.Sprintf(`{"slug":"e2e-project","name":"E2E Project","root_path":%q}`, tempDir)
	projectResp, err := http.Post(daemonURL+"/api/v1/projects", "application/json", strings.NewReader(projectPayload))
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}
	projectResp.Body.Close()
	if projectResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 creating project, got %d", projectResp.StatusCode)
	}

	// 5. Create dummy markdown plan
	planFile := filepath.Join(tempDir, "plan.md")
	mdContent := `
# Plan Dossier: e2e-system · ACTIVE

| Task ID | Description | Dependencies | Status | Harness |
| :--- | :--- | :--- | :--- | :--- |
| **SLICE-01** | Root Slice | - | **READY** | orchestrator |
| **SLICE-02** | Dependent Slice | SLICE-01 | **PLANNED** | orchestrator |
`
	if err := os.WriteFile(planFile, []byte(mdContent), 0644); err != nil {
		t.Fatalf("failed to write plan file: %v", err)
	}

	// 6. Test `zenfd plan push`
	pushCmd := exec.Command(binPath, "plan", "push", planFile, "--project", "e2e-project", "--url", daemonURL)
	out, err = pushCmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "Ingested Plan 'e2e-system'") {
		t.Fatalf("plan push failed: %v, out: %s", err, string(out))
	}

	// 7. Test `zenfd plan get e2e-system`
	getCmd := exec.Command(binPath, "plan", "get", "e2e-system", "--url", daemonURL)
	out, err = getCmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "SLICE-01") {
		t.Fatalf("plan get failed: %v, out: %s", err, string(out))
	}

	// 8. Test `zenfd task list`
	listCmd := exec.Command(binPath, "task", "list", "--url", daemonURL)
	out, err = listCmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "SLICE-01") {
		t.Fatalf("task list failed: %v, out: %s", err, string(out))
	}

	// 9. Test `zenfd task start SLICE-01`
	startCmd := exec.Command(binPath, "task", "start", "SLICE-01", "--url", daemonURL)
	out, err = startCmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "status updated to RUNNING") {
		t.Fatalf("task start failed: %v, out: %s", err, string(out))
	}

	// 10. Test `zenfd task done SLICE-01`
	doneCmd := exec.Command(binPath, "task", "done", "SLICE-01", "--url", daemonURL)
	out, err = doneCmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "status updated to COMPLETED") {
		t.Fatalf("task done failed: %v, out: %s", err, string(out))
	}

	// 11. Test `zenfd task update SLICE-02 --status READY`
	updateCmd := exec.Command(binPath, "task", "update", "SLICE-02", "--status", "READY", "--url", daemonURL)
	out, err = updateCmd.CombinedOutput()
	if err != nil || !strings.Contains(string(out), "status updated to READY") {
		t.Fatalf("task update failed: %v, out: %s", err, string(out))
	}

	t.Log("✓ All Cobra CLI commands (status, plan push, plan get, task list, task start/done/update) verified successfully against running daemon")
}
