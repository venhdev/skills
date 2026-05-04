# unified-docs: Architecture & Extension Guide

## Skill Architecture

The skill is organized in **3 layers**, from general (outer) to specific (inner):

```
SKILL.md                              ← Entry point: first user-gated + routing
├── modes/[mode].md                   ← General rules per mode + routing table
├── contracts/doctypes/[type].md      ← Per-doc-type rules (mode-independent, always-on)
├── workflows/[name]/                 ← Specific logic for arg-triggered operations only
│   ├── flow.md                       ← ## Load (self-declared deps) + instructions
│   └── [supporting-files].md        ← Templates and content referenced by flow.md
├── contracts/                        ← Cross-cutting rules (lazy-loaded)
│   ├── frontmatter.md               ← Universal metadata schema
│   ├── classification.md            ← Type/kind taxonomy
│   ├── cascade.md                   ← Dependency graph rules
│   ├── multi-flow.md                ← Multi-flow detection (lazy, only when needed)
│   └── doctypes/                    ← Per-doc-type rules (6 files, one per type)
├── templates/
│   ├── authoring/                    ← Doc skeletons (types without dedicated workflows)
│   └── reports/                      ← Report templates (shared across modes)
└── scripts/                          ← Validation utilities (gitignore-aware)
```

### 3-Layer Loading

```
Request
  ↓
SKILL.md — first user-gated (always) + routing
  ↓
[Layer 2 — both always loaded when type is known]
modes/[mode].md              +   contracts/doctypes/[type].md
general rules + routing          type-specific rules, mode-independent
  ↓ (only when arg present)
[Layer 3]
workflows/[name]/flow.md
## Load (self-declared) + specific instructions
```


### Visual 3-layer

---
Kiến trúc mới — 3 layers

┌─────────────────────────────────────────────────────────┐
│ SKILL.md                                                │
│  └─ First user-gated: xác định flow(s) cần chạy        │
│      ├─ Single flow → proceed                           │
│      └─ Multi-flow → contracts/multi-flow.md (lazy)    │
│          ├─ A liên quan B? → sequential                 │
│          └─ Không? → parallel (subagents nếu hỗ trợ)   │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│ Layer 2 — luôn load cả hai, không thứ tự ưu tiên       │
│  ├─ modes/[mode].md    — general rules của mode đó      │
│  └─ contracts/doctypes/[type].md — rules của doc type   │
│      (mode-independent, luôn tuân theo)                 │
└─────────────────────────────────────────────────────────┘
                          ↓ (chỉ khi có arg)
┌─────────────────────────────────────────────────────────┐
│ Layer 3 — workflows/[name]/flow.md                      │
│  └─ Specific logic + ## Load (tự khai báo deps)         │
└─────────────────────────────────────────────────────────┘
---

### Lazy-Load Principle

- **SKILL.md**: always loads (small, fast routing + first user-gated)
- **Mode file**: loads once mode is determined; contains general rules only
- **Doctype file** (`contracts/doctypes/[type].md`): loads once doc type is known; always loaded for that type regardless of mode
- **Workflow** (`workflows/[name]/flow.md`): loads only when a matching arg is active; self-declares its own deps via `## Load` section
- **`contracts/multi-flow.md`**: lazy — only when multi-flow is detected
- **Contracts** (frontmatter, classification, cascade): loaded by workflow's `## Load` or by mode for non-workflow operations

This keeps context small and focused.

### Args (Quick Triggers)

Args pre-select mode + operation, routing directly to the corresponding workflow:

| Arg | Workflow | Effect |
|---|---|---|
| `--create-plan` | `workflows/create-plan/` | Create plan doc, skip to content discovery |
| `--audit-codebase` | `workflows/audit-codebase/` | Full corpus audit: chains audit-org + audit-naming + full checks |
| `--audit-org` | `workflows/audit-org/` | Organization check only |
| `--audit-naming` | `workflows/audit-naming/` | Naming check — asks scope if not specified |
| `--maintain-plan` | `workflows/maintain-plan/` | Scan and update draft/in-progress plans (no archive) |

**Rule**: A workflow exists only if there is a corresponding arg shortcut. If there is no arg, the operation is handled by mode + doctype rules directly.

---

## Current Implementation

### Modes

| Mode | File | General rules | Routing to |
|------|------|--------------|------------|
| **Read** | `modes/read.md` | Lifecycle state reading, replacement pointer following | — |
| **Create** | `modes/create.md` | Context scan, discovery gate, template mapping | `workflows/create-plan/` |
| **Maintain** | `modes/maintain.md` | Preserve body, normalize, cascade policy, drift detection | `workflows/maintain-plan/`, `workflows/audit-org/` |
| **Audit** | `modes/audit.md` | Scope, corpus boundaries, checks, severity, mutation policy | `workflows/audit-org/`, `workflows/audit-naming/`, `workflows/audit-codebase/` |

### Contracts

| Contract | File | Purpose | Load trigger |
|----------|------|---------|--------------|
| Frontmatter schema | `contracts/frontmatter.md` | Universal metadata schema (7 required fields) | Always |
| Type/kind taxonomy | `contracts/classification.md` | Type/kind rules, discovery-first requirement | Create, Audit, Maintain |
| Cascade graph | `contracts/cascade.md` | `depends-on`/`updates` semantics and repair policy | Audit, Maintain |
| Multi-flow | `contracts/multi-flow.md` | Multi-flow detection, sequential/parallel logic, plan-first | Lazy — only when multi-flow detected |
| **Plan rules** | **`contracts/doctypes/plan.md`** | **Plan lifecycle, status transitions, replacedBy, cascade rules** | **When doc type = plan** |
| **ADR rules** | **`contracts/doctypes/adr.md`** | **ADR lifecycle, adr-id, supersession, write-once body** | **When doc type = adr** |
| **Spec rules** | **`contracts/doctypes/spec.md`** | **Spec status, ssot rules, current-truth requirements** | **When doc type = spec** |
| **How-to rules** | **`contracts/doctypes/how-to.md`** | **Purpose, kind defaults, no lifecycle** | **When doc type = how-to** |
| **Explanation rules** | **`contracts/doctypes/explanation.md`** | **Purpose, kind defaults** | **When doc type = explanation** |
| **TIL rules** | **`contracts/doctypes/til.md`** | **Not canonical, not depends-on target** | **When doc type = til** |

### Workflows

| Folder | Arg | Purpose |
|--------|-----|---------|
| `workflows/create-plan/` | `--create-plan` | Two-tier planning: milestone plan + optional detailed implementation |
| `workflows/audit-org/` | `--audit-org` | Organization check (read-only) + reorganization (mutation, Maintain-triggered) |
| `workflows/audit-naming/` | `--audit-naming` | Naming check — asks scope if not specified |
| `workflows/audit-codebase/` | `--audit-codebase` | Full corpus: chains audit-org + audit-naming + full frontmatter/lifecycle/cascade |
| `workflows/maintain-plan/` | `--maintain-plan` | Plan status scan and update |

---

## How to Extend: Add a Doc-Type Workflow

When you want to add a special workflow for a doc type (e.g., Create mode for ADRs), follow this checklist:

### 1. Create the workflow folder
```
workflows/[mode]-[doc-type]/
├── flow.md                      (AI-executable instructions)
├── [primary-template].md        (doc skeleton to author)
└── [supporting-templates].md    (any additional templates)
```

Example: `workflows/create-adr/`

### 2. Write `flow.md`
- Start with description of the workflow
- Include trigger condition (when it's invoked)
- List concrete steps for the AI to execute
- Reference templates in the same folder

### 3. Add the primary template
- Name it after the doc type (e.g., `adr.md`)
- This is what the AI authors
- Can extract from existing `templates/authoring/` if exists

### 4. Add supporting templates (if needed)
- Descriptive names (e.g., `decision-template.md`, `option-comparison-template.md`)
- Referenced in flow.md

### 5. Update `modes/[mode].md`
Add two references:
- **Inputs to load**: Add condition to load the workflow directory
  ```markdown
  - all files in `workflows/create-adr/` if the doc is an ADR
  ```
- **Template mapping**: Add the doc type → template mapping
  ```markdown
  - ADR -> `workflows/create-adr/adr.md`
  ```

### 6. Update `SKILL.md`
Add to the "Lazy-load routing" section:
```markdown
- [Description]: all files in `workflows/[mode]-[doc-type]/` if [condition].
```

### 7. Update `ROADMAP.md`
Update the "Current Workflows" table to list the new workflow.

### 8. Optional: Remove from `templates/authoring/`
If the doc type previously had a generic skeleton in `templates/authoring/`, you can delete it — the workflow template replaces it.

---

## How to Extend: Add a New Mode

When you want to add a fourth mode (e.g., "Generate" mode that creates multiple docs):

### 1. Create `modes/[mode-name].md`
Include:
- Description: what this mode does
- Common behavior across all doc types
- Input loading strategy
- Doc type routing logic
- Template mapping

### 2. Update `SKILL.md`
- Add to "Modes" section with description
- Add to "Lazy-load routing" section

### 3. Create workflows for each doc type
For any doc type needing special behavior in this mode:
- Create `workflows/[mode]-[doc-type]/` folder
- Add `flow.md`, templates, etc.

### 4. Add mode-specific report template (if needed)
- Create `templates/reports/[mode]-report.md`
- Reference it in the mode's behavior

### 5. Update `ROADMAP.md`
- Add mode to "Current Implementation" section
- Document the new mode's purpose and workflow

---

## Naming Conventions

### Workflow Folder Names
Pattern: `workflows/[mode]-[doc-type]/`

- **mode**: one of {create, read, maintain, audit} (or custom new mode)
- **doc-type**: {plan, spec, adr, til, how-to, explanation}

Examples:
- `workflows/create-plan/` ✓
- `workflows/create-spec/` ✓
- `workflows/maintain-adr/` ✓
- `workflows/foobar-xyz/` ✗ (unclear mode/type)

### File Names in Workflows
- `flow.md` — always use this name for AI instructions
- **Primary template**: Should describe the doc type. Pattern: `[doc-type]-template.md` or just `[doc-type].md`
  - Example: `plan.md` (abstract pattern) or `implementation-plan-template.md` (descriptive variant)
  - Use whichever name makes the template's purpose clear
- **Supporting templates**: Should be descriptive (e.g., `option-comparison-template.md`, `phase-checklist-template.md`)

### New Modes
- Use imperative, present-tense verbs when possible ({create, read, maintain, audit})
- Keep names short and clear (one or two words max)

---

## Architecture Principles

1. **3-layer loading**: SKILL.md (routing) → mode general rules + doctype rules → (arg only) workflow specific logic.
2. **Mode = general rules only**: Mode files contain behavior rules that apply to ALL operations of that mode. No doc-type-specific steps, no arg-triggered logic blocks.
3. **Doctype = mode-independent rules**: `contracts/doctypes/[type].md` holds what makes that doc type unique. Loaded whenever that type is active, regardless of mode.
4. **Workflow = arg-triggered only**: A workflow folder exists if and only if there is a corresponding arg shortcut. Non-arg operations are handled by mode + doctype rules.
5. **`flow.md` self-declares deps**: Every `flow.md` has a `## Load` section listing exactly which contracts and sibling files it needs. No external list of workflow files.
6. **First user-gated always**: Before any action, SKILL.md probes intent, validates the request matches skill capability, and detects single vs. multi-flow. All user-gated checkpoints must be approved before proceeding.
7. **Multi-flow → plan-first**: When multiple independent flows are detected, load `contracts/multi-flow.md`. Present a plan for all flows, confirm, then execute sequentially (related) or in parallel via subagents (unrelated).
8. **Many small focused files**: Prefer more files that are each focused over fewer large files. Each file should have one clear responsibility.
9. **No logic duplication**: If something is in a doctype file, it is not repeated in a mode file. If it is in a workflow, the mode only routes to it.
10. **Clear routing chain**: SKILL.md → mode (routing table) → workflow. Each hop is explicit in the file that makes it.

---

## Adding Future Features

### Adding a new doc type (e.g., "runbook")
1. Create `contracts/doctypes/runbook.md` — define type, kind, any additional fields, lifecycle rules, cascade rules.
2. Add skeleton to `templates/authoring/runbook.md` (for doc creation without a dedicated workflow).
3. If there is an arg shortcut needed → create `workflows/[action]-runbook/` with `flow.md` + `## Load` section.
4. Update ROADMAP.md contracts table and workflows table.

### Adding a new arg + workflow
1. Decide the arg name: `--[mode]-[action]` pattern.
2. Create `workflows/[name]/flow.md` with `## Load` section and instructions.
3. Add the arg to `SKILL.md` Args table.
4. Add a routing line to the relevant mode file's routing table.
5. Update ROADMAP.md.

### Adding a new mode
1. Create `modes/[mode].md` with: description, general behavior rules, routing table.
2. Mode file must NOT contain doc-type-specific logic — that lives in `contracts/doctypes/`.
3. Update `SKILL.md` Modes section.
4. Update ROADMAP.md.

### Splitting a large workflow
1. Add supporting files inside the same folder (e.g., `workflows/create-plan/tier2-checklist.md`).
2. Reference them from `flow.md`.
3. Update ROADMAP.md.

---

## Verification Checklist

After adding a workflow or doc type:

- [ ] `contracts/doctypes/[type].md` exists with: type, kind, additional fields, lifecycle rules, cascade rules
- [ ] `workflows/[name]/flow.md` has `## Load` section declaring all deps
- [ ] Workflow exists only if there is a matching arg in SKILL.md Args table
- [ ] Mode file routing table updated (1 line: `--arg → workflows/[name]/`)
- [ ] Mode file contains NO doc-type-specific logic (it routes, not implements)
- [ ] SKILL.md Args table updated
- [ ] ROADMAP.md contracts and workflows tables updated
- [ ] No logic appears in more than one place (no duplication between mode, doctype, workflow)
- [ ] No circular workflow references

---

## Questions or Edge Cases?

- **Q**: Can a workflow reference templates from another workflow?  
  **A**: Avoid it. If templates are shared, promote to `templates/authoring/`.

- **Q**: What if a doc type doesn't need a workflow?  
  **A**: Leave the skeleton in `templates/authoring/`. Workflows are optional.

- **Q**: Can I have multiple templates in one workflow?  
  **A**: Yes. One primary (doc skeleton), many supporting. All in the same folder.

- **Q**: How do I deprecate a workflow?  
  **A**: Mark it as deprecated in flow.md, keep it for backward compatibility, update ROADMAP.md.

---

**Last updated**: 2026-05-04  
**Skill version**: 3  
**Architecture version**: 3 (3-layer: mode + doctype + workflow; first user-gated; multi-flow)
