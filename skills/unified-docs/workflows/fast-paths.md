# Fast Paths

Use these paths before loading heavier references.

## Quick read/status

Use when the user asks whether one doc is current, what replaced it, or what to read.

1. Read target frontmatter only.
2. If `status: superseded`, read replacement frontmatter only.
3. Report current?, replacement, direct `updates`, and obvious stale dependency if visible.
4. Do not scan corpus.
5. Do not run scripts.
6. Read body only if the user asks for summary/current guidance.

## Assess-only triage

Use for audit/check/review unless user says full/complete/entire corpus.

1. Inspect requested doc(s) and directly linked docs only.
2. Check contract, missing targets, superseded dependencies, and direct cascade gaps.
3. Report max 5 findings, critical first.
4. Do not mutate files.
5. Do not run scripts by default; use direct reads/searches first.

## Assess+repair

Use when user asks fix/update/repair/create/supersede.

1. Patch only requested or directly required files.
2. Validate changed docs with `check_frontmatter.py`.
3. Run `cascade_targets.py` only for changed docs or directly linked cascade docs.
4. Report changed files, validation, and unresolved followups.

## Full audit escalation

Only for explicit full/complete/entire corpus requests.

1. Discover corpus.
2. Run script checks as useful.
3. Rank findings by severity.
4. Keep detailed findings capped unless user asks for exhaustive output.
