package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ClientType designates supported AI agent hosts.
type ClientType string

const (
	ClientAgy    ClientType = "agy"
	ClientClaude ClientType = "claude"
	ClientCursor ClientType = "cursor"
)

// CheckResult represents the outcome of an MCP integration audit.
type CheckResult struct {
	Client        ClientType `json:"client"`
	ConfigPath    string     `json:"config_path"`
	FileExists    bool       `json:"file_exists"`
	Configured    bool       `json:"configured"`
	BinaryPath    string     `json:"binary_path,omitempty"`
	BinaryExists  bool       `json:"binary_exists"`
	DBPath        string     `json:"db_path,omitempty"`
	StatusMessage string     `json:"status_message"`
}

// SetupOptions holds parameters for configuring the zenfd MCP server.
type SetupOptions struct {
	Client     ClientType
	ConfigPath string
	DBPath     string
	BinaryPath string
	Force      bool
	DryRun     bool
}

// SetupResult details the modifications performed during MCP setup.
type SetupResult struct {
	Client      ClientType `json:"client"`
	ConfigPath  string     `json:"config_path"`
	BackupPath  string     `json:"backup_path,omitempty"`
	BinaryPath  string     `json:"binary_path"`
	DBPath      string     `json:"db_path"`
	WrittenJSON string     `json:"written_json"`
	Action      string     `json:"action"` // "configured", "already_configured", "dry_run"
}

// ExpandHome resolves a leading "~" or "~/" to the current user's home directory.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

// DefaultConfigFile returns the canonical MCP configuration file path for the client.
func DefaultConfigFile(client ClientType) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to resolve user home dir: %w", err)
	}
	switch client {
	case ClientAgy, "":
		return filepath.Join(home, ".gemini", "config", "mcp_config.json"), nil
	case ClientClaude:
		return filepath.Join(home, ".claude.json"), nil
	case ClientCursor:
		return filepath.Join(home, ".cursor", "mcp.json"), nil
	default:
		return "", fmt.Errorf("unsupported client: %s (supported: agy, claude, cursor)", client)
	}
}

// ResolveBinaryPath locates the zenfd executable.
func ResolveBinaryPath(override string) (string, error) {
	if override != "" {
		clean := ExpandHome(override)
		abs, err := filepath.Abs(clean)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(abs); err != nil {
			return "", fmt.Errorf("binary not found at %s: %w", abs, err)
		}
		return abs, nil
	}

	// 1. Check current process executable
	if execPath, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(execPath); err == nil {
			execPath = resolved
		}
		// Skip temporary test runner binaries
		if !strings.Contains(execPath, "/tmp/") && !strings.Contains(execPath, "\\Temp\\") {
			return execPath, nil
		}
	}

	// 2. Check PATH
	if path, err := exec.LookPath("zenfd"); err == nil {
		if abs, err := filepath.Abs(path); err == nil {
			return abs, nil
		}
	}

	// 3. Check standard repository/relative locations
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(cwd, "apps", "daemon", "bin", "zenfd"),
		filepath.Join(cwd, "bin", "zenfd"),
		filepath.Join(cwd, "zenfd"),
	}
	for _, cand := range candidates {
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			return cand, nil
		}
	}

	// Fallback to executable even if temp
	if execPath, err := os.Executable(); err == nil {
		return execPath, nil
	}

	return "", errors.New("could not resolve zenfd binary path; please provide --binary flag")
}

// CheckMCP audits the client's MCP configuration for zenfd.
func CheckMCP(client ClientType, configPathOverride string) (*CheckResult, error) {
	cfgPath := configPathOverride
	if cfgPath == "" {
		var err error
		cfgPath, err = DefaultConfigFile(client)
		if err != nil {
			return nil, err
		}
	} else {
		cfgPath = ExpandHome(cfgPath)
	}

	res := &CheckResult{
		Client:     client,
		ConfigPath: cfgPath,
	}

	fi, err := os.Stat(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			res.FileExists = false
			res.StatusMessage = fmt.Sprintf("Config file does not exist: %s", cfgPath)
			return res, nil
		}
		return nil, err
	}
	res.FileExists = true

	if fi.Size() == 0 {
		res.StatusMessage = fmt.Sprintf("Config file is empty (0 bytes): %s", cfgPath)
		return res, nil
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, err
	}

	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		res.StatusMessage = fmt.Sprintf("Invalid JSON in config file: %v", err)
		return res, nil
	}

	serversRaw, ok := root["mcpServers"]
	if !ok {
		res.StatusMessage = "No 'mcpServers' object found in config"
		return res, nil
	}

	servers, ok := serversRaw.(map[string]any)
	if !ok {
		res.StatusMessage = "'mcpServers' is not a JSON object"
		return res, nil
	}

	zenfdRaw, ok := servers["zenfd"]
	if !ok {
		res.StatusMessage = "Server 'zenfd' is not configured in mcpServers"
		return res, nil
	}

	zenfdMap, ok := zenfdRaw.(map[string]any)
	if !ok {
		res.StatusMessage = "'zenfd' entry is not a JSON object"
		return res, nil
	}

	res.Configured = true

	if cmdStr, ok := zenfdMap["command"].(string); ok {
		res.BinaryPath = cmdStr
		if _, err := os.Stat(cmdStr); err == nil {
			res.BinaryExists = true
		} else {
			res.BinaryExists = false
			res.StatusMessage = fmt.Sprintf("Configured command does not exist on disk: %s", cmdStr)
			return res, nil
		}
	}

	// Extract db argument if present
	if argsRaw, ok := zenfdMap["args"].([]any); ok {
		for i, arg := range argsRaw {
			if argStr, ok := arg.(string); ok && argStr == "--db" && i+1 < len(argsRaw) {
				if dbStr, ok := argsRaw[i+1].(string); ok {
					res.DBPath = dbStr
				}
			}
		}
	}

	res.StatusMessage = fmt.Sprintf("zenfd MCP server is configured and valid in %s", cfgPath)
	return res, nil
}

// SetupMCP injects or updates the zenfd MCP server entry in the target client's configuration file.
func SetupMCP(opts SetupOptions) (*SetupResult, error) {
	if opts.Client == "" {
		opts.Client = ClientAgy
	}
	cfgPath := opts.ConfigPath
	if cfgPath == "" {
		var err error
		cfgPath, err = DefaultConfigFile(opts.Client)
		if err != nil {
			return nil, err
		}
	} else {
		cfgPath = ExpandHome(cfgPath)
	}

	binPath, err := ResolveBinaryPath(opts.BinaryPath)
	if err != nil {
		return nil, err
	}

	dbPath := opts.DBPath
	if dbPath == "" {
		dbPath = "~/.zenforge/state.db"
	}
	expandedDB := ExpandHome(dbPath)

	res := &SetupResult{
		Client:     opts.Client,
		ConfigPath: cfgPath,
		BinaryPath: binPath,
		DBPath:     expandedDB,
	}

	// Read existing configuration
	root := make(map[string]any)
	fileExisted := false
	if fi, err := os.Stat(cfgPath); err == nil && fi.Size() > 0 {
		fileExisted = true
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read existing config: %w", err)
		}
		if err := json.Unmarshal(data, &root); err != nil {
			if !opts.Force {
				return nil, fmt.Errorf("existing config has invalid JSON (use --force to overwrite): %w", err)
			}
			root = make(map[string]any)
		}
	}

	// Extract or initialize mcpServers map
	var mcpServers map[string]any
	if existingServers, ok := root["mcpServers"].(map[string]any); ok {
		mcpServers = existingServers
	} else {
		mcpServers = make(map[string]any)
		root["mcpServers"] = mcpServers
	}

	// Check if already cleanly configured
	if existingEntry, ok := mcpServers["zenfd"].(map[string]any); ok && !opts.Force {
		existingCmd, _ := existingEntry["command"].(string)
		if existingCmd == binPath {
			res.Action = "already_configured"
			formatted, _ := json.MarshalIndent(root, "", "  ")
			res.WrittenJSON = string(formatted)
			return res, nil
		}
	}

	// Construct zenfd server entry
	zenfdEntry := map[string]any{
		"command": binPath,
		"args":    []string{"mcp", "--db", expandedDB},
	}
	mcpServers["zenfd"] = zenfdEntry

	formatted, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	formatted = append(formatted, '\n')
	res.WrittenJSON = string(formatted)

	if opts.DryRun {
		res.Action = "dry_run"
		return res, nil
	}

	// Ensure directory exists
	dir := filepath.Dir(cfgPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Backup existing non-empty file
	if fileExisted {
		backupPath := cfgPath + ".bak"
		data, err := os.ReadFile(cfgPath)
		if err == nil && len(data) > 0 {
			_ = os.WriteFile(backupPath, data, 0644)
			res.BackupPath = backupPath
		}
	}

	if err := os.WriteFile(cfgPath, formatted, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config to %s: %w", cfgPath, err)
	}

	res.Action = "configured"
	return res, nil
}
