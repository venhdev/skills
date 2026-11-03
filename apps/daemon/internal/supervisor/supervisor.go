package supervisor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"zenfd/internal/domain"
	"zenfd/internal/harness"
)

var (
	ErrRunNotFound    = errors.New("active run not found")
	ErrAlreadyRunning = errors.New("run is already running")
	ErrTimeout        = errors.New("run wait timeout")
)

type activeProcess struct {
	mu      sync.RWMutex
	run     *domain.Run
	cmd     *exec.Cmd
	ptyFile *os.File
	pgid    int
	done    chan struct{}
}

// ProcessSupervisor implements domain.Supervisor with strict process group isolation.
type ProcessSupervisor struct {
	mu              sync.RWMutex
	processes       map[string]*activeProcess
	defaultDir      string
	harnessRegistry *harness.Registry
}

// NewProcessSupervisor creates a new supervisor storing logs in the specified directory.
func NewProcessSupervisor(defaultLogDir string) *ProcessSupervisor {
	return &ProcessSupervisor{
		processes:       make(map[string]*activeProcess),
		defaultDir:      defaultLogDir,
		harnessRegistry: harness.DefaultRegistry(),
	}
}

// SetHarnessRegistry configures the harness adapter registry.
func (s *ProcessSupervisor) SetHarnessRegistry(reg *harness.Registry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.harnessRegistry = reg
}

func (s *ProcessSupervisor) buildCommand(task domain.Task) (*exec.Cmd, bool, error) {
	if s.harnessRegistry != nil {
		adapter, err := s.harnessRegistry.Resolve(task)
		if err == nil {
			spec, err := adapter.BuildExecution(task)
			if err == nil {
				if spec.IsExternal {
					return nil, true, nil
				}
				if spec.Binary != "" {
					return exec.Command(spec.Binary, spec.Args...), false, nil
				}
			}
		}
	}

	cmdStr := strings.TrimSpace(task.SkillCommand)
	if cmdStr == "" {
		return exec.Command("bash"), false, nil
	}
	return exec.Command("bash", "-c", cmdStr), false, nil
}

// SpawnHeadless launches an agent runner using stdio pipes (non-interactive).
func (s *ProcessSupervisor) SpawnHeadless(ctx context.Context, task domain.Task, envOverrides map[string]string) (*domain.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	runID := fmt.Sprintf("run-%s-%d", task.ID, time.Now().UnixNano())
	if _, exists := s.processes[runID]; exists {
		return nil, ErrAlreadyRunning
	}

	cmd, isExt, err := s.buildCommand(task)
	if err != nil {
		return nil, err
	}
	if isExt {
		return nil, errors.New("cannot spawn headless process for external harness")
	}
	if task.WorktreePath != "" {
		cmd.Dir = task.WorktreePath
	}

	cmd.Env = os.Environ()
	for k, v := range envOverrides {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Enforce independent process group for headless execution
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	logPath := filepath.Join(s.defaultDir, fmt.Sprintf("%s.log", runID))
	_ = os.MkdirAll(filepath.Dir(logPath), 0755)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("failed to start headless process: %w", err)
	}

	pid := cmd.Process.Pid
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		pgid = pid
	}

	now := time.Now().UTC()
	run := &domain.Run{
		ID:         runID,
		TaskID:     task.ID,
		RepoPath:   task.RepoPath,
		PID:        pid,
		PGID:       pgid,
		Status:     domain.StatusRunning,
		LogPath:    logPath,
		ExitCode:   -1,
		StartedAt:  now,
		LastBeatAt: now,
	}

	proc := &activeProcess{
		run:  run,
		cmd:  cmd,
		pgid: pgid,
		done: make(chan struct{}),
	}
	s.processes[runID] = proc

	runCopy := *run

	// Background wait to reap process upon natural exit
	go func() {
		defer close(proc.done)
		defer logFile.Close()

		waitErr := cmd.Wait()

		proc.mu.Lock()
		defer proc.mu.Unlock()

		finishedAt := time.Now().UTC()
		proc.run.FinishedAt = &finishedAt

		if waitErr != nil {
			var exitErr *exec.ExitError
			if errors.As(waitErr, &exitErr) {
				proc.run.ExitCode = exitErr.ExitCode()
			} else {
				proc.run.ExitCode = 1
			}
			proc.run.Status = domain.StatusFailed
		} else {
			proc.run.ExitCode = 0
			proc.run.Status = domain.StatusCompleted
		}
	}()

	return &runCopy, nil
}

// SpawnPTY allocates a pseudo-terminal and launches an interactive agent runner.
func (s *ProcessSupervisor) SpawnPTY(ctx context.Context, task domain.Task, win domain.WindowSize, envOverrides map[string]string) (*domain.Run, io.ReadWriteCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	runID := fmt.Sprintf("run-%s-%d", task.ID, time.Now().UnixNano())
	if _, exists := s.processes[runID]; exists {
		return nil, nil, ErrAlreadyRunning
	}

	cmd, isExt, err := s.buildCommand(task)
	if err != nil {
		return nil, nil, err
	}
	if isExt {
		return nil, nil, errors.New("cannot spawn PTY process for external harness")
	}
	if task.WorktreePath != "" {
		cmd.Dir = task.WorktreePath
	}

	cmd.Env = os.Environ()
	for k, v := range envOverrides {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// pty.Start automatically creates a new session (setsid) making PGID == PID
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start pty process: %w", err)
	}

	if win.Rows > 0 && win.Cols > 0 {
		_ = pty.Setsize(ptyFile, &pty.Winsize{Rows: win.Rows, Cols: win.Cols})
	}

	pid := cmd.Process.Pid
	pgid, err := syscall.Getpgid(pid)
	if err != nil {
		pgid = pid
	}

	now := time.Now().UTC()
	logPath := filepath.Join(s.defaultDir, fmt.Sprintf("%s.pty.log", runID))

	run := &domain.Run{
		ID:         runID,
		TaskID:     task.ID,
		RepoPath:   task.RepoPath,
		PID:        pid,
		PGID:       pgid,
		Status:     domain.StatusRunning,
		LogPath:    logPath,
		ExitCode:   -1,
		StartedAt:  now,
		LastBeatAt: now,
	}

	proc := &activeProcess{
		run:     run,
		cmd:     cmd,
		ptyFile: ptyFile,
		pgid:    pgid,
		done:    make(chan struct{}),
	}
	s.processes[runID] = proc

	runCopy := *run

	go func() {
		defer close(proc.done)
		defer ptyFile.Close()

		waitErr := cmd.Wait()

		proc.mu.Lock()
		defer proc.mu.Unlock()

		finishedAt := time.Now().UTC()
		proc.run.FinishedAt = &finishedAt

		if waitErr != nil {
			var exitErr *exec.ExitError
			if errors.As(waitErr, &exitErr) {
				proc.run.ExitCode = exitErr.ExitCode()
			} else {
				proc.run.ExitCode = 1
			}
			proc.run.Status = domain.StatusFailed
		} else {
			proc.run.ExitCode = 0
			proc.run.Status = domain.StatusCompleted
		}
	}()

	return &runCopy, ptyFile, nil
}

// GetRun returns a thread-safe snapshot copy of the Run status.
func (s *ProcessSupervisor) GetRun(runID string) (*domain.Run, error) {
	s.mu.RLock()
	proc, exists := s.processes[runID]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrRunNotFound
	}

	proc.mu.RLock()
	defer proc.mu.RUnlock()

	runCopy := *proc.run
	return &runCopy, nil
}

// Wait blocks until the process finishes or timeout expires.
func (s *ProcessSupervisor) Wait(runID string, timeout time.Duration) (*domain.Run, error) {
	s.mu.RLock()
	proc, exists := s.processes[runID]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrRunNotFound
	}

	select {
	case <-proc.done:
		return s.GetRun(runID)
	case <-time.After(timeout):
		return nil, ErrTimeout
	}
}

// Kill terminates the entire process group using -PGID (SIGTERM then SIGKILL escalation).
func (s *ProcessSupervisor) Kill(runID string, gracePeriod time.Duration) error {
	s.mu.RLock()
	proc, exists := s.processes[runID]
	s.mu.RUnlock()

	if !exists {
		return ErrRunNotFound
	}

	pgid := proc.pgid

	// 1. Send SIGTERM to -PGID
	_ = syscall.Kill(-pgid, syscall.SIGTERM)

	// 2. Wait up to gracePeriod for process group leader to exit
	select {
	case <-proc.done:
	case <-time.After(gracePeriod):
		// 3. Escalate to SIGKILL if still alive
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		select {
		case <-proc.done:
		case <-time.After(500 * time.Millisecond):
		}
	}

	// 4. Polling wait for kernel scheduler to finish reaping all descendants
	for i := 0; i < 50; i++ {
		err := syscall.Kill(-pgid, 0)
		if err != nil { // ESRCH: process group no longer exists
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

// ResizePTY sends a window size update to the active PTY.
func (s *ProcessSupervisor) ResizePTY(runID string, win domain.WindowSize) error {
	s.mu.RLock()
	proc, exists := s.processes[runID]
	s.mu.RUnlock()

	if !exists || proc.ptyFile == nil {
		return ErrRunNotFound
	}

	return pty.Setsize(proc.ptyFile, &pty.Winsize{
		Rows: win.Rows,
		Cols: win.Cols,
	})
}
