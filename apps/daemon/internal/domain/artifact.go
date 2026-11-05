package domain

import "time"

// ArtifactType categorizes concrete outputs produced during task execution.
type ArtifactType string

const (
	ArtifactTypeImage    ArtifactType = "IMAGE"
	ArtifactTypeMarkdown ArtifactType = "MARKDOWN"
	ArtifactTypeDoc      ArtifactType = "DOC"
	ArtifactTypeFile     ArtifactType = "FILE"
)

const (
	ArtifactRoleInput  = "INPUT"
	ArtifactRoleOutput = "OUTPUT"
)

// IsValid checks if the artifact type is supported.
func (a ArtifactType) IsValid() bool {
	switch a {
	case ArtifactTypeImage, ArtifactTypeMarkdown, ArtifactTypeDoc, ArtifactTypeFile:
		return true
	default:
		return false
	}
}

// Artifact represents a concrete file input or output tracked via the
// Claim-Check pattern.
type Artifact struct {
	ID        string       `json:"id"`
	TaskID    string       `json:"task_id"`
	Role      string       `json:"role"` // "INPUT" (spec/mockup) or "OUTPUT" (log/result)
	Name      string       `json:"name"` // Display filename or caption
	Type      ArtifactType `json:"type"`
	FilePath  string       `json:"file_path"`
	MimeType  string       `json:"mime_type"`
	ByteSize  int64        `json:"byte_size"`
	Sha256    string       `json:"sha256"`
	CreatedAt time.Time    `json:"created_at"`
}
