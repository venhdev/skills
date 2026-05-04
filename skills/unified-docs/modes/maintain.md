# Maintain Mode

Use this mode when the user asks to update an existing doc, normalize metadata, supersede an ADR, update a plan, or repair dependency/cascade metadata.

## Inputs to load

- `contracts/frontmatter.md`
- `contracts/classification.md` when reclassification is involved
- `contracts/doctypes/[type].md` when type-specific lifecycle rules apply
- `contracts/cascade.md` for `depends-on` / `updates` changes
- `templates/reports/mutation-report.md`

## Behavior

1. Treat existing-doc maintenance as the default before creating a replacement.
2. Preserve body content unless lifecycle state requires a visible note, Status Log update, or archive banner.
3. Normalize frontmatter to the contract without discarding valid secondary `kind` values.
4. For cascade checks, use Audit mode when the user wants assessment-only output, including prompts that ask what would be repaired but do not authorize mutation. Use Maintain mode only when repair is explicitly requested. In Maintain mode, patch inverse links only when current and unambiguous.
5. Validate changed docs with `scripts/check_frontmatter.py` when practical. If unavailable, apply the schema rules from `contracts/frontmatter.md` directly: check that all required fields are present, all list-fields use YAML list syntax, type/kind values are valid, and ADR/plan/spec coupling rules are met.
6. When cascade metadata changed, use `scripts/cascade_targets.py` to map outgoing and incoming links. If unavailable, follow the manual cascade check procedure in `contracts/cascade.md`.

## Safe auto-repair examples

- A current spec lists an explanation doc in `updates`, but the explanation doc clearly depends on that spec and is missing the `depends-on` entry.
- A doc depends on an old path, and the target has a direct current replacement.

## Report instead of auto-repair

- The target is archived or superseded and multiple replacements are plausible.
- The relationship appears intentionally one-way.
- Fixing the issue requires moving files or restructuring folders.

## Proactive drift detection

**Trigger**: When the user describes a code change, new implementation, or recently shipped feature — even if they did not explicitly ask about docs.

**When to apply**: User mentions "I just implemented X", "We shipped feature Y", "Code in module Z changed", "I updated the API", or similar implementation/change context (not an explicit doc request).

**Steps**:

1. Identify the feature/component mentioned
2. Search the project for specs, ADRs, or how-tos that cover this area
3. Check cascade graph: do those docs' `depends-on` or `updates` chains reference the changed area?
4. Compare `updated` dates: are they older than the described change?

**If affected docs found**:
- Surface them: "These docs may be affected by your change: [X, Y]. Should I review and update them?"
- Do NOT auto-update without confirmation
- Wait for user decision before making any mutations

**If no affected docs found**:
- Note it: "No docs appear to cover [feature]. Consider creating a spec or TIL if this is durable behavior."

**Intent**: Code-first developers are reminded about docs without needing to remember "update docs after pushing code".

## Reorganization

**Trigger**: User explicitly says "reorganize", "restructure docs", "fix structure", or similar.

When reorganization is requested:
1. Load `workflows/audit-org/organization-patterns.md`
2. Inventory current structure (list all doc files and folders)
3. Detect red flags from the patterns document
4. Recommend a pattern from the decision matrix based on team size and doc count
5. Present the recommendation and ask for confirmation before any mutation
6. If confirmed, follow Migration Checklist in `organization-patterns.md`
7. Report all moved files and updated cross-references in mutation report

**Important**: Never move or rename files without explicit user confirmation of the target structure.

## Arg-triggered routing

| Arg | Workflow |
|---|---|
| `--maintain-plan` | `workflows/maintain-plan/` |
