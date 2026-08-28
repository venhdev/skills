---
name: docs-governance
description: Govern documentation authority and ownership boundaries. Use to search, audit, create, update, or clean SSOTs, ADRs, specs, and metadata when facts are needed, duplicated, conflicting, stale, or misplaced.
---

# Documentation Governance

## Core Rules

- Read every repository and directory instruction applicable to a target file.
- Treat filenames, folders, and self-declared metadata as evidence, not proof of authority.
- Keep one authoritative owner per rule. A correct fact in the wrong document is still cleanup work.
- Treat code, configuration, and tests as implementation evidence, not automatic replacements for accepted durable intent.
- Preserve read-only snapshots, generated docs, and decision history by default.
- Touch, merge, move, or delete only exact files approved by the user.

## Content Actions

| Type | Use when | Action |
| --- | --- | --- |
| `create` | Durable fact or document does not yet exist. | Draft in the designated authoritative owner. |
| `update` | Content exists but has factual gaps, errors, or missing information. | Correct facts, add missing details, fix inaccuracies. |
| `refine` | Content is factually correct but could be clearer, better structured, or have better examples. | Reorganize, clarify language, add examples, improve formatting. |
| `move` | Durable content belongs to another approved owner. | Merge into the owner, validate it, then remove the source copy. |
| `drop` | The owner already preserves the fact, or the content has no independent value. | Remove the source copy. |
| `escalate` | Authority, intent, or destination remains unclear. | Preserve it, report evidence and options, wait for user decision. |

Move before dropping. Do not replace every removed duplicate with a link; link only when kept content needs the owner for context.

## Process

### 1. Discover

1. **Fast-Path**: Check repository documentation index (`docs/README.md` or `README.md`) for an established Directory Structure and Authoritative Placement Matrix. If present, use it directly. Read `references/document-contract.md` for lifecycle status taxonomy, governance boundaries, and recommended directory templates.
2. **Fallback / Deep Audit**: If no index exists or scope is uncertain, inspect applicable instructions (`AGENTS.md`) and run the inventory script:

   ```bash
   bash scripts/discover-docs.sh /path/to/repository
   ```

   *Bootstrap recommendation*: If the repository lacks a clear documentation map, propose creating `docs/README.md` to define placement boundaries permanently using the template in `references/document-contract.md`.
3. Map relevant concepts to their canonical owner by scope (global standards, domain specs, subsystem guides).
4. **Read-only requests**: Stop and report authoritative files, sections, and findings when the request is purely search or inspection.

### 2. Propose

1. Classify each required change as `create`, `update`, `refine`, `move`, `drop`, or `escalate`.
2. When authoring or amending documents, follow the Universal Disciplines and Archetype Skeletons in `references/document-contract.md`.
3. When merging scattered facts into an owner:
   - **Synthesize**: Consolidate descriptions into a single canonical section or matrix; do not append parallel duplicate text.
   - **Reconcile**: Align conflicting text with code/contracts, or mark as `escalate`.
   - **Clean sources**: Replace moved text with a direct anchor link (`[Doc § N.N](path#anchor)`) only when local reading requires upstream context; otherwise drop.
4. Present the **Mutation Set Matrix** using this compact format:

   | Action | Source File (opt: -> Target) | Fact / Purpose |
   |---|---|---|
   | `move` | `path/source.md` -> `path/owner.md § N` | Consolidate naming invariant |
   | `create` | `path/owner.md` | Draft new protocol specification |
   | `update` | `path/owner.md` | Correct API version references |
   | `refine` | `path/owner.md` | Add examples to authentication section |
   | `drop` | `path/source.md` | Remove duplicate schema text |
   | `escalate` | `path/source.md` (vs `code`) | Conflicting state transition rules |

5. **Approval Gate**: Stop and wait for explicit user approval before modifying files.

### 3. Apply

1. Apply only approved mutations. Complete and validate `create` and `move` before `drop`. Stop on new escalations.
2. Amend existing owners with concise terminology (via `update`/`refine`).
3. Advance `updated: YYYY-MM-DD` in frontmatter of touched documents.

### 4. Verify

1. Check markdown link resolution and anchor validity.
2. Verify moved facts are complete and no duplicate text remains outside its SSOT.
