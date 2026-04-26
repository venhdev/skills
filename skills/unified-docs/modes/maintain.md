# Maintain Mode

Use this mode when the user asks to update an existing doc, normalize metadata, supersede an ADR, update a plan, or repair dependency/cascade metadata.

## Inputs to load

- `contracts/frontmatter.md`
- `contracts/classification.md` when reclassification is involved
- `contracts/lifecycle.md` for ADR, plan, spec, stale, archive, or replacement behavior
- `contracts/cascade.md` for `depends-on` / `updates` changes
- `templates/reports/mutation-report.md`

## Behavior

1. Treat existing-doc maintenance as the default before creating a replacement.
2. Preserve body content unless lifecycle state requires a visible note, Status Log update, or archive banner.
3. Normalize frontmatter to the contract without discarding valid secondary `kind` values.
4. Keep `reviewCadence` optional; do not add it unless useful for lifecycle/stale tracking or already present.
5. For ADR supersession, create a new ADR, update old metadata and Status Log, and do not rewrite the accepted decision body.
6. For completed plans, promote durable outcomes into a spec before archiving.
7. For cascade repair, patch inverse links only when current and unambiguous.
8. Validate changed docs with `scripts/check_frontmatter.py` when practical.
9. Use `scripts/cascade_targets.py` when cascade metadata changed.

## Safe auto-repair examples

- A current spec lists an explanation doc in `updates`, but the explanation doc clearly depends on that spec and is missing the `depends-on` entry.
- A doc depends on an old path, and the target has a direct current replacement.

## Report instead of auto-repair

- The target is archived or superseded and multiple replacements are plausible.
- The relationship appears intentionally one-way.
- Fixing the issue requires moving files or restructuring folders.
