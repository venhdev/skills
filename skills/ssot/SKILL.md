---
name: ssot
description: "Audit documentation authority, reconcile conflicting specifications, and maintain canonical Single Source of Truth records across repository documentation."
---

# ssot — Single Source of Truth & Documentation Custodian

## Operating Invariants

- **Authority Grounding**: Maintain exactly one authoritative owner per rule, contract, or concept; forbid duplicate or competing authorities across documents. Treat folder paths and file metadata as evidence, not proof of authority.
- **Read-Only Inspection Discipline**: Inspect and report authoritative files, sections, and findings for search, navigation, or audit-only requests; keep documentation unchanged in those modes.
- **Scope Discipline**: Touch, move, or prune exclusively approved documentation files using repo-relative POSIX links; forbid codebase mutations, host-absolute paths, or `file:///` URIs.
- **Pre-Mutation Gate**: Stage documentation changes in Changeset format (`[CREATE]`, `[UPDATE]`, `[MOVE]`, `[DELETE]`); halt before edits when the concrete scope lacks authorization, and honor authorization already provided.

## Domain Rubric

Classify every planned documentation operation into exactly one of the 4 atomic actions:

- `[CREATE]`: New canonical specifications, ADRs, schemas, or placement maps (`docs/README.md`).
- `[UPDATE]`: Correcting factual gaps, amending existing owners with concise terminology, or advancing frontmatter metadata (`updated: YYYY-MM-DD`).
- `[MOVE]`: Relocating durable content to its designated canonical owner, replacing with direct anchor link (`[Doc § N.N](path#anchor)`) only when local reading requires upstream context.
- `[DELETE]`: Pruning source duplicates once consolidated into canonical owner, or removing obsolete documentation with zero independent value.

## Execution Protocol

### Phase 1: Discover & Authority Audit

1. **Index Fast-Path**: Check repository documentation index (`docs/README.md` or `README.md`) for an established Placement Matrix. Refer to local `references/document-contract.md` for lifecycle taxonomy and recommended directory templates.
2. **Deep Audit Fallback**: If no index exists or scope is broad, run the local inventory scanner:

   ```bash
   bash scripts/discover-docs.sh <path_to_repository>
   ```

3. **Read-Only Exit**: If the request is search or audit only, report authoritative locations with pointer citations (`file:///path#L1-L20`) and halt turn immediately.
4. **Contradiction Escalation Gate**: If specifications conflict or canonical authority is ambiguous:
   - Halt turn immediately. Forbid proceeding to Phase 2.
   - Report conflicting evidence with pointer citations (`[Doc A:L12]` vs `[Doc B:L40]`).
   - Present resolution options and wait for explicit human direction before staging mutations.

### Phase 2: Staging & Authorization Gate

1. Stage planned modifications in Changeset format:

   ```text
   # Changeset: SSOT Documentation Governance (<Target Scope>) [(<optional_docs_root>/)]

   |-- docs/<collapsed_subsystem_dir>/
   |   |-- <document_1>.md
   |   |   `-- [<ACTION>] <Specification or canonical update>.
   |   `-- <document_2>.md
   |       `-- [<ACTION>] <Reconciliation or duplicate pruning>:
   |           * <Specific document authority or section update>.
   `-- docs/<deep/path/to/canonical_spec.md>
       `-- [<ACTION>] <Direct leaf spec without redundant folder node>.

   Summary: <N> files affected (<C> created, <U> updated, <M> moved, <D> deleted).
   ```

2. Present strictly the high-density Changeset summary and technical rationale. Forbid dumping raw document diffs into chat.
3. If the staged scope lacks authorization, halt before modifying documentation. Continue to application when that concrete scope is already authorized.

### Phase 3: Application & Link Verification

1. Upon receiving approval, apply mutations to disk (complete `[CREATE]` and `[MOVE]` before `[DELETE]`).
2. Advance `updated: YYYY-MM-DD` only for meaningful content or lifecycle changes, following the effective documentation contract.
3. Verify markdown link resolution and anchor validity.
4. Correct link and anchor failures within authorized scope and re-verify. If resolution requires a new authority decision, expanded scope, or unavailable verification, report `UNRESOLVED` with evidence and retained edits; keep redesign outside the original scope.
