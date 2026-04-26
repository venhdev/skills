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
10. If the user asks for both a doc and a short summary, emit the doc first, then a compact report.

## Template mapping

- plan -> `templates/authoring/plan.md`
- spec -> `templates/authoring/spec.md`
- ADR -> `templates/authoring/adr.md`
- TIL -> `templates/authoring/til.md`
- how-to -> `templates/authoring/how-to.md`
- explanation -> `templates/authoring/explanation.md`

## Reporting

After creating docs, use `templates/reports/mutation-report.md`.
