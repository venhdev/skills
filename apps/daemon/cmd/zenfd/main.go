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

const Version = "0.2.1"

func main() {
	rootCmd := &cobra.Command{
		Use:     "zenfd",
		Version: Version,
		Short:   "ZenForge Daemon (zenfd) - Background Agent Execution & Orchestration Daemon",
		Long: `zenfd is the single-binary control plane for ZenForge.
It owns the SQLite WAL tracking plane, exposes an MCP stdio surface for agents,
and serves a read-only cockpit with real-time SSE updates on port 1123.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Check if daemon is active
			statusStr := "○ Stopped"
			client := &http.Client{Timeout: 1 * time.Second}
			resp, err := client.Get(defaultURL + "/api/v1/projects")
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					statusStr = fmt.Sprintf("● Active (%s)", defaultURL)
				}
			}

			fmt.Printf("ZenForge Control Plane (zenfd v%s)\n", Version)
			fmt.Printf("Daemon Status: %s\n\n", statusStr)
			fmt.Println("Quickstart Commands:")
			fmt.Println("  zenfd daemon        Start background orchestrator & web cockpit (:1123)")
			fmt.Println("  zenfd status        View detailed status of daemon & all AI MCP clients")
			fmt.Println("  zenfd mcp           Audit AI client MCP setups and display MCP actions")
			fmt.Println("  zenfd task list     List current tasks across active projects")
			fmt.Println("\nRun 'zenfd --help' for the complete command reference.")
		},
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
	var planProject string
	planPushCmd := &cobra.Command{
		Use:   "push <file.md>",
		Short: "Parse and ingest a living Markdown plan dossier into the daemon",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			if planProject == "" {
				return fmt.Errorf("--project is required; plans always belong to a project")
			}

			plan, tasks, err := parser.ParseMarkdownPlanFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to parse plan dossier %s: %w", filePath, err)
			}

			payload := gateway.CreatePlanRequest{
				Slug:        plan.Slug,
				ProjectSlug: planProject,
				Title:       plan.Title,
				Tasks:       tasks,
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
	planPushCmd.Flags().StringVarP(&planProject, "project", "p", "", "Owning project slug (required)")
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

			var payload struct {
				Tasks []domain.Task `json:"tasks"`
			}
			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			if err := json.Unmarshal(bodyBytes, &payload); err != nil {
				// Fallback to raw slice if connected to legacy daemon
				var rawTasks []domain.Task
				if err2 := json.Unmarshal(bodyBytes, &rawTasks); err2 != nil {
					return fmt.Errorf("failed to decode tasks list: %w", err)
				}
				payload.Tasks = rawTasks
			}
			tasks := payload.Tasks

			if len(tasks) == 0 {
				fmt.Println("No tasks found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "TASK ID\tSCOPE\tSTATUS\tHARNESS\n")
			for _, t := range tasks {
				scope := t.PlanSlug
				if scope == "" {
					scope = t.ProjectSlug
				}
				if scope == "" {
					scope = "orphan"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.ID, scope, t.Status, t.Harness)
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
	var statusJSON bool
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Check the health, metrics, and MCP configuration status across all AI clients",
		RunE: func(cmd *cobra.Command, args []string) error {
			type ClientStatus struct {
				Client        string `json:"client"`
				ConfigPath    string `json:"config_path"`
				Configured    bool   `json:"configured"`
				BinaryValid   bool   `json:"binary_valid"`
				StatusMessage string `json:"status_message"`
			}
			type DaemonStatusReport struct {
				Active     bool           `json:"active"`
				GatewayURL string         `json:"gateway_url"`
				StatusCode int            `json:"status_code,omitempty"`
				Projects   int            `json:"projects,omitempty"`
				Plans      int            `json:"plans,omitempty"`
				Tasks      int            `json:"tasks,omitempty"`
				TasksBySt  map[string]int `json:"tasks_by_status,omitempty"`
				Clients    []ClientStatus `json:"clients"`
			}

			report := DaemonStatusReport{
				GatewayURL: defaultURL,
				TasksBySt:  make(map[string]int),
			}

			// 1. Probe Daemon Control Plane
			client := &http.Client{Timeout: 1500 * time.Millisecond}
			resp, err := client.Get(defaultURL + "/api/v1/tasks")
			if err == nil {
				defer resp.Body.Close()
				report.StatusCode = resp.StatusCode
				if resp.StatusCode == http.StatusOK {
					report.Active = true
					var payload struct {
						Tasks []domain.Task `json:"tasks"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil {
						report.Tasks = len(payload.Tasks)
						for _, t := range payload.Tasks {
							report.TasksBySt[string(t.Status)]++
						}
					}

					// Query projects count
					if pResp, pErr := client.Get(defaultURL + "/api/v1/projects"); pErr == nil {
						defer pResp.Body.Close()
						var pPayload struct {
							Projects []domain.Project `json:"projects"`
						}
						if err := json.NewDecoder(pResp.Body).Decode(&pPayload); err == nil {
							report.Projects = len(pPayload.Projects)
						}
					}

					// Query plans count
					if plResp, plErr := client.Get(defaultURL + "/api/v1/plans"); plErr == nil {
						defer plResp.Body.Close()
						var plPayload struct {
							Plans []domain.Plan `json:"plans"`
						}
						if err := json.NewDecoder(plResp.Body).Decode(&plPayload); err == nil {
							report.Plans = len(plPayload.Plans)
						}
					}
				}
			}

			// 2. Audit MCP configuration for all supported clients
			supportedClients := []zenfdmcp.ClientType{
				zenfdmcp.ClientAgy,
				zenfdmcp.ClientClaude,
				zenfdmcp.ClientCursor,
				zenfdmcp.ClientOpenCode,
			}

			for _, c := range supportedClients {
				checkRes, _ := zenfdmcp.CheckMCP(c, "")
				cs := ClientStatus{
					Client: string(c),
				}
				if checkRes != nil {
					cs.ConfigPath = checkRes.ConfigPath
					cs.Configured = checkRes.Configured
					cs.BinaryValid = checkRes.BinaryExists
					cs.StatusMessage = checkRes.StatusMessage
				}
				report.Clients = append(report.Clients, cs)
			}

			if statusJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(report)
			}

			// Text Output
			fmt.Printf("ZenForge Control Plane Status (zenfd v%s)\n", Version)
			fmt.Println("────────────────────────────────────────────────────────────────────────")
			fmt.Println("DAEMON & GATEWAY RUNTIME")
			if report.Active {
				fmt.Printf(" • Status       : ● ACTIVE (Serving at %s)\n", report.GatewayURL)
				fmt.Printf(" • Cockpit UI   : %s (Web Dashboard)\n", report.GatewayURL)
				fmt.Printf(" • Database     : %s (SQLite WAL)\n", resolveHomePath(defaultDBPath))
				fmt.Printf(" • Metrics      : %d Projects | %d Plans | %d Tasks", report.Projects, report.Plans, report.Tasks)
				if report.Tasks > 0 {
					fmt.Printf(" (%d Ready, %d Running, %d Done)",
						report.TasksBySt[string(domain.StatusReady)],
						report.TasksBySt[string(domain.StatusRunning)],
						report.TasksBySt[string(domain.StatusCompleted)])
				}
				fmt.Println()
			} else {
				fmt.Printf(" • Status       : ○ INACTIVE (Daemon not running)\n")
				fmt.Printf(" • Gateway URL  : %s\n", report.GatewayURL)
				fmt.Println(" • Quickstart   : Run 'zenfd daemon' to start orchestrator and web UI")
			}

			fmt.Println("\nAI CLIENT MCP INTEGRATIONS")
			for _, cs := range report.Clients {
				var badge string
				if cs.Configured && cs.BinaryValid {
					badge = "✔ CONFIGURED    "
				} else if cs.Configured && !cs.BinaryValid {
					badge = "⚠ BROKEN_BINARY "
				} else {
					badge = "○ NOT CONFIGURED"
				}
				fmt.Printf(" • %-14s : %s (%s)\n", cs.Client, badge, cs.ConfigPath)
			}
			fmt.Println("\n💡 To configure an AI client: run 'zenfd mcp setup --client <name>'")
			fmt.Println("────────────────────────────────────────────────────────────────────────")
			return nil
		},
	}
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output status report as JSON")
	rootCmd.AddCommand(statusCmd)

	// MCP Stdio Server Command
	var mcpDBPath string
	var mcpStdioForce bool
	mcpCmd := &cobra.Command{
		Use:   "mcp [command]",
		Short: "Manage or run Model Context Protocol (MCP) server for agents",
		Long: `The 'zenfd mcp' command manages MCP integrations for AI agents.
When invoked directly by an AI client without a TTY (via stdin/stdout pipe), it automatically starts the JSON-RPC stdio server.
When invoked in an interactive terminal, it displays the integration status and available actions.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if stdin is a character device (interactive terminal)
			isTerminal := false
			if fileInfo, err := os.Stdin.Stat(); err == nil {
				isTerminal = (fileInfo.Mode() & os.ModeCharDevice) != 0
			}

			// If called directly in terminal and not forced via --stdio:
			// Show status overview + actions help rather than hanging on stdio.
			if isTerminal && !mcpStdioForce {
				fmt.Printf("ZenForge MCP Integrations (zenfd v%s)\n", Version)
				fmt.Println("────────────────────────────────────────────────────────────────────────")
				fmt.Println("AI CLIENT MCP STATUS:")

				supportedClients := []zenfdmcp.ClientType{
					zenfdmcp.ClientAgy,
					zenfdmcp.ClientClaude,
					zenfdmcp.ClientCursor,
					zenfdmcp.ClientOpenCode,
				}
				for _, c := range supportedClients {
					checkRes, _ := zenfdmcp.CheckMCP(c, "")
					var badge string
					var cfgPath string
					if checkRes != nil {
						cfgPath = checkRes.ConfigPath
						if checkRes.Configured && checkRes.BinaryExists {
							badge = "✔ CONFIGURED    "
						} else if checkRes.Configured && !checkRes.BinaryExists {
							badge = "⚠ BROKEN_BINARY "
						} else {
							badge = "○ NOT CONFIGURED"
						}
					}
					fmt.Printf(" • %-14s : %s (%s)\n", c, badge, cfgPath)
				}

				fmt.Println("\nAVAILABLE MCP ACTIONS:")
				fmt.Println("  zenfd mcp setup --client <name>   Configure MCP server for AI client (agy, claude, cursor, opencode)")
				fmt.Println("  zenfd mcp check --client <name>   Audit detailed client configuration and binary validity")
				fmt.Println("  zenfd mcp --stdio                 Force start JSON-RPC stdio server (for manual piping)")
				fmt.Println("\nℹ When an AI client spawns 'zenfd mcp', stdio JSON-RPC runs automatically.")
				fmt.Println("────────────────────────────────────────────────────────────────────────")
				return nil
			}

			// Non-interactive mode (spawned by AI Agent via pipe) or --stdio flag
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
	mcpCmd.Flags().BoolVar(&mcpStdioForce, "stdio", false, "Force run stdio JSON-RPC server even if attached to terminal")
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
