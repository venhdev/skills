//go:build !windows

package updater

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// CheckWritePermission checks whether the current user has write access to targetPath and its parent directory.
func CheckWritePermission(targetPath string) error {
	dir := filepath.Dir(targetPath)

	// Check directory write permission
	if err := syscall.Access(dir, syscall.O_RDWR); err != nil {
		return fmt.Errorf("cannot write to directory %s (try running with sudo or install to ~/.local/bin): %w", dir, err)
	}

	// If file exists, check file write permission
	if _, err := os.Stat(targetPath); err == nil {
		if err := syscall.Access(targetPath, syscall.O_RDWR); err != nil {
			return fmt.Errorf("cannot write to binary %s (permission denied): %w", targetPath, err)
		}
	}

	return nil
}

// ReplaceBinary atomically replaces the executable at targetPath with newBinaryData.
// On POSIX, replacing a running executable is permitted by renaming the existing file first.
func ReplaceBinary(targetPath string, newBinaryData []byte) error {
	dir := filepath.Dir(targetPath)

	// Write new binary to a temporary file in the same directory (ensuring same filesystem mount)
	tmpFile, err := os.CreateTemp(dir, "zenf_update_*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file in %s: %w", dir, err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(newBinaryData); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed writing new binary data: %w", err)
	}
	if err := tmpFile.Chmod(0755); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed setting executable permissions: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed closing temporary binary file: %w", err)
	}

	backupPath := targetPath + ".old"
	_ = os.Remove(backupPath)

	// Step 1: Move running binary to backup
	if _, err := os.Stat(targetPath); err == nil {
		if err := os.Rename(targetPath, backupPath); err != nil {
			return fmt.Errorf("failed to move existing binary to %s: %w", backupPath, err)
		}
	}

	// Step 2: Move new binary into place
	if err := os.Rename(tmpPath, targetPath); err != nil {
		// Rollback if placement fails
		if _, statErr := os.Stat(backupPath); statErr == nil {
			_ = os.Rename(backupPath, targetPath)
		}
		return fmt.Errorf("failed to swap in new binary: %w", err)
	}

	// Step 3: Remove old backup
	_ = os.Remove(backupPath)

	// Step 4: Zero-trace purge: actively remove any legacy 'zenfd' symlink or binary in the same directory
	legacyZenfd := filepath.Join(dir, "zenfd")
	if _, err := os.Lstat(legacyZenfd); err == nil {
		_ = os.Remove(legacyZenfd)
	}

	return nil
}
