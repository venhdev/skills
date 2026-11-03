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
	fakeBin := filepath.Join(tmpDir, "fake_zenfd")
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
	fakeBin := filepath.Join(tmpDir, "fake_zenfd")
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

	// Read new file and verify both sqlite-helper and zenfd are present
	newData, err := os.ReadFile(cfgPath)
	require.NoError(t, err)

	var root map[string]any
	require.NoError(t, json.Unmarshal(newData, &root))
	assert.Equal(t, "active", root["customSetting"])

	servers := root["mcpServers"].(map[string]any)
	assert.Contains(t, servers, "sqlite-helper")
	assert.Contains(t, servers, "zenfd")

	zenfdEntry := servers["zenfd"].(map[string]any)
	assert.Equal(t, fakeBin, zenfdEntry["command"])

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
	fakeBin := filepath.Join(tmpDir, "fake_zenfd")
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
