---
title: ZenForge Daemon (zenfd) Domain Contracts & Interface Specifications
description: Canonical domain models, task state machine, storage engine, MCP protocol, and REST/SSE contracts for v0.1.0 baseline.
status: active
created: 2026-11-03
updated: 2026-11-03
---

# Domain Contracts & Interface Specifications: ZenForge Daemon (zenfd) v0.1.0

**Status**: FROZEN (2026-11-03)  
**Package**: `domain` / `mcp` / `gateway` / `storage`  
**Revision**: `v0.1.0` (Consolidated Baseline: SQLite WAL, 20 MCP Tools, Ephemeral Swarm & Web Cockpit)

---

## 1. Domain Entities & State Models

```go
package domain

import (
	"context"
	"io"
	"time"
)

// TaskStatus defines all 10 canonical states of an atomic slice in the DAG.
type TaskStatus string

const (
	StatusPending        TaskStatus = "PENDING"         // Waiting for upstream DAG dependencies
	StatusReady          TaskStatus = "READY"           // Dependencies fulfilled, ready for agent claim
	StatusRunning        TaskStatus = "RUNNING"         // Actively executing under an ephemeral agent runner
	StatusAwaitingReview TaskStatus = "AWAITING_REVIEW" // Paused: Plan/scope drift detected, awaiting decision (Worker exited)
	StatusSuspended      TaskStatus = "SUSPENDED"       // Paused: Worker process died unexpectedly or orchestrator disconnected
	StatusAwaitingMerge  TaskStatus = "AWAITING_MERGE"  // Implemented & tested on worktree, staged for git merge
	StatusConflict       TaskStatus = "CONFLICT"        // Git merge conflict encountered during integration
	StatusBlocked        TaskStatus = "BLOCKED"         // Blocked by upstream failure or cancellation
	StatusCompleted      TaskStatus = "COMPLETED"       // Verified with exit code 0 + tests pass + merged (Worker terminated)
	StatusFailed         TaskStatus = "FAILED"          // Execution failure or unresolvable error
	StatusCancelled      TaskStatus = "CANCELLED"       // Terminated via operator decision
)

// State Machine Transition Rules:
// - PENDING         -> READY | BLOCKED | CANCELLED
// - READY           -> RUNNING | BLOCKED | CANCELLED
// - RUNNING         -> AWAITING_REVIEW | SUSPENDED | AWAITING_MERGE | COMPLETED | FAILED | CANCELLED
// - AWAITING_REVIEW -> READY (upon resolution) | FAILED | CANCELLED
// - SUSPENDED       -> READY (reset) | RUNNING (re-claimed with new agent) | CANCELLED
// - AWAITING_MERGE  -> COMPLETED | CONFLICT | RUNNING | CANCELLED
// - CONFLICT        -> READY | RUNNING | FAILED | CANCELLED
// - BLOCKED         -> READY | PENDING | CANCELLED
// - FAILED          -> READY | CANCELLED
// - COMPLETED       -> READY (explicit operator re-open)
// - CANCELLED       -> (Strictly terminal)

type TaskPriority string

const (
	PriorityP0 TaskPriority = "P0" // Blocker / Critical
	PriorityP1 TaskPriority = "P1" // High priority
	PriorityP2 TaskPriority = "P2" // Medium priority
	PriorityP3 TaskPriority = "P3" // Low / Nice to have
)

type TaskComplexity string

const (
	ComplexityLow    TaskComplexity = "LOW"
	ComplexityMedium TaskComplexity = "MEDIUM"
	ComplexityHigh   TaskComplexity = "HIGH"
	ComplexitySpike  TaskComplexity = "SPIKE"
)

type TaskRisk string

const (
	RiskLow      TaskRisk = "LOW"
	RiskMedium   TaskRisk = "MEDIUM"
	RiskHigh     TaskRisk = "HIGH"
	RiskCritical TaskRisk = "CRITICAL"
)

type Plan struct {
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	RepoPath  string    `json:"repo_path"`
	Status    string    `json:"status"` // ACTIVE, COMPLETED, DRIFTED
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Task struct {
	// 1. Identity & DAG Topology
	ID           string   `json:"id"`
	PlanID       string   `json:"plan_id"`
	RepoPath     string   `json:"repo_path"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`

	// 2. Triage & Fan-out Attributes
	Priority             TaskPriority   `json:"priority"`
	Complexity           TaskComplexity `json:"complexity"`
	RiskLevel            TaskRisk       `json:"risk_level"`
	RequiredSkills       []string       `json:"required_skills"`
	RequiredTools        []string       `json:"required_tools"`
	TargetFiles          []string       `json:"target_files"`
	AcceptanceCriteria   []string       `json:"acceptance_criteria"`
	VerificationCommands []string       `json:"verification_commands"`
	Harness              string         `json:"harness"`
	SkillCommand         string         `json:"skill_command"`
	WorktreePath         string         `json:"worktree_path"`

	// 3. Ephemeral Swarm Execution & Mutual Exclusion Lock
	Status          TaskStatus `json:"status"`
	ClaimedBy       string     `json:"claimed_by,omitempty"`       // Subagent / Worker ID
	OrchestratorID  string     `json:"orchestrator_id,omitempty"`  // Main Agent / Orchestrator Session ID
	WorkerRole      string     `json:"worker_role,omitempty"`      // 'CODER', 'TESTER', 'INSPECTOR', 'SCOUT'
	PID             int        `json:"pid,omitempty"`              // Worker OS PID for liveness check
	OrchestratorPID int        `json:"orchestrator_pid,omitempty"` // Orchestrator OS PID
	ClaimedAt       *time.Time `json:"claimed_at,omitempty"`

	// 4. Developer Seeding & Supplemental Notes (Seeded when READY)
	Notes string `json:"notes,omitempty"`

	// 5. Review & Architectural Drift Handshake (AWAITING_REVIEW)
	ReviewReason   string   `json:"review_reason,omitempty"`
	ReviewDetails  string   `json:"review_details,omitempty"`
	ReviewOptions  []string `json:"review_options,omitempty"`
	ReviewResponse string   `json:"review_response,omitempty"`
	ResolvedBy     string   `json:"resolved_by,omitempty"` // 'DEVELOPER' | 'ORCHESTRATOR'

	// 6. Integration & Merge Gate (AWAITING_MERGE / CONFLICT)
	ConflictFiles []string `json:"conflict_files,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AgentSession tracks active live agent sessions in the local operating environment.
type AgentSession struct {
	SessionID       string    `json:"session_id"`
	ParentSessionID string    `json:"parent_session_id,omitempty"` // NULL for Orchestrators; OrchestratorID for Subagents
	Role            string    `json:"role"`                        // 'ORCHESTRATOR' | 'SUBAGENT' | 'HUMAN_OPERATOR'
	Name            string    `json:"name"`                        // Display label (e.g. 'ZenForge Architect', 'Coder-1')
	PID             int       `json:"pid"`                         // OS PID monitored via syscall.Kill(pid, 0)
	Status          string    `json:"status"`                      // 'ACTIVE', 'DISCONNECTED', 'TERMINATED'
	ClaimedTasks    []string  `json:"claimed_tasks"`               // Currently claimed task IDs
	CreatedAt       time.Time `json:"created_at"`
	LastBeatAt      time.Time `json:"last_beat_at"`
}

// Artifact represents file assets (input mockups/media or output test reports/diffs).
type Artifact struct {
	ID        string    `json:"id"`
	RunID     string    `json:"run_id,omitempty"`
	TaskID    string    `json:"task_id"`
	Role      string    `json:"role"` // 'INPUT' (Developer seeded media/specs) | 'OUTPUT' (Diffs, logs)
	Name      string    `json:"name"`
	Type      string    `json:"type"` // 'IMAGE', 'DIFF', 'LOG', 'DOCUMENT'
	FilePath  string    `json:"file_path"`
	MimeType  string    `json:"mime_type"`
	ByteSize  int64     `json:"byte_size"`
	SHA256    string    `json:"sha256"`
	CreatedAt time.Time `json:"created_at"`
}

// ProjectDoc represents a living architectural document, ADR, or active memory.
type ProjectDoc struct {
	Slug            string    `json:"slug"`
	Title           string    `json:"title"`
	Category        string    `json:"category"` // ARCHITECTURE, SCHEMA, ADR, GUIDE, MEMORY
	ContentMarkdown string    `json:"content_markdown"`
	MermaidDiagrams []string  `json:"mermaid_diagrams,omitempty"`
	Version         int       `json:"version"`
	RepoPath        string    `json:"repo_path"`
	GitBranch       string    `json:"git_branch"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ContextPacket is the prompt-ready manifest delivered to an agent upon claim.
type ContextPacket struct {
	Task                 Task                 `json:"task"`
	TargetFiles          []string             `json:"target_files"`
	SkillPaths           []string             `json:"skill_paths"`
	UpstreamDiffs        []string             `json:"upstream_diffs"`
	InputAttachments     []Artifact           `json:"input_attachments"`     // Seeded media & screenshots
	VerificationCommands []string             `json:"verification_commands"`
	PromptMarkdown       string               `json:"prompt_markdown"`
	LinkedSpecs          []CompactSpecPayload `json:"linked_specs,omitempty"` // Eagerly resolved path-bound tech specs
}
```

---

## 2. MCP Protocol Contracts (JSON-RPC 2.0 via Stdio)

### Tools (20 Canonical Tools in v2.5.0)
1. `zenfd_create_plan(slug string, title string, repo_path string)`: Creates a new living plan directly in SQLite WAL.
2. `zenfd_create_task(plan_id, id, title, description, dependencies, target_files, skills, tools, priority)`: Creates a vertical slice task in the plan DAG.
3. `zenfd_claim_task(task_id string, worker_id string, orchestrator_id string, worker_role string)`: Atomically locks task to `RUNNING`, records worker PID & orchestrator session, returns populated `ContextPacket` with eagerly injected `linked_specs`.
4. `zenfd_request_review(task_id string, reason string, details string, options []string)`: Transitions task to `AWAITING_REVIEW`, releases active worker lock (ephemeral worker exits), surfaces question on Web Cockpit and Chat.
5. `zenfd_resolve_review(task_id string, response string)`: Records decision into `review_response` and transitions task back to **`READY`** (so next subagent can claim with resolved guidance).
6. `zenfd_complete_task(task_id string, summary string)`:
   - In Worktree mode: captures diff, transitions to `AWAITING_MERGE`.
   - In Single workspace mode: transitions to `COMPLETED`, unlocks downstream DAG. Worker process is terminated/released.
7. `zenfd_merge_task(task_id string)`: Performs git merge from worktree to target branch. Transitions `AWAITING_MERGE` $\rightarrow$ `COMPLETED` (or `CONFLICT`).
8. `zenfd_fail_task(task_id string, reason string)`: Transitions task to `FAILED`.
9. `zenfd_reclaim_task(task_id string, new_worker_id string, new_orchestrator_id string)`: Re-claims a `SUSPENDED` task, preserving worktree and git diff without restarting from scratch.
10. `zenfd_evict_task(task_id string, reason string, action string)`: Orchestrator evicts a stalled/hung subagent (`action = "RESET_TO_READY" | "CANCEL"`).
11. `zenfd_register_session(session_id string, role string, name string, pid int)`: Registers an orchestrator or worker session in `agent_sessions`.
12. `zenfd_get_task(task_id string)`: Returns task metadata and Context Packet without mutation.
13. `zenfd_read_doc(slug string)`: Returns architectural doc or project memory.
14. `zenfd_write_doc(slug string, title string, category string, content string)`: Upserts doc into SQLite WAL.
15. `zenfd_export_docs(target_dir string)`: Exports docs to repository markdown files.
16. `zenfd_upsert_spec(slug, repo_path, tier, kind, title, diagram_type, diagram_code, structured_payload, notes, bound_paths)`: Registers or updates a structured architecture spec.
17. `zenfd_get_spec(slug string)`: Retrieves a specific architecture spec.
18. `zenfd_search_specs(query string)`: Full-text search across all specs via SQLite FTS5.
19. `zenfd_snapshot_plan(plan_slug string, folder_path string, revision string, summary string)`: Atomically snapshots all documents in a plan folder into SQLite WAL with revision tagging.
20. `zenfd_list_snapshots(plan_slug string)`: Lists all snapshot revisions and file manifests for a plan.

### Resources
- `zenfd://global/env`: Host machine toolchains, specs, and health diagnostics.
- `zenfd://global/sessions`: Active orchestrators and ephemeral workers currently running.
- `zenfd://global/skills`: System-wide installed agent skills.
- `zenfd://project/rules`: Project rules (`AGENTS.md`, `.agents/environment.md`).
- `zenfd://project/memory`: Active project memory and context distilled scratchpad.
- `zenfd://project/docs`: Index of all architectural documents.
- `zenfd://project/docs/{slug}`: Specific document content and Mermaid diagrams.
- `zenfd://tasks/{id}/context`: Formatted prompt packet for subagent execution.
- `zenfd://tasks/{id}/diff`: Git diff for task or upstream dependency.

---

## 3. REST API Gateway Contracts (`http://127.0.0.1:1123`)

| Method | Endpoint | Description |
| :--- | :--- | :--- |
| `GET` | `/api/v1/projects` | Lists all distinct repository paths tracked in the database. |
| `GET` | `/api/v1/plans` | Lists plans (supports `?repo_path=...`). |
| `POST` | `/api/v1/plans` | Creates a new plan (`slug`, `title`, `repo_path`). |
| `DELETE` | `/api/v1/plans/{slug}` | Cascading delete of plan, tasks, runs, artifacts, and docs. |
| `GET` | `/api/v1/tasks` | Lists tasks with optional filters (`?repo_path=...&status=...&plan_id=...`). |
| `POST` | `/api/v1/tasks` | Creates a new task in the specified plan. |
| `POST` | `/api/v1/tasks/{id}/dependencies` | Updates DAG dependency edges (`{"dependencies": ["P-00", "P-01"]}`). |
| `POST` | `/api/v1/tasks/{id}/attachments` | Uploads multipart media/screenshots (`role='INPUT'`, saved to disk & `artifacts`). |
| `PATCH` | `/api/v1/tasks/{id}/context` | Updates supplemental developer notes or criteria (`{"notes": "..."}`). |
| `POST` | `/api/v1/tasks/{id}/resolve-review` | Resolves review with developer decision; transitions task to `READY`. |
| `POST` | `/api/v1/tasks/{id}/reclaim` | Re-claims a `SUSPENDED` task with a new agent worker. |
| `POST` | `/api/v1/tasks/{id}/merge` | Integrates verified worktree into target git branch. |
| `GET` | `/api/v1/sessions` | Lists currently active agent sessions and their claimed tasks. |
| `GET` | `/api/v1/events` | Server-Sent Events (SSE) stream for real-time task status & DAG mutations. |
| `POST` | `/api/v1/plans/{slug}/snapshot` | Ingests a folder snapshot of living documents with revision tag and optional structured tasks. |
| `GET` | `/api/v1/plans/{slug}/revisions` | Returns all snapshot revisions recorded for a plan (`plan_revisions`). |
| `GET` | `/api/v1/plans/{slug}/snapshots` | Returns all snapshot files and contents for a specific revision (`?revision=...`). |
| `GET` | `/api/v1/plans/{slug}/diff` | Computes file-by-file text unified diff between two revisions (`?from=...&to=...`). |
| `GET` | `/api/v1/specs` | Lists architectural specifications with optional tier/kind filtering. |
| `POST` | `/api/v1/specs` | Upserts an architectural specification into `tech_specs`. |
| `GET` | `/api/v1/specs/search` | Full-text query against `tech_specs_fts` (`?q=...`). |

---

## 4. Phase P-13: Living Plan Folder Snapshot & Blueprint Studio Contracts

### 4.1 SQLite WAL Schema (`tech_specs`, `tech_specs_fts`, `plan_revisions`, and `plan_snapshots`)

```sql
-- 5-Tier Architecture Blueprints (Replaces raw docs/*.md)
CREATE TABLE IF NOT EXISTS tech_specs (
    slug TEXT PRIMARY KEY,
    repo_path TEXT NOT NULL,
    tier INTEGER NOT NULL,               -- 1: Topology, 2: Dynamics, 3: Data, 4: Contracts, 5: ADR
    kind TEXT NOT NULL,                  -- 'CONTEXT', 'FLOWCHART', 'SEQUENCE', 'STATE', 'ERD', 'CONTRACT', 'ADR'
    title TEXT NOT NULL,
    diagram_type TEXT NOT NULL,          -- 'mermaid:flowchart', 'mermaid:sequenceDiagram', 'mermaid:erDiagram', 'json:schema'
    diagram_code TEXT NOT NULL DEFAULT '',
    structured_payload TEXT NOT NULL DEFAULT '{}',
    notes TEXT NOT NULL DEFAULT '',
    bound_paths TEXT NOT NULL DEFAULT '[]', -- JSON array of file/folder prefixes for Path-Bound Auto-Linking
    version INTEGER NOT NULL DEFAULT 1,
    updated_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_tech_specs_tier ON tech_specs (tier, kind);

-- Full-Text Search Virtual Table for Specifications
CREATE VIRTUAL TABLE IF NOT EXISTS tech_specs_fts USING fts5(
    slug,
    title,
    diagram_code,
    notes,
    content='tech_specs',
    content_rowid='rowid'
);

-- FTS Synchronization Triggers
CREATE TRIGGER IF NOT EXISTS trg_tech_specs_ai AFTER INSERT ON tech_specs BEGIN
  INSERT INTO tech_specs_fts(rowid, slug, title, diagram_code, notes)
  VALUES (new.rowid, new.slug, new.title, new.diagram_code, new.notes);
END;

CREATE TRIGGER IF NOT EXISTS trg_tech_specs_ad AFTER DELETE ON tech_specs BEGIN
  INSERT INTO tech_specs_fts(tech_specs_fts, rowid, slug, title, diagram_code, notes)
  VALUES ('delete', old.rowid, old.slug, old.title, old.diagram_code, old.notes);
END;

CREATE TRIGGER IF NOT EXISTS trg_tech_specs_au AFTER UPDATE ON tech_specs BEGIN
  INSERT INTO tech_specs_fts(tech_specs_fts, rowid, slug, title, diagram_code, notes)
  VALUES ('delete', old.rowid, old.slug, old.title, old.diagram_code, old.notes);
  INSERT INTO tech_specs_fts(rowid, slug, title, diagram_code, notes)
  VALUES (new.rowid, new.slug, new.title, new.diagram_code, new.notes);
END;

CREATE TABLE IF NOT EXISTS plan_revisions (
    plan_slug TEXT NOT NULL,
    revision TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    comment TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (plan_slug, revision),
    FOREIGN KEY (plan_slug) REFERENCES plans(slug) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_plan_revisions_slug ON plan_revisions (plan_slug, created_at DESC);

CREATE TABLE IF NOT EXISTS plan_snapshots (
    id TEXT PRIMARY KEY,
    plan_slug TEXT NOT NULL,
    revision TEXT NOT NULL,
    rel_path TEXT NOT NULL,
    file_name TEXT NOT NULL,
    content_text TEXT NOT NULL,
    byte_size INTEGER NOT NULL,
    sha256 TEXT NOT NULL,
    mime_type TEXT NOT NULL DEFAULT 'text/markdown',
    created_at DATETIME NOT NULL,
    FOREIGN KEY (plan_slug, revision) REFERENCES plan_revisions(plan_slug, revision) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_plan_snapshots_lookup ON plan_snapshots (plan_slug, revision, rel_path);
```

### 4.2 Domain Entity Contracts (`internal/domain/entities.go`)

```go
// TechSpec represents a formal architectural specification stored as structured data.
type TechSpec struct {
    Slug              string         `json:"slug"`
    RepoPath          string         `json:"repo_path"`
    Tier              int            `json:"tier"`
    Kind              string         `json:"kind"`
    Title             string         `json:"title"`
    DiagramType       string         `json:"diagram_type"`
    DiagramCode       string         `json:"diagram_code"`
    StructuredPayload map[string]any `json:"structured_payload"`
    Notes             string         `json:"notes"`
    BoundPaths        []string       `json:"bound_paths"`
    Version           int            `json:"version"`
    UpdatedAt         time.Time      `json:"updated_at"`
}

// CompactSpecPayload represents eagerly resolved architecture context injected into ContextPacket.
type CompactSpecPayload struct {
    Slug        string         `json:"slug"`
    Kind        string         `json:"kind"`
    Title       string         `json:"title"`
    DiagramCode string         `json:"diagram_code"`
    Payload     map[string]any `json:"payload"`
}

// PlanRevision represents an immutable snapshot milestone for a living plan.
type PlanRevision struct {
    PlanSlug  string    `json:"plan_slug"`
    Revision  string    `json:"revision"`
    CreatedAt time.Time `json:"created_at"`
    Comment   string    `json:"comment"`
}

// SnapshotFile represents an individual document or asset preserved in a plan snapshot.
type SnapshotFile struct {
    ID          string    `json:"id"`
    PlanSlug    string    `json:"plan_slug"`
    Revision    string    `json:"revision"`
    RelPath     string    `json:"rel_path"`
    FileName    string    `json:"file_name"`
    ContentText string    `json:"content_text"`
    ByteSize    int64     `json:"byte_size"`
    SHA256      string    `json:"sha256"`
    MIMEType    string    `json:"mime_type"`
    CreatedAt   time.Time `json:"created_at"`
}

// FileDiff represents a computed unified diff between two file revisions.
type FileDiff struct {
    RelPath      string `json:"rel_path"`
    Status       string `json:"status"` // "ADDED", "MODIFIED", "DELETED", "UNCHANGED"
    OldSHA       string `json:"old_sha,omitempty"`
    NewSHA       string `json:"new_sha,omitempty"`
    UnifiedDiff  string `json:"unified_diff,omitempty"`
}
```
