---
name: docs-governance
description: Discover documentation authorities in unfamiliar repositories, classify durable changes, and minimally update the appropriate SSOTs, ADRs, specifications, research, runbooks, policy files, and task trackers. Use when Codex must review or change important project documentation, reconcile implementation with documented decisions, locate the document that owns a rule, or remove conflicts and duplication across docs.
---

# Documentation Governance

## Outcome

Discover the repository's documentation authority before editing. Put each
durable fact in the document that owns it, preserve decision history, and avoid
duplicating rules across SSOTs, ADRs, research, runbooks, and task trackers.

## Core Rules

- Treat filenames and conventional folders as evidence, not proof of authority.
- Follow explicit repository precedence and scoped instruction files over this
  skill's generic taxonomy.
- Inspect implementation before documenting implemented behavior.
- Distinguish current state, proposed state, decision rationale, evidence, and
  migration status.
- Store a rule in one authoritative location; link or summarize context
  elsewhere without restating the rule.
- Make the smallest durable update that prevents future ambiguity.
- Preserve read-only snapshots, generated docs, and historical records.
- Do not edit when the user requested only review, explanation, or diagnosis.

## Workflow

### 1. Establish Scope and Instructions

1. Resolve the repository or component placed in scope by the user.
2. Search from the working directory toward the repository root for applicable
   instruction files such as `AGENTS.md`, `CLAUDE.md`, `CONTRIBUTING.md`,
   `RULES.md`, or documented equivalents.
3. Read every instruction file applicable to each target document before
   editing it.
4. Note restrictions such as read-only directories, generated files, required
   templates, approval gates, and validation commands.
5. Do not run repository-disallowed discovery commands merely to locate the
   root. Accept an explicit root or inspect ancestors with filesystem tools.

### 2. Discover Important Documents

Run the bundled read-only inventory when shell access is available:

```bash
bash scripts/discover-docs.sh /path/to/repository
```

Use its output as a candidate list. Also inspect links and indexes referenced by
the applicable instruction files. Search for explicit declarations such as:

- `single source of truth` or `SSOT`;
- `authoritative`, `authority`, or `source of truth`;
- `conflict order`, `precedence`, or `wins over`;
- `supersedes`, `replaced by`, or ADR status markers;
- statements that a file or directory is read-only or generated.

Do not infer that `ARCHITECTURE.md`, `docs/decisions/`, or any conventional name
is authoritative without checking repository declarations and current content.

### 3. Build an Authority Map

Record a compact working map before proposing edits:

| Concern | Authority | Supporting docs | Precedence or mutability |
|---|---|---|---|
| Business behavior | discovered file | related specs | repository rule |
| Current architecture | discovered file | diagrams/indexes | repository rule |
| Decisions and rationale | ADR/RFC location | decision index | local convention |
| Engineering policy | instruction/policy file | contributor docs | scope rule |
| Operations | runbook location | deployment docs | owner rule |
| Research evidence | research location | external sources | snapshot policy |
| Work status | task/roadmap file | migration plan | lifecycle policy |

Use the repository's declared conflict order. Do not impose a universal order
between business, architecture, policy, ADR, and task documents. If authority
remains ambiguous and choosing incorrectly would materially change the result,
report the candidates and ask the user to decide.

### 4. Inspect Reality and Existing Coverage

1. Read the relevant section of every candidate authority, not just search
   snippets.
2. Inspect the implementation, configuration, tests, or operational artifact
   supporting each proposed statement.
3. Search all candidate docs for the same concept and terminology.
4. Identify contradictions, stale claims, duplication, and missing links.
5. Label each fact as one of:
   - implemented and verified;
   - accepted decision not yet implemented;
   - proposal requiring approval;
   - historical or research evidence;
   - unresolved work.

Never describe a proposal as current architecture. Never rewrite an accepted
decision solely because implementation temporarily diverges; surface the
divergence and follow the repository's decision process.

### 5. Classify Each Fact

Use this taxonomy only when the repository does not define a more specific
owner:

| Information | Typical destination |
|---|---|
| Business behavior, domain constraints | Business or product specification |
| Current ownership, boundaries, dependencies, invariants | Architecture SSOT |
| Chosen option, rationale, trade-offs, consequences | ADR or RFC |
| Package, platform, or version constraint | Technology specification |
| Operational command or recovery procedure | Runbook |
| External evidence, experiments, alternatives | Research document |
| Remaining work, gates, migration status | Task tracker or roadmap |
| Directory-scoped implementation rule | Scoped instruction/policy file |

Apply these tests before adding a statement to an SSOT:

- **Durability:** Will this remain useful after the current task is completed?
- **Authority:** Is this file the declared owner of the concern?
- **Normativity:** Does the statement constrain future implementations?
- **Scope:** Is it system-wide, component-local, operational, or temporary?
- **Uniqueness:** Is the same rule already stated authoritatively elsewhere?

If a statement fails durability or uniqueness, omit it or link to its owner.

### 6. Update an SSOT Minimally

- Amend the existing section that already owns the concern.
- Add a section only for a genuinely new concern with durable boundaries.
- Prefer normative language such as `must`, `must not`, `owns`, and `on failure`.
- Capture ownership, dependency direction, ordering, failure semantics, privacy
  boundaries, or lifecycle constraints only when they shape future work.
- Keep vendor-specific mechanics out of provider-neutral architecture unless
  the vendor choice itself is architectural.
- Reuse the repository's terminology and identifiers.
- Replace or consolidate stale text instead of appending a second version.
- Link to detailed evidence, procedures, or decisions rather than copying them.
- Exclude implementation history, test names, console navigation, smoke-test
  steps, and commands unless the target authority explicitly owns them.

### 7. Update ADRs and RFCs Carefully

1. Read the local ADR template, index, naming scheme, status vocabulary, and
   supersession convention.
2. Search for an existing decision covering the same question.
3. Refine the existing record when ownership, provider choice, or core trade-off
   remains the same.
4. Create a new record only for an independent decision or an explicit
   replacement that local conventions require to be separate.
5. Preserve historical identity, original context, and status transitions.
6. Record supersession explicitly; never silently erase a prior decision.
7. Update a status log only when the repository uses one.

Keep roles distinct:

- Architecture states the invariant currently in force.
- ADR/RFC explains why the decision was made and its consequences.
- Research preserves evidence and alternatives.
- Task documents track implementation and verification state.

Do not copy the same normative paragraph into all four.

### 8. Reconcile Cross-Document Consistency

After editing, search again for the affected concept and verify:

- lower-authority documents do not contradict the owner;
- supporting documents link to the authority where clarification is needed;
- accepted-but-unimplemented decisions are labelled accurately;
- task status does not masquerade as architecture;
- research findings have not become policy without a decision;
- renamed or superseded documents have no stale authoritative references.

Remove duplication only when the user's request authorizes editing those files.
Otherwise report it with exact file references.

### 9. Validate

Run the repository-prescribed documentation checks, linters, tests, or audits.
At minimum, verify:

- frontmatter and indexes remain valid;
- internal links and referenced paths resolve where tooling exists;
- documented claims match inspected implementation;
- no generated or read-only artifact was modified;
- the final change contains no unrelated documentation cleanup.

Treat validation failures as unresolved work. Do not claim completion when a
required check has not passed.

## Report the Result

Lead with the documentation outcome. State:

- which authorities were discovered;
- which files changed and why each owns the information;
- which duplication or conflict was removed or intentionally left untouched;
- which validation ran and its result;
- which ambiguity or owner decision remains.

Keep the report self-contained and distinguish automated validation from manual
review gates.

## Discovery Script

`scripts/discover-docs.sh` is read-only. It inventories likely governance docs
and authority declarations while excluding common dependency, build, cache, and
VCS directories. It does not assign authority; interpret its results using the
workflow above.
