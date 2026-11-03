---
title: ZenForge Daemon (zenfd) Web Cockpit UI & Control Plane Specification
description: Control plane layout, ASCII wireframes, node visual progressions, and Blueprint Studio specifications for v0.1.0 baseline.
status: active
created: 2026-11-03
updated: 2026-11-03
---

# UI Layout Architecture & Control Plane Specification: v0.1.0

**Status**: `CONFIRMED`  
**Revision**: `v0.1.0` (Consolidated Baseline: Agentic Cockpit, Inspection Dock, Blueprint Studio & Swarm Drawer)  
**Package**: `apps/web` (Vite 6 + React 19 + Tailwind v4 + `@xyflow/react` + Radix UI)

---

## 1. Architectural Posture: The Agentic Cockpit

The web dashboard is an ergonomic, real-time control plane designed for **read-only autonomous observation** coupled with **strict human-in-the-loop triage gates**.

```text
┌────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│ ⚡ ZenForge  [📁 my-skills ▾] [Plan: zenfd ▾] [+ Plan] [🗑]   [ DAG Flow | Docs | Env ]   [⚡ Active: 2] [● 1123] │
├──────────────────────────────────────────────────────────────────────┬─────────────────────────────────┤
│  CENTER TOPOLOGICAL STAGE (React Flow)                               │  INSPECTION DOCK (Right Panel)  │
│                                                                      │                                 │
│   ┌───────────────┐        ┌─────────────────────────┐               │  [ Task P-01: domain entities ] │
│   │ [P-00] Seam   │        │ [P-01] Domain Entities  │               │  Status: RUNNING ⚡             │
│   │ COMPLETED     │───────>│ RUNNING ⚡              │               ├─────────────────────────────────┤
│   │ ✓ by Coder-0  │        │ 👑 Orch: ZenF Architect │               │  ACTIVE EPHEMERAL WORKER        │
│   └───────────────┘        │ 🤖 Worker: Coder-1 (PID)│               │  • Worker ID: subagent-coder-1  │
│           │                └─────────────────────────┘               │  • Orchestrator: orch-fedfc548  │
│           │                             │                            │  • Worker PID: 48310 (Alive)    │
│           │                             ▼                            │  • Worktree: .wt/p01            │
│           │                ┌─────────────────────────┐               │  • Runtime: 02m 15s             │
│           └───────────────>│ [P-02] DB Storage       │               │                                 │
│                            │ READY                   │               │  [ 🛑 Evict Worker ]            │
│                            │ (Awaiting claim...)     │               ├─────────────────────────────────┤
│                            └─────────────────────────┘               │  TARGET FILES                   │
│                                         │                            │  • internal/domain/entities.go  │
│                                         ▼                            │                                 │
│                            ┌─────────────────────────┐               │  VERIFICATION                   │
│                            │ [P-03] Web Cockpit      │               │  • go test -v -race ./...       │
│                            │ PENDING (Locked)        │               │                                 │
│                            └─────────────────────────┘               │  [ 📋 Copy MCP Claim Command ]  │
├──────────────────────────────────────────────────────────────────────┴─────────────────────────────────┤
│  [▼] AGENT SWARM DRAWER (Trượt lên khi click [⚡ Active: 2])                                            │
│   👑 Orch: ZenForge Architect (Session: fedfc548 • PID: 48291)                                         │
│    ├── ⚡ Coder-1 (PID: 48310) ──> Task P-01: domain entities [RUNNING - 2m 15s] [Worktree: .wt/p01]   │
│    └── ⚡ Coder-2 (PID: 48312) ──> Task P-04: scout research   [RUNNING - 0m 45s]                      │
└────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Key UI Principles in v0.1.0

### A. HeaderBar Evolution
- **Purged**: `Worker Pool: 0/5` (Fixed runner slots removed; ephemeral workers are dynamically counted).
- **Purged**: Dummy `Pause / Resume` button.
- **Purged**: `[🛑 Kill All]` button (Developers are not terminal babysitters; no arbitrary SIGKILL triggers).
- **Added**:
  - `Project Switcher` (`📁 repo_path ▾`): Discovers and switches between active local workspaces via `GET /api/v1/projects`.
  - `Plan Switcher & CRUD`: Dropdown `Plan: <slug> ▾` with Radix UI modal triggers `[+ Plan]` (Create Plan) and `[🗑]` (Delete Plan).
  - `Active Workers Counter` (`⚡ Active: N`): Displays count of actively running ephemeral workers. Clicking toggles the **Agent Swarm Drawer**.

### B. The 2 Human-in-the-Loop Touchpoints (Inspection Dock)
Developer interaction is restricted strictly to high-value gates:

#### Touchpoint 1: Developer Seeding Hub (`Task Status == READY`)
Before an agent claims a ready task, the developer can inspect and enrich the context:
- **Visual Attachments & Screenshots**: Dropzone supporting drag-and-drop and clipboard paste (`Ctrl+V`) for PNG/JPG/WebP mockups.
- **Supplemental Architectural Hints**: Textarea for developer-provided notes, constraints, or gotchas.
- **Save to Context**: Submits to `POST /api/v1/tasks/{id}/attachments` and `PATCH /api/v1/tasks/{id}/context`. Automatically populated into `ContextPacket.input_attachments` when claimed.

#### Touchpoint 2: Architectural Review Handshake (`Task Status == AWAITING_REVIEW`)
When a subagent detects drift and yields:
- Subagent has terminated; task displays reason, technical details, and suggested options.
- Developer selects an option or types guidance.
- Submitting calls `POST /api/v1/tasks/{id}/resolve-review`.
- **Task transitions back to `READY`** with the resolution recorded in history, ready for the next subagent to claim.

---

## 3. Node Visual Progression (`TaskNode.tsx`)

| State | Border & Glow | Visual Badges & Content | Action / Seam |
| :--- | :--- | :--- | :--- |
| **`READY`** | Amber `border-amber-500/40` | `[ READY ]` • Priority `P1` • Complexity `MEDIUM` | Click opens Developer Seeding Hub |
| **`RUNNING`** | Violet Pulsing `shadow-violet-500/40` | `[ RUNNING ]` • 👑 Orchestrator • 🤖 Worker ID (PID) • Elapsed Time | Read-only observation & Worker eviction |
| **`AWAITING_REVIEW`** | Orange Pulsing `shadow-orange-500/40` | `[ AWAITING_REVIEW ]` • Reason: Schema Drift | Click opens Review Decision form |
| **`SUSPENDED`** | Amber Static `border-amber-600/70` | `[ SUSPENDED ]` • 🔌 Worker PID Disconnected | `[ 🔄 Re-claim ]` and `[ ↩ Reset ]` |
| **`AWAITING_MERGE`** | Cyan `border-cyan-500/50` | `[ AWAITING_MERGE ]` • Isolated Worktree Verified | `[ Merge to Main ]` button |
| **`CONFLICT`** | Rose `border-rose-500/70` | `[ CONFLICT ]` • List of conflicting files | Auto-abort with manual resolution hint |
| **`COMPLETED`** | Emerald `border-emerald-500/40` | `[ COMPLETED ]` • ✓ Verified by `Coder-1` (historical) | Verified diff inspection |

---

## 4. Phase P-13: Dossier Explorer & Revision Diff (`DossierExplorer.tsx`)

### Layout Topology

```text
+---------------------------------------------------------------------------------------------------+
|  [📁 Repo: my-skills ▾] [Plan: zenfd ▾] [Active: P-13]     [ DAG | Swimlanes | Dossier | Docs ]   |
+---------------------------------------------------------------------------------------------------+
|  REVISION BAR:  [ Revision: v0.1.0 ▾ ]  (2 files, 22.7 KB)              [ ⇄ Compare with: draft ▾ ]|
+-----------------------------------+---------------------------------------------------------------+
|  FILES TREE (Snapshot)            |  DOCUMENT VIEW / DIFF VIEWER                                  |
|                                   |                                                               |
|  ▼ zenfd/                         |  # ZenForge Daemon: v0.1.0                                    |
|    📄 contracts.md      (14.2 KB) |  **Target**: docs/zenfd/ (Revision: v0.1.0)                   |
|    📄 ui-layout.md       (8.5 KB) |                                                               |
|                                   |  [Live Mermaid Diagram Rendering]                             |
|                                   |                                                               |
|                                   |  (If Compare Mode Active):                                    |
|                                   |  [ - Red Deleted Lines ]                                      |
|                                   |  [ + Green Added Lines ]                                      |
+-----------------------------------+---------------------------------------------------------------+
```

### Component Breakdown
1. **Header Navigation**: Add `Dossier` view button in `HeaderBar.tsx` next to `DAG` and `Docs`.
2. **Revision Bar**:
   - Primary Revision selector (`v0.1.0`, `draft`).
   - "Compare" toggle: opens secondary comparison selector.
3. **Left File Tree**:
   - Lists relative file paths in the active snapshot.
   - Shows file icons and byte sizes.
   - Highlights active file.
4. **Main Viewer**:
   - Single view: Full Markdown viewer with syntax highlighting and live dark-mode Mermaid rendering.
   - Diff view: Colored unified text diff highlighting added/removed lines between two revisions.

---

## 5. Phase P-13: System Blueprint Studio (`BlueprintStudio.tsx`)

### Layout Topology

```text
+---------------------------------------------------------------------------------------------------+
|  [📁 Repo: my-skills ▾] [Plan: zenfd ▾]                      [ DAG | Swimlanes | Dossier | Studio ]|
+-----------------------------------+---------------------------------------------------------------+
|  5-TIER BLUEPRINT NAVIGATOR       |  CANVAS & SPEC INSPECTOR                                      |
|                                   |                                                               |
|  🏛️ TIER 1: CONTEXT & TOPOLOGY    |  [ core-erd ] • Tier 3 (Persistence) • v2 • 3 Bound Paths    |
|    • system-context               +---------------------------------------------------------------+
|    • component-topology           |  [ 📊 Diagram ]  [ 📋 Schema Tables ]  [ 📝 Design Notes ]    |
|                                   |                                                               |
|  ⚡ TIER 2: WORKFLOWS & SEQUENCES  |  erDiagram                                                    |
|    • agent-claim-handshake        |    plans ||--o{ tasks : contains                              |
|    • review-resolution-flow       |    plans ||--o{ plan_revisions : tracks                       |
|                                   |    plan_revisions ||--o{ plan_snapshots : captures            |
|  🗄️ TIER 3: PERSISTENCE (ERD)     |                                                               |
|    • core-erd (Active)            |  [ Live Interactive Mermaid Pan/Zoom Canvas ]                 |
|                                   |                                                               |
|  🔌 TIER 4: BOUNDARY CONTRACTS    |                                                               |
|    • gateway-rest-api             |                                                               |
|    • mcp-tools-matrix             |                                                               |
|                                   |                                                               |
|  📜 TIER 5: ARCHITECTURE DECISIONS|                                                               |
|    • adr-001-ephemeral-swarm      |                                                               |
|    • adr-002-blueprint-studio     |                                                               |
+-----------------------------------+---------------------------------------------------------------+
```

### Component Breakdown
1. **Tier Grouping**: Automatic categorization into 5 orthogonal tiers (Topology, Sequences, ERD, Contracts, ADRs).
2. **Tri-View Inspector**:
   - **Diagram Tab**: Dark theme Mermaid renderer with zoom, pan, and full-screen expansion.
   - **Schema Tables Tab**: Formatted table of entities, columns, endpoints, or parameters.
   - **Design Notes Tab**: Markdown view of invariants, edge cases, and architectural rationale.
3. **Bound Paths Badge**: Displays files/folders governed by the spec with live auto-linking status.
