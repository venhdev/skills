---
name: zenforge-init
description: "Initialize repository task tracking, configure .agents/task-tracker.md, and scaffold SSOT documentation governance."
disable-model-invocation: true
---

# zenforge-init — Repository Scaffolding & Ecosystem Bootstrapper

Initialize repository task tracking, Git privacy safeguards, SSOT documentation governance, and agent constitution hooks.

## Operating Invariants

- **Authority Grounding**: Inspect existing repository conventions before proposing structure; forbid modifying established documentation layouts without explicit human consent.
- **Pre-Mutation Gate**: Stage all planned file additions, modifications, and `.gitignore` entries in Changeset format and halt turn immediately; forbid creating or modifying files on disk without affirmative human approval.
- **Privacy First**: Isolate `.agents/scratch/` via `.gitignore`; forbid committing temporary scratchpads. Govern `.agents/tasks/` via explicit dual-mode consent (Local-Only vs Team-Shared).

## Domain Engine & Standards

### 1. Tracker Type Selection

- **Feasibility Matrix**:
  - Remote GitHub + `gh auth status` valid -> Propose **GitHub Issues (`gh`)** [Recommended] or **Local Markdown**. Seed from `references/issue-tracker-github.md`.
  - Remote GitHub + `gh` unauthenticated/missing -> Propose **Local Markdown**; suggest `gh auth login` for GitHub.
  - Remote GitLab + `glab auth status` valid -> Propose **GitLab Issues (`glab`)** [Recommended] or **Local Markdown**. Seed from `references/issue-tracker-gitlab.md`.
  - No remote / offline / solo -> Propose **Local Markdown (`.agents/tasks/`)**. Seed from `references/issue-tracker-local.md`.

### 2. Git Privacy & Ignore Invariants

- **Private Scratchpad (Strict Invariant)**: `.agents/scratch/` is always added to `.gitignore`.
- **Task Tracker Dual-Mode**:
  - *Local-Only (Recommended Default for local to prevent repo clutter)*: `.agents/tasks/` is added to `.gitignore`.
  - *Team-Shared*: `.agents/tasks/` is tracked in Git alongside code changesets.
- Confirm mode explicitly via Changeset before mutating `.gitignore`.

### 3. Canonical Templates

#### A. Task Tracker (`.agents/task-tracker.md`)

Instantiate `.agents/task-tracker.md` directly from the appropriate Tier 3 seed template in `references/`:

- **GitHub**: Seed from [`references/issue-tracker-github.md`](./references/issue-tracker-github.md)
- **GitLab**: Seed from [`references/issue-tracker-gitlab.md`](./references/issue-tracker-gitlab.md)
- **Local Markdown**: Seed from [`references/issue-tracker-local.md`](./references/issue-tracker-local.md)

#### B. Baseline Placement Matrix (`docs/README.md`)

Use as baseline for greenfield repositories; adapt rows dynamically to map observed documentation for existing projects:

```markdown
# Documentation Index & Placement Matrix

| Topic / Scope | Authoritative SSOT | Responsibility |
| :--- | :--- | :--- |
| **Tasks & Execution** | `.agents/tasks/` | Decomposed task units and progress tracking |
| **Domain & Specifications** | `docs/specs/` | Business rules, use cases, domain vocabulary |
| **Architecture & Decisions** | `docs/adr/` | Architectural Decision Records (MADR format) |
```

#### C. Constitution Hook (`AGENTS.md`)

```markdown
## Agent Workflow
- **Task Tracker**: Configured in `.agents/task-tracker.md`. Active tasks in `.agents/tasks/`.
- **Documentation**: Governed by `docs/README.md`.
```

## Execution Protocol

**SUB-SKILL:** changeset, ssot

### Phase 1: Read-Only Discovery

Inspect repository state:

1. **Remote & Topology**: Run `git remote -v` and inspect `.git/config` to resolve host and repository path (`owner/repo`).
2. **Toolchain Pre-flight**: Verify `command -v gh/glab` and test session via `gh auth status` / `glab auth status`.
3. **Workspace Signals**:
   - Check `AGENTS.md` vs `CLAUDE.md` (preserve existing, forbid duplicating).
   - Check `.gitignore`, `.agents/task-tracker.md`, `.agents/tasks/`.
   - Scan existing documentation (`docs/README.md`, `docs/specs/`, `docs/adr/`) and monorepo indicators.

### Phase 2: Changeset Staging & Lean Delivery

1. Stage proposed modifications for missing or unconfigured scaffolding assets in Changeset format:

   ```text
   # Changeset: Repository Initialization

   📁 .agents/
   ├── 📄 task-tracker.md
   │   └── [CREATE] Configure task tracking mode and execution protocols.
   ├── 📁 tasks/
   │   └── [CREATE] Create task directory for local markdown workflow.
   └── 📁 scratch/
       └── [CREATE] Create scratchpad directory for temporary test harnesses and probes.

   📁 docs/
   └── 📄 README.md
       └── [CREATE] Establish documentation Placement Matrix.

   📁 /
   ├── 📄 .gitignore
   │   └── [UPDATE] Add .agents/scratch/ and .agents/tasks/ privacy rules.
   └── 📄 AGENTS.md
       └── [UPDATE] Add agent workflow and documentation pointers.
   ```

2. **Lean Delivery**: Present strictly the Changeset summary, recommended configuration choices, and technical rationale. Forbid dumping voluminous raw file contents into chat by default.

### Phase 3: Authorization Gate

1. Present the staged Changeset and offer execution options:
   - Approve applying proposed configuration directly.
   - Request to adjust settings (Tracker type, Placement Matrix paths, Git privacy mode).
   - Request to inspect markdown previews.
2. Forbid modifying workspace files on disk within this turn.
3. Halt turn immediately and wait for affirmative human authorization (e.g., 'proceed', 'approved').

### Phase 4: Atomic Application & Handoff

1. Upon receiving approval, write staged files to disk.
2. **Circuit Breaker**: If writing files or updating `.gitignore` fails, halt immediately, report stderr, and prompt user whether to retry or abort. Forbid continuing silently on write failure.
3. Report completed setup and suggest next commands: `/clarify` (to deliberate new features) or `/to-tasks` (to decompose existing plans).
