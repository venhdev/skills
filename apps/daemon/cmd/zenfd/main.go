package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"text/tabwriter"
	"time"

	"zenfd/internal/domain"
	"zenfd/internal/gateway"
	"zenfd/internal/harness"
	zenfdmcp "zenfd/internal/mcp"
	"zenfd/internal/parser"
	"zenfd/internal/scheduler"
	"zenfd/internal/storage"
	"zenfd/internal/supervisor"

	"github.com/spf13/cobra"
)

var (
	defaultPort    int
	defaultDBPath  string
	defaultURL     string
	defaultWorkers int
	defaultTickMs  int
)

func resolveHomePath(path string) string {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(os.TempDir(), "zenforge", "state.db")
		}
		return filepath.Join(home, ".zenforge", "state.db")
	}
	if path == "~" || path == "~/" {
		home, _ := os.UserHomeDir()
		return home
	}
	if len(path) > 2 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "zenfd",
		Short: "ZenForge Daemon (zenfd) - Background Agent Execution & Orchestration Daemon",
		Long: `zenfd is the single-binary control plane for ZenForge.
It manages POSIX process groups, PTY sessions, DAG scheduler execution, 
SQLite WAL state, and provides real-time SSE/WebSocket endpoints on port 1123.`,
	}

	rootCmd.PersistentFlags().StringVar(&defaultURL, "url", "http://127.0.0.1:1123", "Daemon control plane URL")

	// 1. Daemon Command
	daemonCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Start the long-running background orchestrator and gateway server",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := resolveHomePath(defaultDBPath)
			if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
				return fmt.Errorf("failed to create state directory: %w", err)
			}

			fmt.Printf("ZenForge Daemon (zenfd) starting...\n")
			fmt.Printf("  • Port:     %d\n", defaultPort)
			fmt.Printf("  • Database: %s (SQLite WAL)\n", dbPath)
			fmt.Printf("  • Workers:  %d concurrent\n", defaultWorkers)

			// 1. Storage
			db, err := storage.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open storage engine: %w", err)
			}
			defer db.Close()
			store := storage.NewStore(db)

			// 2. Supervisor & Harness Registry
			scratchDir := filepath.Dir(dbPath)
			super := supervisor.NewProcessSupervisor(filepath.Join(scratchDir, "logs"))
			harnessReg := harness.DefaultRegistry()
			super.SetHarnessRegistry(harnessReg)

			// 3. Scheduler
			cfg := scheduler.SchedulerConfig{
				MaxConcurrentWorkers: defaultWorkers,
				HeartbeatTimeout:     30 * time.Second,
				TickInterval:         time.Duration(defaultTickMs) * time.Millisecond,
			}
			sched := scheduler.NewDAGScheduler(cfg, store, super)
			sched.SetHarnessRegistry(harnessReg)

			// 4. Gateway
			gw := gateway.NewServer(store, super, sched)
			sched.SetBroadcaster(gw)

			// 5. Context & Signals
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			// Start scheduler ticker loop
			ticker := time.NewTicker(time.Duration(defaultTickMs) * time.Millisecond)
			defer ticker.Stop()

			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						if err := sched.Tick(ctx); err != nil {
							// Log transient tick errors without crashing
						}
					}
				}
			}()

			// Start HTTP Server
			listenAddr := fmt.Sprintf("127.0.0.1:%d", defaultPort)
			errChan := make(chan error, 1)

			go func() {
				if err := gw.Start(listenAddr); err != nil && err != http.ErrServerClosed {
					errChan <- err
				}
			}()

			fmt.Printf("  ✓ Control Plane active at http://%s\n", listenAddr)
			fmt.Printf("  ✓ Realtime Event Stream at http://%s/api/v1/events/stream\n", listenAddr)

			select {
			case err := <-errChan:
				return fmt.Errorf("gateway error: %w", err)
			case <-ctx.Done():
				fmt.Printf("\nShutdown requested. Draining active runs and flushing WAL...\n")
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = gw.Shutdown(shutdownCtx)
				fmt.Printf("ZenForge Daemon stopped cleanly.\n")
				return nil
			}
		},
	}

	daemonCmd.Flags().IntVarP(&defaultPort, "port", "p", 1123, "Port to bind HTTP/SSE/WebSocket gateway")
	daemonCmd.Flags().StringVarP(&defaultDBPath, "db", "d", "~/.zenforge/state.db", "Path to SQLite WAL database")
	daemonCmd.Flags().IntVarP(&defaultWorkers, "workers", "w", 5, "Maximum concurrent worker slots")
	daemonCmd.Flags().IntVar(&defaultTickMs, "tick", 500, "Scheduler tick interval in milliseconds")
	rootCmd.AddCommand(daemonCmd)

	// 2. Plan Push Command
	var planRepo string
	planPushCmd := &cobra.Command{
		Use:   "push <file.md>",
		Short: "Parse and ingest a living Markdown plan dossier into the daemon",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			repo := planRepo
			if repo == "" {
				cwd, _ := os.Getwd()
				repo = cwd
			}

			plan, tasks, err := parser.ParseMarkdownPlanFile(filePath, repo)
			if err != nil {
				return fmt.Errorf("failed to parse plan dossier %s: %w", filePath, err)
			}

			payload := gateway.CreatePlanRequest{
				Slug:     plan.Slug,
				Title:    plan.Title,
				RepoPath: plan.RepoPath,
				Tasks:    tasks,
			}

			bodyBytes, err := json.Marshal(payload)
			if err != nil {
				return err
			}

			resp, err := http.Post(defaultURL+"/api/v1/plans", "application/json", bytes.NewReader(bodyBytes))
			if err != nil {
				return fmt.Errorf("failed to connect to daemon at %s: %w", defaultURL, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
				return fmt.Errorf("daemon responded with HTTP %d", resp.StatusCode)
			}

			fmt.Printf("✓ Ingested Plan '%s' (%d tasks) into %s\n", plan.Slug, len(tasks), defaultURL)
			for _, t := range tasks {
				fmt.Printf("    [%s] %s (deps: %v)\n", t.Status, t.ID, t.Dependencies)
			}
			return nil
		},
	}
	planPushCmd.Flags().StringVarP(&planRepo, "repo", "r", "", "Repository root path (defaults to CWD)")

	// 3. Plan Get Command
	planGetCmd := &cobra.Command{
		Use:   "get <slug>",
		Short: "Retrieve and view a plan graph from the daemon",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			resp, err := http.Get(fmt.Sprintf("%s/api/v1/plans/%s", defaultURL, slug))
			if err != nil {
				return fmt.Errorf("failed to query plan: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				return fmt.Errorf("plan '%s' not found on %s", slug, defaultURL)
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("daemon returned HTTP %d", resp.StatusCode)
			}

			var result struct {
				Plan  domain.Plan   `json:"plan"`
				Tasks []domain.Task `json:"tasks"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}

			fmt.Printf("Plan: %s (Status: %s)\n", result.Plan.Slug, result.Plan.Status)
			fmt.Printf("Title: %s\n", result.Plan.Title)
			fmt.Printf("Repo:  %s\n\n", result.Plan.RepoPath)

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "TASK ID\tSTATUS\tDEPENDENCIES\tTITLE\n")
			for _, t := range result.Tasks {
				deps := "-"
				if len(t.Dependencies) > 0 {
					deps = fmt.Sprintf("%v", t.Dependencies)
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.ID, t.Status, deps, t.Title)
			}
			return w.Flush()
		},
	}

	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Manage living plan dossiers",
	}
	planCmd.AddCommand(planPushCmd)
	planCmd.AddCommand(planGetCmd)
	rootCmd.AddCommand(planCmd)

	// 4. Task List Command
	var taskStatus string
	var taskRepo string
	taskListCmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks across active plans",
		RunE: func(cmd *cobra.Command, args []string) error {
			url := fmt.Sprintf("%s/api/v1/tasks?status=%s&repo_path=%s", defaultURL, taskStatus, taskRepo)
			resp, err := http.Get(url)
			if err != nil {
				return fmt.Errorf("failed to fetch tasks: %w", err)
			}
			defer resp.Body.Close()

			var tasks []domain.Task
			if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
				return err
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "TASK ID\tPLAN\tSTATUS\tCOMMAND\n")
			for _, t := range tasks {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.ID, t.PlanID, t.Status, t.SkillCommand)
			}
			return w.Flush()
		},
	}
	taskListCmd.Flags().StringVarP(&taskStatus, "status", "s", "", "Filter by task status (e.g. READY, RUNNING)")
	taskListCmd.Flags().StringVarP(&taskRepo, "repo", "r", "", "Filter by repo path")

	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "Query and inspect tasks",
	}
	taskCmd.AddCommand(taskListCmd)

	// Helper for task status transitions via CLI
	sendTaskStatus := func(taskID string, newStatus domain.TaskStatus) error {
		payload := map[string]string{
			"status": string(newStatus),
		}
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		resp, err := http.Post(fmt.Sprintf("%s/api/v1/tasks/%s/status", defaultURL, taskID), "application/json", bytes.NewReader(bodyBytes))
		if err != nil {
			return fmt.Errorf("failed to connect to daemon at %s: %w", defaultURL, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("daemon responded with HTTP %d", resp.StatusCode)
		}

		fmt.Printf("✓ Task '%s' status updated to %s\n", taskID, newStatus)
		return nil
	}

	taskStartCmd := &cobra.Command{
		Use:   "start <task-id>",
		Short: "Mark a task as RUNNING",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return sendTaskStatus(args[0], domain.StatusRunning)
		},
	}
	taskCmd.AddCommand(taskStartCmd)

	taskDoneCmd := &cobra.Command{
		Use:   "done <task-id>",
		Short: "Mark a task as COMPLETED",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return sendTaskStatus(args[0], domain.StatusCompleted)
		},
	}
	taskCmd.AddCommand(taskDoneCmd)

	taskFailCmd := &cobra.Command{
		Use:   "fail <task-id>",
		Short: "Mark a task as FAILED",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return sendTaskStatus(args[0], domain.StatusFailed)
		},
	}
	taskCmd.AddCommand(taskFailCmd)

	var updateStatusStr string
	taskUpdateCmd := &cobra.Command{
		Use:   "update <task-id>",
		Short: "Update task status to a specific state",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if updateStatusStr == "" {
				return fmt.Errorf("--status flag is required")
			}
			st := domain.TaskStatus(updateStatusStr)
			if !st.IsValid() {
				return fmt.Errorf("invalid status '%s'", updateStatusStr)
			}
			return sendTaskStatus(args[0], st)
		},
	}
	taskUpdateCmd.Flags().StringVarP(&updateStatusStr, "status", "s", "", "Target status (e.g. READY, RUNNING, COMPLETED, FAILED)")
	_ = taskUpdateCmd.MarkFlagRequired("status")
	taskCmd.AddCommand(taskUpdateCmd)

	// Task Claim Command
	var claimWorker string
	var claimJSON bool
	taskClaimCmd := &cobra.Command{
		Use:   "claim <task-id>",
		Short: "Claim a ready task and display its context packet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			payload := map[string]any{
				"worker_id": claimWorker,
				"pid":       os.Getpid(),
			}
			bodyBytes, _ := json.Marshal(payload)
			resp, err := http.Post(fmt.Sprintf("%s/api/v1/tasks/%s/claim", defaultURL, taskID), "application/json", bytes.NewReader(bodyBytes))
			if err != nil {
				return fmt.Errorf("failed to connect to daemon: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("claim failed (HTTP %d): %s", resp.StatusCode, string(body))
			}

			var packet domain.ContextPacket
			if err := json.NewDecoder(resp.Body).Decode(&packet); err != nil {
				return err
			}

			if claimJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(packet)
			}

			fmt.Printf("✓ Task '%s' successfully claimed by %s (Status: RUNNING)\n\n", taskID, packet.Task.ClaimedBy)
			fmt.Println(packet.PromptMarkdown)
			return nil
		},
	}
	taskClaimCmd.Flags().StringVarP(&claimWorker, "worker", "w", "", "Worker identifier (defaults to CLI PID)")
	taskClaimCmd.Flags().BoolVar(&claimJSON, "json", false, "Output raw ContextPacket JSON")
	taskCmd.AddCommand(taskClaimCmd)

	// Task Merge Command
	taskMergeCmd := &cobra.Command{
		Use:   "merge <task-id>",
		Short: "Merge an awaiting worktree task into main branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID := args[0]
			resp, err := http.Post(fmt.Sprintf("%s/api/v1/tasks/%s/merge", defaultURL, taskID), "application/json", nil)
			if err != nil {
				return fmt.Errorf("failed to connect to daemon: %w", err)
			}
			defer resp.Body.Close()

			var result struct {
				Merged        bool     `json:"merged"`
				ConflictFiles []string `json:"conflict_files"`
				Status        string   `json:"status"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}

			if !result.Merged {
				fmt.Printf("✗ Merge conflict encountered for task %s (Status: CONFLICT)\n", taskID)
				fmt.Printf("Conflicts in:\n")
				for _, f := range result.ConflictFiles {
					fmt.Printf("  • %s\n", f)
				}
				return fmt.Errorf("merge conflict in %d files", len(result.ConflictFiles))
			}

			fmt.Printf("✓ Task '%s' cleanly merged and completed (Status: COMPLETED)\n", taskID)
			return nil
		},
	}
	taskCmd.AddCommand(taskMergeCmd)

	rootCmd.AddCommand(taskCmd)

	// 5. Run Kill Command
	runKillCmd := &cobra.Command{
		Use:   "kill <run-id>",
		Short: "Immediately terminate an active agent run cascade using -PGID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			runID := args[0]
			resp, err := http.Post(fmt.Sprintf("%s/api/v1/runs/%s/kill", defaultURL, runID), "application/json", nil)
			if err != nil {
				return fmt.Errorf("failed to issue kill request: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("kill failed with HTTP %d", resp.StatusCode)
			}

			fmt.Printf("✓ Process group for run %s terminated (-PGID cascade kill executed)\n", runID)
			return nil
		},
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Inspect and control task runs",
	}
	runCmd.AddCommand(runKillCmd)
	rootCmd.AddCommand(runCmd)

	// 6. Status Command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check the health and connection status of the ZenForge Daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := http.Get(defaultURL + "/api/v1/tasks")
			if err != nil {
				fmt.Printf("✗ ZenForge Daemon is NOT running at %s\n", defaultURL)
				return nil
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				fmt.Printf("✓ ZenForge Daemon is healthy and active at %s\n", defaultURL)
			} else {
				fmt.Printf("⚠ ZenForge Daemon returned HTTP %d at %s\n", resp.StatusCode, defaultURL)
			}
			return nil
		},
	}
	rootCmd.AddCommand(statusCmd)

	// MCP Stdio Server Command
	var mcpDBPath string
	var mcpRepoPath string
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start the stdio Model Context Protocol (MCP) server for agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := resolveHomePath(mcpDBPath)
			db, err := storage.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open storage engine: %w", err)
			}
			defer db.Close()
			store := storage.NewStore(db)

			repo := mcpRepoPath
			if repo == "" {
				repo, _ = os.Getwd()
			}

			srv := zenfdmcp.NewServer(store, repo)
			return srv.ServeStdio()
		},
	}
	mcpCmd.Flags().StringVarP(&mcpDBPath, "db", "d", "~/.zenforge/state.db", "Path to SQLite WAL database")
	mcpCmd.Flags().StringVarP(&mcpRepoPath, "repo", "r", "", "Repository root path (defaults to CWD)")
	rootCmd.AddCommand(mcpCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
