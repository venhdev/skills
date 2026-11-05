// Package attachment copies agent-provided media into the ZenForge attachment
// store and classifies it as a task INPUT artifact. The daemon never executes
// the file; it only stores bytes and metadata for the next claim.
package attachment

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"zenfd/internal/domain"
)

// MaxBytes bounds a single attachment. Agents should downscale screenshots
// rather than pushing multi-megabyte blobs through the context packet.
const MaxBytes = 25 << 20

// RootEnv overrides the attachment directory (used by tests and by operators
// who keep state outside the home directory).
const RootEnv = "ZENFD_ATTACHMENTS_DIR"

func storageRoot() (string, error) {
	if override := os.Getenv(RootEnv); override != "" {
		return override, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot resolve home directory: %w", err)
	}
	return filepath.Join(home, ".zenforge", "attachments"), nil
}

// Ingest copies srcPath into ~/.zenforge/attachments/<taskID>/ and returns the
// artifact metadata to register. The caller owns the returned file and should
// remove it if registration fails.
func Ingest(taskID, srcPath, displayName string) (domain.Artifact, error) {
	if err := ValidateTaskID(taskID); err != nil {
		return domain.Artifact{}, err
	}

	info, err := os.Stat(srcPath)
	if err != nil {
		return domain.Artifact{}, fmt.Errorf("cannot read %s: %w", srcPath, err)
	}
	if !info.Mode().IsRegular() {
		return domain.Artifact{}, fmt.Errorf("%s is not a regular file", srcPath)
	}
	if info.Size() > MaxBytes {
		return domain.Artifact{}, fmt.Errorf("%s is %d bytes, limit is %d", srcPath, info.Size(), MaxBytes)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return domain.Artifact{}, fmt.Errorf("cannot open %s: %w", srcPath, err)
	}
	defer src.Close()

	root, err := storageRoot()
	if err != nil {
		return domain.Artifact{}, err
	}
	dir := filepath.Join(root, taskID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return domain.Artifact{}, fmt.Errorf("cannot create attachment directory: %w", err)
	}

	artID := fmt.Sprintf("art-in-%s-%d", taskID, time.Now().UnixNano())
	baseName := filepath.Base(srcPath)
	targetPath := filepath.Join(dir, artID+"_"+baseName)

	target, err := os.Create(targetPath)
	if err != nil {
		return domain.Artifact{}, fmt.Errorf("cannot create %s: %w", targetPath, err)
	}

	hasher := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(target, hasher), src)
	closeErr := target.Close()
	if copyErr != nil {
		_ = os.Remove(targetPath)
		return domain.Artifact{}, fmt.Errorf("failed to copy media: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(targetPath)
		return domain.Artifact{}, fmt.Errorf("failed to flush media: %w", closeErr)
	}

	name := displayName
	if name == "" {
		name = baseName
	}

	mimeType := ResolveMime(targetPath, baseName, "")

	return domain.Artifact{
		ID:        artID,
		TaskID:    taskID,
		Role:      domain.ArtifactRoleInput,
		Name:      name,
		Type:      Classify(mimeType, baseName),
		FilePath:  targetPath,
		MimeType:  mimeType,
		ByteSize:  written,
		Sha256:    hex.EncodeToString(hasher.Sum(nil)),
		CreatedAt: time.Now().UTC(),
	}, nil
}

// Remove deletes a copied attachment. Paths outside the attachment root are
// refused. A missing file is treated as already removed.
func Remove(filePath string) error {
	root, err := storageRoot()
	if err != nil {
		return err
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(absRoot, absFile)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s is outside the attachment store", filePath)
	}

	if err := os.Remove(absFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove %s: %w", filePath, err)
	}
	return nil
}

// ValidateTaskID rejects values that would escape the attachment directory when
// used as a path component.
func ValidateTaskID(taskID string) error {
	if taskID == "" {
		return fmt.Errorf("task_id cannot be empty")
	}
	if strings.ContainsAny(taskID, `/\`) || taskID == "." || taskID == ".." {
		return fmt.Errorf("task_id %q is not a safe path component", taskID)
	}
	return nil
}

// ResolveMime prefers the declared type, then the file extension, then content
// sniffing. It is shared by the MCP ingest path and the HTTP upload path.
func ResolveMime(path, name, declared string) string {
	if declared != "" && declared != "application/octet-stream" {
		return declared
	}
	if byExt := mime.TypeByExtension(filepath.Ext(name)); byExt != "" {
		return byExt
	}
	return sniffMime(path)
}

// Classify maps a MIME type and file name onto the artifact taxonomy.
func Classify(mimeType, name string) domain.ArtifactType {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return domain.ArtifactTypeImage
	case strings.Contains(mimeType, "markdown") || strings.HasSuffix(strings.ToLower(name), ".md"):
		return domain.ArtifactTypeMarkdown
	case strings.Contains(mimeType, "pdf"):
		return domain.ArtifactTypeDoc
	default:
		return domain.ArtifactTypeFile
	}
}

func sniffMime(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return "application/octet-stream"
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && n == 0 {
		return "application/octet-stream"
	}
	return http.DetectContentType(buf[:n])
}
