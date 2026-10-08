---
name: ci-pub
description: Audit code health, propose semantic version bumps, and verify package publication readiness.
disable-model-invocation: true
---

# ci-pub — Automated Release Preflight & Publication Engine

## Operating Invariants

- **Pre-Bump Quality Gate**: Verify formatting, test suites, and changelog entries pass 100% before proposing version changes; halt on failures instead of proceeding.
- **Turn-Halt Version Selection**: Always halt the turn immediately after presenting version candidates; never mutate manifests or git history before explicit user selection.
- **Atomic Commit Amend**: Stage manifest and changelog updates directly into the HEAD commit via amend; avoid creating detached chore commits.

## Execution Protocol

### Step 1: Preflight Quality Audit
Run repository test suites, format checks, and changelog audits. If any verification fails, report diagnostics and halt immediately.

### Step 2: Semantic Version Proposal & Turn-Halt Gate
Assess changeset scope (breaking, feat, fix) and present exactly 3 SemVer options:

1. `<new-version>` (Patch, Minor, or Major per SemVer scope)
2. `<new-version>-beta` (Pre-release preview)
3. `<new-version>-beta.1` (Iterative pre-release candidate)

Halt turn immediately. Forbid modifying manifests or git history until the user selects a version.

### Step 3: Manifest Mutation, Amend & Dry-Run Check
Upon user selection:
1. Synchronize the chosen version across package manifests and changelog.
2. Stage updated files and amend HEAD: `git commit --amend --no-edit`.
3. Execute package manager publication dry-run (e.g., `dart pub publish --dry-run`).

### Step 4: Publication Readiness Report
Deliver dry-run verification results. If errors or archive leaks occur, supply exact failure diagnostics; otherwise, confirm package is ready to tag and publish.
