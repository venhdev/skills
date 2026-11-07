//go:build windows

package updater

import (
	"fmt"
	"os"
	"path/filepath"
)

// CheckWritePermission checks whether the current user has write access to targetPath and its parent directory on Windows.
func CheckWritePermission(targetPath string) error {
	dir := filepath.Dir(targetPath)
	testFile := filepath.Join(dir, fmt.Sprintf("zenf_perm_test_%d.tmp", os.Getpid()))
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("cannot write to directory %s: %w", dir, err)
	}
	_ = os.Remove(testFile)
	return nil
}

// ReplaceBinary atomically replaces the executable on Windows.
// Windows locks running executables against write/delete, but permits renaming them.
func ReplaceBinary(targetPath string, newBinaryData []byte) error {
	dir := filepath.Dir(targetPath)

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
		return fmt.Errorf("failed writing new binary: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed closing new binary: %w", err)
	}

	backupPath := targetPath + ".old"
	_ = os.Remove(backupPath)

	// Rename running binary
	if _, err := os.Stat(targetPath); err == nil {
		if err := os.Rename(targetPath, backupPath); err != nil {
			return fmt.Errorf("failed renaming running binary on Windows: %w", err)
		}
	}

	// Move new binary into place
	if err := os.Rename(tmpPath, targetPath); err != nil {
		if _, statErr := os.Stat(backupPath); statErr == nil {
			_ = os.Rename(backupPath, targetPath)
		}
		return fmt.Errorf("failed placing new binary on Windows: %w", err)
	}

	// Purge legacy zenfd.exe if present
	legacyZenfd := filepath.Join(dir, "zenfd.exe")
	if _, err := os.Stat(legacyZenfd); err == nil {
		_ = os.Remove(legacyZenfd)
	}

	return nil
}
