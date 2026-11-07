package attachment_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zenf/internal/attachment"
	"zenf/internal/domain"
)

func TestIngest_CopiesAndClassifiesMedia(t *testing.T) {
	root := t.TempDir()
	t.Setenv(attachment.RootEnv, root)

	content := append([]byte("\x89PNG\r\n\x1a\n"), []byte("fake-image-bytes")...)
	src := filepath.Join(t.TempDir(), "mockup.png")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatalf("failed to write source: %v", err)
	}

	artifact, err := attachment.Ingest("T-01", src, "")
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}

	if !strings.HasPrefix(artifact.ID, "art-in-T-01-") {
		t.Errorf("unexpected artifact id %q", artifact.ID)
	}
	if artifact.Role != domain.ArtifactRoleInput {
		t.Errorf("role = %q, want INPUT", artifact.Role)
	}
	if artifact.Type != domain.ArtifactTypeImage {
		t.Errorf("type = %q, want IMAGE", artifact.Type)
	}
	if artifact.Name != "mockup.png" {
		t.Errorf("name = %q, want the source file name", artifact.Name)
	}
	if artifact.ByteSize != int64(len(content)) {
		t.Errorf("byte_size = %d, want %d", artifact.ByteSize, len(content))
	}

	sum := sha256.Sum256(content)
	if artifact.Sha256 != hex.EncodeToString(sum[:]) {
		t.Errorf("sha256 mismatch")
	}

	if !strings.HasPrefix(artifact.FilePath, root) {
		t.Errorf("artifact was not copied under the configured root: %s", artifact.FilePath)
	}
	copied, err := os.ReadFile(artifact.FilePath)
	if err != nil {
		t.Fatalf("copied file is unreadable: %v", err)
	}
	if string(copied) != string(content) {
		t.Errorf("copied bytes differ from source")
	}
}

func TestIngest_RejectsUnsafeTaskID(t *testing.T) {
	t.Setenv(attachment.RootEnv, t.TempDir())

	src := filepath.Join(t.TempDir(), "file.png")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write source: %v", err)
	}

	for _, taskID := range []string{"../escape", "a/b", `a\b`, "..", ""} {
		if _, err := attachment.Ingest(taskID, src, ""); err == nil {
			t.Errorf("expected task id %q to be rejected", taskID)
		}
	}
}

func TestIngest_RejectsMissingAndDirectorySources(t *testing.T) {
	t.Setenv(attachment.RootEnv, t.TempDir())

	if _, err := attachment.Ingest("T-01", filepath.Join(t.TempDir(), "nope.png"), ""); err == nil {
		t.Error("expected missing source to be rejected")
	}
	if _, err := attachment.Ingest("T-01", t.TempDir(), ""); err == nil {
		t.Error("expected directory source to be rejected")
	}
}

func TestRemove_ConstrainsToAttachmentRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv(attachment.RootEnv, root)

	inside := filepath.Join(root, "T-01", "a.png")
	if err := os.MkdirAll(filepath.Dir(inside), 0o755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(inside, []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if err := attachment.Remove(inside); err != nil {
		t.Fatalf("Remove inside root failed: %v", err)
	}
	if err := attachment.Remove(inside); err != nil {
		t.Fatalf("Remove should be idempotent: %v", err)
	}

	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write outside file: %v", err)
	}
	if err := attachment.Remove(outside); err == nil {
		t.Fatal("expected Remove outside the root to fail")
	}
	if _, err := os.Stat(outside); err != nil {
		t.Errorf("outside file must be preserved: %v", err)
	}
}
