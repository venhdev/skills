# Create Mode

Use this mode when the user wants a new doc created from scratch or a rough note promoted into a structured document.

## Inputs to load

- `contracts/frontmatter.md`
- `contracts/classification.md`
- `contracts/lifecycle.md` if the doc is a plan, spec, or ADR
- `contracts/cascade.md` if dependencies or downstream docs matter
- matching file in `templates/authoring/`

## Behavior

1. Classify the target doc before writing.
2. Choose the matching authoring template.
3. Produce the authored document artifact itself as the primary output. Do not wrap the created doc inside a mutation report unless the user explicitly asks for a report in addition to the doc.
4. Preserve full-skeleton structure unless the request clearly needs less.
5. Include `depends-on: []` and `updates: []` even when empty.
6. Include `reviewCadence` by default in plan/spec/ADR templates only.
7. For `how-to` and ordinary `explanation` docs, leave `kind` empty unless the request clearly requires a lifecycle role such as `plan` or `til`. Do not infer `kind: [spec]` for procedural guidance.
8. Follow existing project placement conventions first; if unclear, use standard docs structure or ask before placing the file.
9. Avoid copying canonical content from existing SSOT docs; link instead.
10. For plan docs, keep Gate conditions optional; after creating or updating a plan, add a brief hint that the user may add gate conditions under Milestones if implementation should wait for prerequisites.
11. If the user asks for both a doc and a short summary, emit the doc first, then a compact report.

## Intake-informed plan creation

Use this branch in exactly two cases: (1) the user explicitly marks an idea with `--intake`; or (2) the user provides source material such as extracted Markdown/text from a PDF, DOCX, API JSON, partner spec, meeting notes, imported requirements, or another attached document, and asks this skill to create or convert documentation from it. Do not trigger intake for ordinary notes or requests that lack both an intake flag and source material.

Default output is Plan + proposal:

1. Create a primary `kind: [plan]` artifact when the feature is small enough to stay readable.
2. If the plan for one large feature would exceed roughly 500 lines, create a root index plan and split execution detail into linked sub-plans.
3. Recommend follow-up specs, ADRs, how-tos, or API references, but do not create them unless requested.

Before writing, classify intake blocks into requirements, constraints, milestones, dependencies, risks, open questions, and follow-up docs. For `--intake` user ideas, clarify outcome, users, scope boundaries, success criteria, constraints, risks, dependencies, and whether the output should be one plan, split sub-plans, a spec, an ADR, or only follow-up recommendations before authoring. If the prompt already answers some dimensions, do not ask them again; ask only narrow remaining blocking questions, and record non-blocking unknowns as open questions. For source-material intake, ask concise blocking questions before creating only when a gap or conflict affects goal, scope, milestone order, gate conditions, ownership, go/no-go direction, ADR-level decisions, or whether durable behavior belongs in a spec. Do not turn blockers into assumptions, and do not invent operational, compliance, data-source, or measurement details that the intake did not provide.

For intake plans, add compact sections when relevant:

- `## Source materials`
- `## Assumptions`
- `## Open questions`
- `## Recommended follow-up docs`

Keep these sections brief. Do not paste raw source material into the generated plan.

For large-feature splits, the root index plan keeps shared goal, scope, source materials, high-level milestones, dependencies, risks, and links to sub-plans. Each sub-plan owns one coherent execution slice and links back to the root plan through cascade metadata or an explicit related-doc link. When an intake plan references durable specs or SSOT docs that may need updates after completion, put those durable docs in the plan's own `updates`; do not add the temporary plan to the durable docs' `updates` or `depends-on` as a reciprocal link.

## Template mapping

- plan -> `templates/authoring/plan.md`
- spec -> `templates/authoring/spec.md`
- ADR -> `templates/authoring/adr.md`
- TIL -> `templates/authoring/til.md`
- how-to -> `templates/authoring/how-to.md`
- explanation -> `templates/authoring/explanation.md`

## Reporting

Use `templates/reports/mutation-report.md` only when the user explicitly asks for a report or summary in addition to the authored document.
