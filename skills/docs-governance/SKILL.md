---
name: docs-governance
description: Govern documentation authority and ownership boundaries. Use to audit or clean SSOTs, ADRs, specs, and metadata when facts are duplicated, conflicting, stale, or stored in the wrong document.
---

# Documentation Governance

## Core Rules

- Read every repository and directory instruction applicable to a target file.
- Treat filenames, folders, and self-declared metadata as evidence, not proof of
  authority.
- Keep one authoritative owner per rule. A correct fact in the wrong document is
  still cleanup work.
- Treat code, configuration, and tests as implementation evidence, not automatic
  replacements for accepted durable intent.
- Preserve read-only snapshots, generated docs, and decision history by default.
- Touch, merge, move, or delete only exact files approved by the user.

## Content Actions

| Type | Use when | Action |
| --- | --- | --- |
| `keep` | Content belongs to the document's responsibility. | Preserve or refine it. |
| `move` | Durable content belongs to another approved owner. | Merge into the owner, validate it, then remove the source copy. |
| `drop` | The owner already preserves the fact, or the content has no independent value. | Remove the source copy. |
| `escalate` | Authority, intent, or destination remains unclear. | Preserve it, report evidence and options, and wait for the user's decision. |

Move before dropping. Do not replace every removed duplicate with a link; link
only when retained content needs the owner for context.

## Process

### 1. Discover

1. Resolve the repository/component in scope. Discover and read existing
   repository or directory instructions applicable to it, such as `AGENTS.md`
   and `CLAUDE.md`. Their absence is not a defect.
2. Run the read-only inventory when shell access is available:

   ```bash
   bash scripts/discover-docs.sh /path/to/repository
   ```

   Use `--ext`, `--exclude`, or `--help` as needed. Treat the output as evidence,
   not an authority decision; verify relevant governance and metadata signals in
   full context, and treat missing signals as unknown.
3. Use the inventory and repository context to identify and read relevant
   authorities. Common candidates include `README.md`, `ARCHITECTURE.md`,
   `BUSINESS.md`, `DESIGN.md`, `SECURITY.md`, and `TEST.md`; filenames remain
   evidence, not proof of authority.
4. Establish the effective contract and each authority's responsibility. Read
   `references/document-contract.md` when auditing metadata, lifecycle, or
   type/status conventions, or when the repository defines no effective
   documentation contract.
5. Read relevant ADRs/RFCs and inspect implementation evidence. Classify
   relevant content as `keep`, `move`, `drop`, or `escalate`. When no owner
   exists, propose one based on the fact's role; escalate unclear authority or
   intent.

### 2. Propose

Before editing, report the effective authorities and responsibilities; each
relevant `keep`, `move`, `drop`, or `escalate`; exact source-to-owner mappings
and mutations; retained or lost durable value; conflicts; affected tooling;
validation, risk, and recovery. Stop unless the exact mutation set is already
approved.

### 3. Apply

- Apply only the approved mutation set. Complete and validate moves before
  drops; stop if new escalations or out-of-scope content appear.
- Amend existing owners with concise repository terminology. Consolidate stale
  text instead of appending parallel rules.
- Preserve read-only/generated artifacts and decision history. Normalize only
  metadata fields included in scope.
- Refine an ADR when the decision identity is unchanged. Create a new ADR only
  for an independent or explicitly superseding decision required by local
  convention.

### 4. Verify

- Search affected concepts for contradictions, duplicates, and content remaining
  outside its owner.
- Verify moved content is complete and run repository-prescribed documentation
  checks, tests, linters, or audits.
- Report changes, ownership, validation results, recovery options, and remaining
  escalations.
