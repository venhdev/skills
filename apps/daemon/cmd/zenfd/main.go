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
	zenfdmcp "zenfd/internal/mcp"
	"zenfd/internal/parser"
	"zenfd/internal/storage"

	"github.com/spf13/cobra"
)

var (
	defaultPort   int
	defaultDBPath string
	defaultURL    string
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

const Version = "0.1.3"

func main() {
	rootCmd := &cobra.Command{
		Use:     "zenfd",
		Version: Version,
		Short:   "ZenForge Daemon (zenfd) - Background Agent Execution & Orchestration Daemon",
		Long: `zenfd is the single-binary control plane for ZenForge.
It owns the SQLite WAL tracking plane, exposes an MCP stdio surface for agents,
and serves a read-only cockpit with real-time SSE updates on port 1123.`,
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

			fmt.Printf("ZenForge Daemon (zenfd) v%s starting...\n", Version)
			fmt.Printf("  • Port:     %d\n", defaultPort)
			fmt.Printf("  • Database: %s (SQLite WAL)\n", dbPath)

			// 1. Storage
			db, err := storage.Open(dbPath)
			if err != nil {
				return fmt.Errorf("failed to open storage engine: %w", err)
			}
			defer db.Close()
			store := storage.NewStore(db)

			// 2. Gateway (read-only viewer + public data API + SSE)
			gw := gateway.NewServer(store)

			// 4. Context & Signals
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

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
				fmt.Printf("\nShutdown requested. Flushing WAL and closing the gateway...\n")
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = gw.Shutdown(shutdownCtx)
				fmt.Printf("ZenForge Daemon stopped cleanly.\n")
				return nil
			}
		},
	}

	daemonCmd.Flags().IntVarP(&defaultPort, "port", "p", 1123, "Port to bind the HTTP/SSE gateway")
	daemonCmd.Flags().StringVarP(&defaultDBPath, "db", "d", "~/.zenforge/state.db", "Path to SQLite WAL database")
	rootCmd.AddCommand(daemonCmd)

	// 2. Plan Push Command
	planPushCmd := &cobra.Command{
		Use:   "push <file.md>",
		Short: "Parse and ingest a living Markdown plan dossier into the daemon",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]

			plan, tasks, err := parser.ParseMarkdownPlanFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to parse plan dossier %s: %w", filePath, err)
			}

			payload := gateway.CreatePlanRequest{
				Slug:  plan.Slug,
				Title: plan.Title,
				Tasks: tasks,
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
			fmt.Printf("Title: %s\n\n", result.Plan.Title)

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
	taskListCmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks across active plans",
		RunE: func(cmd *cobra.Command, args []string) error {
			url := fmt.Sprintf("%s/api/v1/tasks?status=%s", defaultURL, taskStatus)
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
			fmt.Fprintf(w, "TASK ID\tPLAN\tSTATUS\tHARNESS\n")
			for _, t := range tasks {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.ID, t.PlanID, t.Status, t.Harness)
			}
			return w.Flush()
		},
	}
	taskListCmd.Flags().StringVarP(&taskStatus, "status", "s", "", "Filter by task status (e.g. READY, RUNNING)")

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
			// The CLI is not the worker: it prints the packet and exits, so it
			// never records a process identity on the task.
			payload := map[string]any{
				"worker_id": claimWorker,
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
	taskClaimCmd.Flags().StringVarP(&claimWorker, "worker", "w", "", "Worker identifier (defaults to a generated id)")
	taskClaimCmd.Flags().BoolVar(&claimJSON, "json", false, "Output raw ContextPacket JSON")
	taskCmd.AddCommand(taskClaimCmd)

	rootCmd.AddCommand(taskCmd)

	// 5. Status Command
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

			srv := zenfdmcp.NewServer(store)
			return srv.ServeStdio()
		},
	}
	mcpCmd.Flags().StringVarP(&mcpDBPath, "db", "d", "~/.zenforge/state.db", "Path to SQLite WAL database")
	rootCmd.AddCommand(mcpCmd)

	// Subcommands for MCP: check and setup
	var checkClient string
	var checkConfig string
	var checkJSON bool
	mcpCheckCmd := &cobra.Command{
		Use:   "check",
		Short: "Audit client MCP configuration for zenfd",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := zenfdmcp.CheckMCP(zenfdmcp.ClientType(checkClient), checkConfig)
			if err != nil {
				return fmt.Errorf("mcp check failed: %w", err)
			}
			if checkJSON {
				byt, _ := json.MarshalIndent(res, "", "  ")
				fmt.Println(string(byt))
				return nil
			}
			if res.Configured && res.BinaryExists {
				fmt.Printf("[OK] zenfd MCP is configured and valid in %s (binary: %s)\n", res.ConfigPath, res.BinaryPath)
				return nil
			}
			fmt.Printf("[NOT_CONFIGURED] %s\n", res.StatusMessage)
			return nil
		},
	}
	mcpCheckCmd.Flags().StringVarP(&checkClient, "client", "c", "agy", "Target AI client: agy (default), claude, cursor, opencode")
	mcpCheckCmd.Flags().StringVar(&checkConfig, "config", "", "Override client config file path")
	mcpCheckCmd.Flags().BoolVar(&checkJSON, "json", false, "Output result as JSON")
	mcpCmd.AddCommand(mcpCheckCmd)

	var setupClient string
	var setupConfig string
	var setupDB string
	var setupBinary string
	var setupForce bool
	var setupDryRun bool
	mcpSetupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure zenfd MCP server in client configuration (default: agy)",
		RunE: func(cmd *cobra.Command, args []string) error {
			res, err := zenfdmcp.SetupMCP(zenfdmcp.SetupOptions{
				Client:     zenfdmcp.ClientType(setupClient),
				ConfigPath: setupConfig,
				DBPath:     setupDB,
				BinaryPath: setupBinary,
				Force:      setupForce,
				DryRun:     setupDryRun,
			})
			if err != nil {
				return fmt.Errorf("mcp setup failed: %w", err)
			}

			if res.Action == "dry_run" {
				fmt.Printf("[DRY-RUN] Previewing configuration for %s at %s:\n\n%s\n", res.Client, res.ConfigPath, res.WrittenJSON)
				return nil
			}

			if res.Action == "already_configured" {
				fmt.Printf("[ALREADY_CONFIGURED] zenfd MCP is already configured in %s (binary: %s)\n", res.ConfigPath, res.BinaryPath)
				return nil
			}

			fmt.Printf("[SUCCESS] Configured zenfd MCP server for %s in %s\n", res.Client, res.ConfigPath)
			if res.BackupPath != "" {
				fmt.Printf("Backup saved to: %s\n", res.BackupPath)
			}
			fmt.Printf("Binary: %s\n", res.BinaryPath)
			fmt.Printf("Database: %s\n", res.DBPath)
			fmt.Println("Please restart your AI client session (e.g. opencode or agy) to load the zenfd MCP tools.")
			return nil
		},
	}
	mcpSetupCmd.Flags().StringVarP(&setupClient, "client", "c", "agy", "Target AI client: agy (default), claude, cursor, opencode")
	mcpSetupCmd.Flags().StringVar(&setupConfig, "config", "", "Override client config file path")
	mcpSetupCmd.Flags().StringVarP(&setupDB, "db", "d", "~/.zenforge/state.db", "Path to SQLite WAL database")
	mcpSetupCmd.Flags().StringVarP(&setupBinary, "binary", "b", "", "Path to zenfd binary (auto-detected if omitted)")
	mcpSetupCmd.Flags().BoolVarP(&setupForce, "force", "f", false, "Force overwrite existing entry")
	mcpSetupCmd.Flags().BoolVar(&setupDryRun, "dry-run", false, "Preview configuration changes without writing")
	mcpCmd.AddCommand(mcpSetupCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
