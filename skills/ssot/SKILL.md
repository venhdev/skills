---
name: ssot
description: "Audit documentation authority, reconcile conflicting specifications, and maintain canonical Single Source of Truth records across repository documentation."
---

# ssot — Single Source of Truth & Documentation Custodian

Audit documentation authority, reconcile conflicting specifications, and maintain canonical Single Source of Truth (SSOT) records across the repository.

## Operating Invariants

- **Authority Grounding**: Maintain exactly one authoritative owner per rule, contract, or concept; forbid duplicate or competing authorities across documents. Treat folder paths and file metadata as evidence, not proof of authority.
- **Read-Only Inspection Discipline**: Stop and report authoritative files, sections, and findings immediately when the request is purely search, navigation, or auditing; forbid unsolicited edits.
- **Scope Discipline**: Touch, move, or prune exclusively approved documentation files; forbid modifying codebase implementation files or tests.
- **Pre-Mutation Gate**: Stage changes exclusively in Changeset format (`[CREATE]`, `[UPDATE]`, `[MOVE]`, `[DELETE]`) and halt turn immediately; forbid mutating documentation files on disk without affirmative human authorization.
- **Verification Circuit Breaker**: Stop execution immediately upon broken markdown link or anchor verification failure, report stderr with file citations, and prompt user whether to revert or keep debugging; forbid silent progression.

## Documentation Actions

Classify every planned documentation operation into exactly one of the 4 atomic actions:

- `[CREATE]`: New canonical specifications, ADRs, schemas, or placement maps (`docs/README.md`).
- `[UPDATE]`: Correcting factual gaps, amending existing owners with concise terminology, or advancing frontmatter metadata (`updated: YYYY-MM-DD`).
- `[MOVE]`: Relocating durable content to its designated canonical owner, replacing with direct anchor link (`[Doc § N.N](path#anchor)`) only when local reading requires upstream context.
- `[DELETE]`: Pruning source duplicates once consolidated into canonical owner, or removing obsolete documentation with zero independent value.

## Execution Protocol

**SUB-SKILL:** changeset, clarify

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
   # Changeset: SSOT Documentation Governance (<Target Scope>)

   📁 <directory_or_subsystem>/
   ├── 📄 <path/to/target_document_1>.md
   │   └── [<ACTION>] <Summary of specification or canonical link>.
   └── 📄 <path/to/target_document_2>.md
       └── [<ACTION>] <Summary of update or duplicate pruning>.
   ```

2. Present strictly the high-density Changeset summary and technical rationale. Forbid dumping raw document diffs into chat.
3. Halt turn immediately for human authorization; forbid modifying workspace files on disk without explicit approval.

### Phase 3: Application & Link Verification

1. Upon receiving approval, apply mutations to disk (complete `[CREATE]` and `[MOVE]` before `[DELETE]`).
2. Advance `updated: YYYY-MM-DD` in frontmatter of touched documents.
3. Verify markdown link resolution and anchor validity.
4. If verification fails, stop, report the error output, and ask the user whether to revert or keep debugging.
