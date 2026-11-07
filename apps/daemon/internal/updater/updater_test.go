package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdater_IsNewer(t *testing.T) {
	tests := []struct {
		latest   string
		current  string
		expected bool
	}{
		{"v0.3.2", "0.3.1", true},
		{"v1.0.0", "0.3.1", true},
		{"0.3.2", "v0.3.2", false},
		{"v0.3.1", "0.3.2", false},
		{"v0.4.0", "0.3.9", true},
	}

	for _, tt := range tests {
		got := IsNewer(tt.latest, tt.current)
		if got != tt.expected {
			t.Errorf("IsNewer(%q, %q) = %v; want %v", tt.latest, tt.current, got, tt.expected)
		}
	}
}

func TestUpdater_VerifyChecksum(t *testing.T) {
	content := []byte("binary payload content")
	sum := sha256.Sum256(content)
	hash := hex.EncodeToString(sum[:])

	checksums := fmt.Sprintf("%s  zenf_0.3.2_linux_amd64.tar.gz\n11111111  other.tar.gz\n", hash)

	// Valid hash
	if err := VerifyChecksum(content, checksums, "zenf_0.3.2_linux_amd64.tar.gz"); err != nil {
		t.Fatalf("expected valid checksum, got error: %v", err)
	}

	// Mismatched hash
	tampered := []byte("tampered content")
	if err := VerifyChecksum(tampered, checksums, "zenf_0.3.2_linux_amd64.tar.gz"); err == nil {
		t.Fatal("expected error on tampered content, got nil")
	}

	// Missing file in checksums
	if err := VerifyChecksum(content, checksums, "zenf_0.3.2_darwin_arm64.tar.gz"); err == nil {
		t.Fatal("expected error on missing file, got nil")
	}
}

func TestUpdater_ExtractBinaryFromTarGz(t *testing.T) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	payload := []byte("#!/bin/sh\necho 'zenf v0.3.2'\n")
	hdr := &tar.Header{
		Name: "zenf",
		Mode: 0755,
		Size: int64(len(payload)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write tar header: %v", err)
	}
	if _, err := tw.Write(payload); err != nil {
		t.Fatalf("failed to write tar body: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("failed to close gzip: %v", err)
	}

	extracted, err := ExtractBinary(buf.Bytes(), "zenf_0.3.2_linux_amd64.tar.gz")
	if err != nil {
		t.Fatalf("failed to extract binary: %v", err)
	}
	if !bytes.Equal(extracted, payload) {
		t.Fatalf("extracted payload does not match original: got %q", string(extracted))
	}
}

func TestUpdater_AtomicReplaceAndPurgeLegacy(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "zenf")
	legacyZenfd := filepath.Join(tmpDir, "zenfd")

	// Create existing binary and legacy zenfd symlink
	if err := os.WriteFile(binPath, []byte("old_binary"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyZenfd, []byte("legacy_symlink"), 0755); err != nil {
		t.Fatal(err)
	}

	newPayload := []byte("new_binary_0.3.2")
	if err := ReplaceBinary(binPath, newPayload); err != nil {
		t.Fatalf("ReplaceBinary failed: %v", err)
	}

	// Verify new binary installed
	data, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("failed to read binary after update: %v", err)
	}
	if !bytes.Equal(data, newPayload) {
		t.Fatalf("content mismatch: got %s, want %s", string(data), string(newPayload))
	}

	// Verify legacy zenfd was purged
	if _, err := os.Stat(legacyZenfd); !os.IsNotExist(err) {
		t.Fatalf("legacy zenfd was not purged: %v", err)
	}
}
