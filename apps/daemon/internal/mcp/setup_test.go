package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPCheck_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "mcp_config.json")

	res, err := CheckMCP(ClientAgy, cfgPath)
	require.NoError(t, err)
	assert.False(t, res.FileExists)
	assert.False(t, res.Configured)
}

func TestMCPCheck_ZeroByteFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "mcp_config.json")
	require.NoError(t, os.WriteFile(cfgPath, []byte{}, 0644))

	res, err := CheckMCP(ClientAgy, cfgPath)
	require.NoError(t, err)
	assert.True(t, res.FileExists)
	assert.False(t, res.Configured)
}

func TestMCPSetup_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "mcp_config.json")
	fakeBin := filepath.Join(tmpDir, "fake_zenf")
	require.NoError(t, os.WriteFile(fakeBin, []byte("#!/bin/sh\n"), 0755))

	opts := SetupOptions{
		Client:     ClientAgy,
		ConfigPath: cfgPath,
		BinaryPath: fakeBin,
		DryRun:     true,
	}

	res, err := SetupMCP(opts)
	require.NoError(t, err)
	assert.Equal(t, "dry_run", res.Action)
	assert.Contains(t, res.WrittenJSON, fakeBin)

	// File should not exist on disk
	_, err = os.Stat(cfgPath)
	assert.True(t, os.IsNotExist(err))
}

func TestMCPSetup_PreservesExistingServersAndBacksUp(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "mcp_config.json")
	fakeBin := filepath.Join(tmpDir, "fake_zenf")
	require.NoError(t, os.WriteFile(fakeBin, []byte("#!/bin/sh\n"), 0755))

	// Initial config with custom server and user metadata
	initialJSON := `{
  "customSetting": "active",
  "mcpServers": {
    "sqlite-helper": {
      "command": "sqlite-mcp-server",
      "args": ["/var/data/app.db"]
    }
  }
}`
	require.NoError(t, os.WriteFile(cfgPath, []byte(initialJSON), 0644))

	opts := SetupOptions{
		Client:     ClientAgy,
		ConfigPath: cfgPath,
		BinaryPath: fakeBin,
		DBPath:     filepath.Join(tmpDir, "test_state.db"),
	}

	res, err := SetupMCP(opts)
	require.NoError(t, err)
	assert.Equal(t, "configured", res.Action)
	assert.NotEmpty(t, res.BackupPath)

	// Verify backup file contains initial content
	backupData, err := os.ReadFile(res.BackupPath)
	require.NoError(t, err)
	assert.Equal(t, initialJSON, string(backupData))

	// Read new file and verify both sqlite-helper and zenf are present
	newData, err := os.ReadFile(cfgPath)
	require.NoError(t, err)

	var root map[string]any
	require.NoError(t, json.Unmarshal(newData, &root))
	assert.Equal(t, "active", root["customSetting"])

	servers := root["mcpServers"].(map[string]any)
	assert.Contains(t, servers, "sqlite-helper")
	assert.Contains(t, servers, "zenf")

	zenfEntry := servers["zenf"].(map[string]any)
	assert.Equal(t, fakeBin, zenfEntry["command"])

	// CheckMCP should now report configured & valid
	checkRes, err := CheckMCP(ClientAgy, cfgPath)
	require.NoError(t, err)
	assert.True(t, checkRes.Configured)
	assert.True(t, checkRes.BinaryExists)
	assert.Equal(t, fakeBin, checkRes.BinaryPath)
}

func TestMCPSetup_AlreadyConfiguredIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "mcp_config.json")
	fakeBin := filepath.Join(tmpDir, "fake_zenf")
	require.NoError(t, os.WriteFile(fakeBin, []byte("#!/bin/sh\n"), 0755))

	opts := SetupOptions{
		Client:     ClientAgy,
		ConfigPath: cfgPath,
		BinaryPath: fakeBin,
	}

	// First run
	res1, err := SetupMCP(opts)
	require.NoError(t, err)
	assert.Equal(t, "configured", res1.Action)

	// Second run without force
	res2, err := SetupMCP(opts)
	require.NoError(t, err)
	assert.Equal(t, "already_configured", res2.Action)

	// Third run with force
	opts.Force = true
	res3, err := SetupMCP(opts)
	require.NoError(t, err)
	assert.Equal(t, "configured", res3.Action)
}

func TestDefaultConfigFile_OpenCode(t *testing.T) {
	path, err := DefaultConfigFile(ClientOpenCode)
	require.NoError(t, err)
	assert.Contains(t, path, filepath.Join(".config", "opencode", "opencode.json"))
}

func TestMCPSetup_OpenCodeFormat(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "opencode.json")
	fakeBin := filepath.Join(tmpDir, "fake_zenf")
	require.NoError(t, os.WriteFile(fakeBin, []byte("#!/bin/sh\n"), 0755))

	initialJSON := `{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "other": {
      "type": "local",
      "command": ["other-server"]
    }
  }
}`
	require.NoError(t, os.WriteFile(cfgPath, []byte(initialJSON), 0644))

	dbPath := filepath.Join(tmpDir, "state.db")
	opts := SetupOptions{
		Client:     ClientOpenCode,
		ConfigPath: cfgPath,
		BinaryPath: fakeBin,
		DBPath:     dbPath,
	}

	res, err := SetupMCP(opts)
	require.NoError(t, err)
	assert.Equal(t, "configured", res.Action)
	assert.NotEmpty(t, res.BackupPath)

	data, err := os.ReadFile(cfgPath)
	require.NoError(t, err)

	var root map[string]any
	require.NoError(t, json.Unmarshal(data, &root))

	mcpMap, ok := root["mcp"].(map[string]any)
	require.True(t, ok)
	assert.Contains(t, mcpMap, "other")
	assert.Contains(t, mcpMap, "zenf")

	entry, ok := mcpMap["zenf"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "local", entry["type"])
	assert.Equal(t, true, entry["enabled"])
	assert.NotContains(t, entry, "args")

	cmd, ok := entry["command"].([]any)
	require.True(t, ok)
	require.Len(t, cmd, 4)
	assert.Equal(t, fakeBin, cmd[0])
	assert.Equal(t, "mcp", cmd[1])
	assert.Equal(t, "--db", cmd[2])
	assert.Equal(t, dbPath, cmd[3])

	checkRes, err := CheckMCP(ClientOpenCode, cfgPath)
	require.NoError(t, err)
	assert.True(t, checkRes.Configured)
	assert.True(t, checkRes.BinaryExists)
	assert.Equal(t, fakeBin, checkRes.BinaryPath)
	assert.Equal(t, dbPath, checkRes.DBPath)

	// Second run should be idempotent.
	res2, err := SetupMCP(opts)
	require.NoError(t, err)
	assert.Equal(t, "already_configured", res2.Action)
}

func TestMCPSetup_PurgesLegacyZenfdEntry(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "mcp_config.json")
	fakeBin := filepath.Join(tmpDir, "fake_zenf")
	require.NoError(t, os.WriteFile(fakeBin, []byte("#!/bin/sh\n"), 0755))

	// Initial config with legacy zenfd entry
	initialJSON := `{
  "mcpServers": {
    "zenfd": {
      "command": "/old/path/zenfd",
      "args": ["mcp"]
    }
  }
}`
	require.NoError(t, os.WriteFile(cfgPath, []byte(initialJSON), 0644))

	opts := SetupOptions{
		Client:     ClientAgy,
		ConfigPath: cfgPath,
		BinaryPath: fakeBin,
	}

	res, err := SetupMCP(opts)
	require.NoError(t, err)
	assert.Equal(t, "configured", res.Action)

	newData, err := os.ReadFile(cfgPath)
	require.NoError(t, err)

	var root map[string]any
	require.NoError(t, json.Unmarshal(newData, &root))
	servers := root["mcpServers"].(map[string]any)

	assert.NotContains(t, servers, "zenfd")
	assert.Contains(t, servers, "zenf")
}

func TestMCPCheck_DetectsLegacyZenfdWarning(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "mcp_config.json")

	initialJSON := `{
  "mcpServers": {
    "zenfd": {
      "command": "/old/path/zenf",
      "args": ["mcp"]
    }
  }
}`
	require.NoError(t, os.WriteFile(cfgPath, []byte(initialJSON), 0644))

	res, err := CheckMCP(ClientAgy, cfgPath)
	require.NoError(t, err)
	assert.False(t, res.Configured)
	assert.Contains(t, res.StatusMessage, "found deprecated 'zenfd'")
	assert.Contains(t, res.StatusMessage, "run 'zenf mcp setup agy' to migrate")
}

func TestMCPSetup_RejectsUnsupportedOrEmptyClient(t *testing.T) {
	_, err := SetupMCP(SetupOptions{Client: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported client")

	_, err = SetupMCP(SetupOptions{Client: "vscode"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported client")

	_, err = CheckMCP("", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported client")
}
