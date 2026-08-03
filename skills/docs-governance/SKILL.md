---
name: docs-governance
description: Discover documentation authorities and compare repository docs with a preferred contract. Use to audit, normalize, consolidate, update, or clean SSOTs, ADRs, specs, and their metadata while preserving local conventions, accepted architecture, and durable history.
---

# Documentation Governance

## Non-Negotiable Rules

- Read every repository and directory instruction applicable to a target file.
- Treat filenames, folders, and self-declared metadata as evidence, not proof of
  authority.
- Preserve read-only snapshots, generated docs, and decision history by default.
  Propose deleting rejected or replaced documents only under the confirmed
  merge and cleanup rules.
- Touch, merge, move, or delete only exact files approved by the user.

## Process

### 1. Explore

1. Read `references/document-contract.md`, resolve the repository/component in
   scope, and read applicable instruction files such as `AGENTS.md`,
   `CLAUDE.md`, `CONTRIBUTING.md`, and `RULES.md`.
2. Run the read-only inventory when shell access is available:

   ```bash
   bash scripts/discover-docs.sh /path/to/repository
   ```

   Use `--ext` to narrow formats and repeat `--exclude` for repository-specific
   noise; run the script with `--help` for exact syntax.

   Use its output as an inventory, not an authority decision:
   - read relevant files from `Documentation files`;
   - use `Governance` findings to locate possible ownership, precedence,
     supersession, and mutability rules;
   - use `Metadata` findings to discover existing document conventions;
   - verify every signal in its full file and repository context;
   - treat missing signals as unknown, not proof that no rule exists.
3. Discover the effective repository contract: authority, precedence,
   read-only/generated paths, metadata, ADR conventions, tooling, validation,
   and same-type conventions.
4. When conventions differ, read `references/normalization-and-cleanup.md` and
   classify each difference; do not silently map conflicts or unknowns.
5. Read relevant authorities and ADRs/RFCs. Inspect existing coverage and
   implementation.
6. Classify facts by their repository-defined owner. When no owner exists,
   propose one based on the fact's role: current invariant, decision rationale,
   evidence, operational procedure, or remaining work.

### 2. Report and Stop

Before editing, report:

- the effective contract, authorities, and precedence;
- differences from the preferred contract and proposed mappings;
- implementation/documentation conflicts;
- proposed files, exact changes, ownership rationale, and validation;
- uncertainty and decisions still required.

For normalization, merge, move, or cleanup, include the mutation report required
by `references/normalization-and-cleanup.md`. Stop unless the current request
already explicitly approves the exact files and changes.

### 3. Update After Confirmation

- Follow the contract precedence and apply only approved migrations.
- Modify only confirmed files and facts; amend the existing owning section.
- Preserve durable information. Keep temporary detail in research, runbooks, or
  plans.
- Use repository terminology and concise normative language. Consolidate stale
  text instead of appending a parallel rule.
- Normalize metadata field by field; never mass-update untouched docs.
- Refine an ADR when the decision identity is unchanged. Create a new ADR only
  for an independent or explicitly superseding decision required by local
  convention.

### 4. Validate and Report

Search the affected concepts again for contradiction or duplication. Run
repository-prescribed documentation checks, tests, linters, or audits. Verify
approved merges before cleanup. Report changes, ownership, contract migration,
validation, recovery options, and remaining decisions.
