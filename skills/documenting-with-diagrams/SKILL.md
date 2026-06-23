---
name: documenting-with-diagrams
description: Use when writing or editing Markdown documentation that needs Mermaid diagrams to clarify architecture, workflows, interactions, decisions, state transitions, timelines, or data relationships.
---

# Documenting with Diagrams

## Core Principle

Use Mermaid only when a visual relationship is easier than prose.

> Reference: [mmdc-cli](references/mmdc-cli.md) for CLI flags, [examples](references/examples.md) for walkthroughs.

## Question → Type

| Question | Type |
|---|---|
| What owns / depends on what? | `flowchart` + `subgraph` |
| Who interacts in what order, with what outcomes? | `sequenceDiagram` |
| How does a decision / algorithm branch? | `flowchart` |
| Which durable states + transitions? | `stateDiagram-v2` |
| Data entity relations? | `erDiagram` |
| Class / interface relations? | `classDiagram` |
| What happened / planned over time? | `timeline` |

Never combine static structure (class/ER) with runtime behavior (sequence) in one diagram. Split when questions mix.

## Zoom × typical types

| Zoom | Static (boundary) | Dynamic (flow) | Required text |
|---|---|---|---|
| **Overview** | 1 layer-boundary diagram — UI / State / Data, one named exemplar per layer | 1 sequenceDiagram (happy path) | 1-paragraph purpose + external-deps table + 1-paragraph state summary |
| **Detail** | 1 stateDiagram-v2 OR a deeper boundary | sequenceDiagram (error path or sub-flow) | — |
| **Drill-down** | primary flowchart | erDiagram / classDiagram referenced | — |

> **Don't** use a provider dependency graph as the Overview primary — that's a C4 *Component* diagram taken too far. The layer-boundary diagram replaces it.

## Workflow

### 1. Understand req

Parse source, target, zoom, mode. Detect missing inputs.

### 2. SPEC gate (only if inputs missing)

ONE question per missing input, max 4/call.

- **target:** ASK user. Default: `<req_dir>/diagrams/` where `<req_dir>` is the directory of the function/module from the request. Other valid: inline in existing doc.
- **zoom:** Overview / Detail / Drill-down (alias: Z1 / Z2 / Z3 — see Zoom × typical types section).
- **mode:** `Inline` (default) / `Inline+SVG` / `Static` (alias: R1 / R2 / R3).

Skip if trivial-req (≤1 fn, ≤1 module, no state) or all inputs present in the request.

### 3. Plan inventory (no file writes)

> **Do not call `Write` in this step.** Files are written in Step 5, only after Step 4 approval.

1. Pick a Mermaid type per question from the Question→Type section.
2. Decide file *names* and *types* — not content yet. Naming follows the Zoom × typical types section:
   - `overview.md` — purpose (1 para) + 1 boundary diagram + 1 happy-path sequence + external-deps table + state summary (1 para)
   - `<major>.detail.md` — 1 stateDiagram-v2 OR a deeper boundary + 1 sequence (error path or sub-flow)
   - `<major>-<minor>.drill.md` — primary flowchart + erDiagram / classDiagram referenced
3. Build the inventory table — `name | type | question` — one row per planned file.
4. Draft Mermaid source for each file — needed for Step 4 previews.

### 4. Confirm inventory (STOP GATE)

**Do not proceed to Step 5 until the user responds.** Writing files is irreversible enough to warrant a stop gate before `Write` calls.

Ask the user to approve the inventory. Include in the request:

- The inventory table (`name | type | question`).
- One short draft Mermaid snippet per file.

Wait for a reply, then route:

- **Approve** — proceed to Step 5.
- **Adjust** — apply the changes, loop back to Step 3.
- **Reject** — discard the plan.

### 5. Generate files (after approval only)

> Triggered only when Step 4 returns **Approve**. If Adjust or Reject, return to Step 3.

1. Write each file with `Write`, in parallel where possible.
2. Apply the self-review checklist *before* declaring the files done:
   - `;` in message/Note text → use `&#59;`
   - Bare arrows without `:`
   - `alt:`/`else:`/`opt:`/`loop:`/`par:` colon → use space
   - Literal `end` outside blocks → use `(end)` / `[end]` / `"end"`
   - `@` / `,` / `>` in `participant` IDs
   - `%%` is comment prefix (own line only)
   - **Provider dependency graph as Overview primary** — use the layer-boundary diagram instead
3. List the target directory to confirm all expected files are present before moving on.

### 6. Pre-render safety (STOP gate)

Before `mmdc`: STOP and ask if any of:
- Target dir unspecified
- Install mmdc or Chromium needed (never auto-install)
- Non-SVG format (default: SVG + white bg)
- Replacing existing `diagrams/` — confirm preserved files

Guard fails → Inline mode (no render).

### 7a. Things to know before rendering

- **SVG basename = `-o` stem, not `-i`.** `mmdc -i foo.md -o bar.md` → `bar-1.svg`. (Detail: `mmdc-cli.md` §"Render from .md".)
- **`-o <name>.svg` skips the transformed Markdown output.** mmdc writes SVGs only — no replaced `.md`. This is what lets §7b render without `/tmp/`.
- **`-a <dir>` uglies the Markdown link** when `-a` and `-o` resolve to different directories. Linux: `..` segments. Windows: mixed slashes / backslash paths in the rewritten link. Skip `-a` unless you must.

### 7b. Recipe: Inline + SVG render

Goal: keep `<name>.md` with inline `` ```mermaid `` blocks AND pre-render `<name>-N.svg` alongside.

```bash
cd <repo>/lib/features/<module>/diagrams

for f in *.md; do
  mmdc -i "$f" -o "${f%.md}.svg"
done
```

Renders SVGs directly next to each `<name>.md` — no `/tmp/`, no cleanup. Why: §7a bullet 2.

### 7c. mmdc quirks (worth knowing)

- **`-o foo.md` silently forces `outputFormat` to `svg`.** If you want PNG/PDF output, pass `-e png` or `-e pdf` explicitly — don't rely on `outputFormat` after `-o foo.md`.
- **`-a` rewrites the embedded link with native path separators.** On Windows that means backslashes, which can break rendering in cross-OS repos (Windows-authored markdown viewed on Linux CI). Another reason to skip `-a`.
- **Per-chart SVG suffixes are positional** (`-1`, `-2`, …), not content-addressed. Re-ordering or deleting a block in the source `.md` will renumber all subsequent SVGs and break any cached/external references.
