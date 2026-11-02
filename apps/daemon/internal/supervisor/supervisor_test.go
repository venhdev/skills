package supervisor_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"zenfd/internal/domain"
	"zenfd/internal/supervisor"
)

func TestSupervisor_SpawnHeadless(t *testing.T) {
	tempLogDir := t.TempDir()
	sup := supervisor.NewProcessSupervisor(tempLogDir)
	ctx := context.Background()

	task := domain.Task{
		ID:           "test-task-headless",
		SkillCommand: "echo 'HEADLESS_TRACER_OK'; exit 0",
	}

	run, err := sup.SpawnHeadless(ctx, task, nil)
	if err != nil {
		t.Fatalf("SpawnHeadless failed: %v", err)
	}

	if run.PID <= 0 || run.PGID <= 0 {
		t.Errorf("Invalid PID/PGID: PID=%d, PGID=%d", run.PID, run.PGID)
	}

	// Wait for completion using thread-safe Wait method
	finishedRun, err := sup.Wait(run.ID, 2*time.Second)
	if err != nil {
		t.Fatalf("Wait failed: %v", err)
	}

	if finishedRun.Status != domain.StatusCompleted {
		t.Errorf("Expected status COMPLETED, got: %s", finishedRun.Status)
	}
	if finishedRun.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got: %d", finishedRun.ExitCode)
	}

	// Verify log file content
	logBytes, err := os.ReadFile(finishedRun.LogPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}
	if !strings.Contains(string(logBytes), "HEADLESS_TRACER_OK") {
		t.Errorf("Expected log to contain HEADLESS_TRACER_OK, got: %s", string(logBytes))
	}
}

func TestSupervisor_SpawnPTYAndResize(t *testing.T) {
	tempLogDir := t.TempDir()
	sup := supervisor.NewProcessSupervisor(tempLogDir)
	ctx := context.Background()

	task := domain.Task{
		ID:           "test-task-pty",
		SkillCommand: "echo 'PTY_TRACER_OK'; sleep 0.05; exit 0",
	}

	win := domain.WindowSize{Rows: 24, Cols: 80}
	run, ptyFile, err := sup.SpawnPTY(ctx, task, win, nil)
	if err != nil {
		t.Fatalf("SpawnPTY failed: %v", err)
	}
	defer ptyFile.Close()

	// Test Resize
	if err := sup.ResizePTY(run.ID, domain.WindowSize{Rows: 40, Cols: 120}); err != nil {
		t.Errorf("ResizePTY failed: %v", err)
	}

	// Read PTY output
	var outputBuf bytes.Buffer
	buf := make([]byte, 1024)
	for {
		n, rErr := ptyFile.Read(buf)
		if n > 0 {
			outputBuf.Write(buf[:n])
		}
		if rErr != nil {
			if errors.Is(rErr, io.EOF) || errors.Is(rErr, syscall.EIO) || strings.Contains(rErr.Error(), "input/output error") {
				break
			}
			t.Fatalf("PTY read error: %v", rErr)
		}
	}

	if !strings.Contains(outputBuf.String(), "PTY_TRACER_OK") {
		t.Errorf("Expected PTY output to contain PTY_TRACER_OK, got: %s", outputBuf.String())
	}
}

func TestSupervisor_CascadeKill(t *testing.T) {
	tempLogDir := t.TempDir()
	sup := supervisor.NewProcessSupervisor(tempLogDir)
	ctx := context.Background()

	// Launch parent that creates background grandchildren
	task := domain.Task{
		ID:           "test-task-kill",
		SkillCommand: "sleep 100 & sleep 100 & wait",
	}

	run, err := sup.SpawnHeadless(ctx, task, nil)
	if err != nil {
		t.Fatalf("SpawnHeadless failed: %v", err)
	}

	pgid := run.PGID
	time.Sleep(50 * time.Millisecond)

	// Terminate via supervisor Kill
	if err := sup.Kill(run.ID, 50*time.Millisecond); err != nil {
		t.Fatalf("Kill failed: %v", err)
	}

	// Verify all grandchildren are dead
	err = syscall.Kill(-pgid, 0)
	if err == nil {
		t.Errorf("Process group %d is still alive after Kill!", pgid)
	}
}
