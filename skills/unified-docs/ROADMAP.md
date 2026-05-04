# unified-docs: Architecture & Extension Guide

## Skill Architecture

The skill is organized in **layers**, from general (outer) to specific (inner):

```
SKILL.md                          ← Entry point: mode routing + lazy-load table
├── modes/[mode].md               ← Common logic per mode
├── workflows/[mode]-[doc-type]/  ← Specific logic for mode×doc-type
│   ├── flow.md                   ← AI-executable instructions
│   ├── [doc-type].md            ← Output document skeleton (primary template)
│   └── [supporting-files].md    ← Additional templates used by flow
├── contracts/                    ← Cross-cutting rules (lazy-loaded)
├── templates/
│   ├── authoring/                ← Doc skeletons (doc types without dedicated workflows)
│   └── reports/                  ← Report templates (shared across modes)
└── scripts/                      ← Utility scripts (general, not mode-specific)
```

### Mode → Workflow Routing Pattern

The core routing is simple and consistent:

```
modes/[mode].md
  ↓ (routes to based on doc type)
workflows/[mode]-[doc-type]/
  ├── flow.md               ← AI-executable instructions
  ├── [doc-type].md        ← Document skeleton/template
  └── [supporting].md      ← Additional templates
```

Example: Creating a plan doc → `workflows/create-plan/` → loads `flow.md` + `implementation-plan-template.md`

### Lazy-Load Principle

Only load what the mode and doc type need:

- SKILL.md always loads (small, fast routing)
- mode file loads based on user request classification
- contracts load conditionally (e.g., lifecycle.md only for plan/spec/ADR; organization.md only when reorganization is triggered)
- workflows load only when that specific mode×doc-type is active
- templates load on demand (authoring templates) or when workflow references them

This keeps context small and focused.

### Args (Quick Triggers)

SKILL.md supports optional args that pre-select mode and scope, skipping mode selection:

| Arg | Effect |
|---|---|
| `--create-plan` | Skip to plan doc creation (Create mode, plan type) |
| `--audit-codebase` | Full corpus audit + organization checks |
| `--audit-org` | Organization-only checks, skip frontmatter/lifecycle |
| `--maintain-plan` | Scan and update draft/in-progress plans (no archive) |

Each arg has corresponding logic in its mode file to load only the contracts and templates needed for that scope.

---

## Current Implementation

### Modes

| Mode | File | Purpose |
|------|------|---------|
| **Read** | `modes/read.md` | Answer status/current-truth questions without mutating files |
| **Create** | `modes/create.md` | Create new docs from authoring skeletons |
| **Maintain** | `modes/maintain.md` | Update existing docs, lifecycle metadata, or cascade links |
| **Audit** | `modes/audit.md` | Report docs health across a target set |

### Contracts

| Contract | File | Purpose | Lazy-Load Trigger |
|----------|------|---------|------------------|
| Frontmatter schema | `contracts/frontmatter.md` | Universal metadata schema | Always (all modes) |
| Type/kind choice | `contracts/classification.md` | Doc classification rules | Create, Audit, Maintain |
| Lifecycle rules | `contracts/lifecycle.md` | ADR, plan, spec lifecycle | Create, Maintain (when needed) |
| Cascade graph | `contracts/cascade.md` | Dependency and reciprocal links | Audit, Maintain (when needed) |
| **Organization patterns** | **`contracts/organization.md`** | **Folder structure guidance** | **Trigger: user reorganization request, audit detects structure issues, create from-scratch** |

### Workflows

| Folder | Mode | Doc Type | Purpose |
|--------|------|----------|---------|
| `workflows/create-plan/` | Create | plan | Two-tier planning workflow: milestone plan + optional detailed implementation |

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

1. **Layered by specificity**: General logic at outer layers, specific logic in `workflows/`
2. **Lazy-load discipline**: Only load what's needed for the mode and doc type. Contracts trigger conditionally based on user context.
3. **One workflow per folder**: Each mode×doc-type has its own folder, no shared workflows
4. **Self-contained workflows**: flow.md + templates live together, templates reference each other by relative path
5. **No duplicated templates**: If two workflows need the same template, consider promoting it to `templates/authoring/` (shared)
6. **Clear routing**: SKILL.md and each mode file make routing explicit, not implicit
7. **Arg-based shortcuts**: Args pre-select mode + scope; each mode's file defines arg-triggered behavior. Args load only contracts/templates needed for that specific scope.

---

## Adding Future Features

### Adding a doc type (e.g., "runbook")
1. If all modes treat it generically → add skeleton to `templates/authoring/runbook.md`
2. If a mode needs special handling → create workflow folder for that mode×doc-type

### Adding special behavior for existing doc type
1. Create workflow folder for that mode×doc-type
2. Move template from `templates/authoring/` → `workflows/[mode]-[doc-type]/`
3. Update mode's inputs-to-load and template mapping
4. Delete from `templates/authoring/`

### Splitting a large workflow
1. Create sub-workflows inside the same folder (e.g., `workflows/create-plan/setup.md`, `workflows/create-plan/execution.md`)
2. Reference them from flow.md
3. Update ROADMAP.md to explain the sub-structure

---

## Verification Checklist

After restructuring or adding a workflow:

- [ ] All files in `workflows/[mode]-[doc-type]/` are present and named correctly
- [ ] `flow.md` exists and is AI-readable (clear steps, no ambiguity)
- [ ] Primary template is named after the doc type
- [ ] `modes/[mode].md` has correct lazy-load entry and template mapping
- [ ] `SKILL.md` lazy-load routing mentions the workflow
- [ ] `ROADMAP.md` current workflows table is updated
- [ ] Old duplicate files are deleted (e.g., `templates/authoring/plan.md`)
- [ ] No circular dependencies (workflow A can't load workflow B)
- [ ] Template paths are relative (relative to the workflow folder or skill root)

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

**Last updated**: 2026-04-30  
**Skill version**: 2  
**Architecture version**: 2 (layered workflows)
