---
name: zenforge-init
description: "Scaffold repository task tracking, Git ignore privacy, SSOT documentation index, and agent constitution."
disable-model-invocation: true
---

# zenforge-init — Repository Scaffolding Engine

Initialize repository task tracking, Git privacy safeguards, SSOT documentation governance, and agent constitution hooks.

## Operating Invariants

- **Pre-Mutation Gate**: Stage strictly missing assets via Changeset and halt turn for human approval; never overwrite valid existing configurations.
- **Privacy Baseline**: Always isolate `.agents/scratch/` in `.gitignore`; forbid committing transient scratch files.

## Scaffolding Components & Templates

### 1. Task Tracker Matrix (`.agents/task-tracker.md`)

- Remote GitHub + `gh auth status` valid ──> Propose **GitHub Issues (`gh`)** (template [`references/issue-tracker-github.md`](./references/issue-tracker-github.md)) or **Local Markdown**.
- Remote GitLab + `glab auth status` valid ──> Propose **GitLab Issues (`glab`)** (template [`references/issue-tracker-gitlab.md`](./references/issue-tracker-gitlab.md)) or **Local Markdown**.
- Unauthenticated CLI / Other remotes / Offline ──> Propose **Local Markdown** (`.agents/tasks/`, template [`references/issue-tracker-local.md`](./references/issue-tracker-local.md)). Report CLI auth status to user so they can choose to authenticate or remain local.

### 2. Git Privacy Configuration (`.gitignore`)

- `.agents/scratch/`: Private scratchpad (immutable).
- `.agents/tasks/`: Local-Only default for local markdown; omit if user requests team-shared tracking.

### 3. Documentation Placement Matrix (`docs/README.md`)

```markdown
# Documentation Index & Placement Matrix

| Topic / Scope | Authoritative SSOT | Responsibility |
| :--- | :--- | :--- |
| **Tasks & Execution** | Defined in `.agents/task-tracker.md` | Decomposed task units and progress tracking |
| **Documentation Governance** | `docs/` | Governed via `/ssot` (when available) |
```

### 4. Agent Constitution Hook (`AGENTS.md` or `CLAUDE.md`)

Target `CLAUDE.md` if present; otherwise target `AGENTS.md` (create if absent). If target file exists, append this block; do not overwrite:

```markdown
## Agent Workflow
- **Task Tracker**: Governed by `.agents/task-tracker.md`.
- **Documentation**: Governed by `docs/README.md` (via `/ssot` when available).
```

## Execution Protocol

**SUB-SKILL:** changeset, ssot

### Phase 1: Read-Only Discovery

1. **Remote & Topology**: Run `git remote -v` to resolve host and repository path (`owner/repo`).
2. **Toolchain Pre-flight**: Test `gh auth status` / `glab auth status` and record authentication state for the detected remote.
3. **Workspace Signals**:
   - Check `AGENTS.md` vs `CLAUDE.md` (preserve existing, forbid duplicating).
   - Check `.gitignore`, `.agents/task-tracker.md`, `.agents/tasks/`.
   - Scan existing documentation (`docs/README.md` or `README.md`).
4. **Clean-Pass Short-Circuit**: If all scaffolding assets (`.agents/task-tracker.md`, `.gitignore` privacy rules, `docs/README.md`, and constitution hook) are already configured and valid, report:

   ```text
   Repository ecosystem is already initialized and up-to-date.

   Pipeline Next Steps:
   - /clarify   ── Deliberate architecture or feature trade-offs.
   - /to-tasks  ── Decompose approved plans into vertical slice tasks.
   - /ssot      ── Audit and organize documentation authority.
   ```

   Halt turn immediately without staging redundant mutations.

### Phase 2: Staging & Authorization Gate

1. Stage missing or unconfigured scaffolding assets in Changeset format:

   ```text
   # Changeset: Repository Initialization

   📁 .agents/
   ├── 📄 task-tracker.md
   │   └── [CREATE] Configure task tracking mode and execution protocols.
   ├── 📁 tasks/ (Local Markdown only)
   │   └── [CREATE] Create task directory for local markdown workflow.
   └── 📁 scratch/
       └── [CREATE] Create scratchpad directory for temporary test harnesses and probes.

   📁 docs/
   └── 📄 README.md
       └── [CREATE | UPDATE] Establish documentation Placement Matrix.

   📁 <root>/
   ├── 📄 .gitignore
   │   └── [CREATE | UPDATE] Add .agents/scratch/ and .agents/tasks/ privacy rules.
   └── 📄 <AGENTS.md | CLAUDE.md>
       └── [CREATE | UPDATE] Add agent workflow and documentation pointers.
   ```

2. Present Changeset summary, recommended tracker choice, and toolchain status (report auth status if CLI is unauthenticated). Forbid emitting raw file bodies into chat.
3. Halt turn immediately for human approval; forbid modifying workspace files on disk without affirmative authorization.

### Phase 3: Atomic Application & Handoff

1. Upon receiving approval, write staged files to disk.
2. Report setup complete and suggest next step: `/clarify`, `/to-tasks`, or `/ssot`.
