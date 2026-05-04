# Create Mode

Use this mode when the user wants a new doc created from scratch or a rough note promoted into a structured document.

## Inputs to load

- `contracts/frontmatter.md`
- `contracts/classification.md`
- `contracts/doctypes/[type].md` if the doc is a plan, spec, or ADR
- `contracts/cascade.md` if dependencies or downstream docs matter
- all files in `workflows/create-plan/` if the doc is a plan (two-tier planning workflow)
- matching file in `templates/authoring/` (or `workflows/create-plan/` for plan)

## Behavior

**Step 0: Context scan** (before any discovery question)

When a Create request arrives, first scan the project's existing docs:
1. Search for docs related to the topic/keywords mentioned in the request
2. If relevant docs found: surface them — "Found related: [X, Y]. Is this an update to one of these, or a new doc?"
   - If update → switch to Maintain mode immediately
   - If new doc → treat found docs as known context; skip discovery questions about existence ("What docs already exist on this topic?")
3. If nothing found: note explicitly ("No existing docs on this topic found"), then proceed with standard discovery
4. If no existing docs folder or convention exists (from-scratch project): suggest a starting folder structure pattern. Ask the user which pattern fits before placing the new doc. This is a hint, not a discovery gate — do not block authoring

1. **MANDATORY: Enter discovery before authoring for every Create request. Never skip or defer discovery.**
2. Ask one focused question at a time. Even apparently clear requests are not exempt. A clear title or explicit type hint does NOT satisfy the discovery requirement.
3. Use each answer to reduce ambiguity around problem, users, success, scope, constraints, dependencies, gate conditions, and artifact choice.
4. Continue discovery until you can restate the request without needing assumptions.
5. Restate your understanding and ask the user to confirm before writing.
6. Only after confirmation, classify the target doc and choose the matching authoring template.
7. Produce the authored document artifact itself as the primary output. Do not wrap the created doc inside a mutation report unless the user explicitly asks for a report in addition to the doc.
8. Preserve full-skeleton structure unless the request clearly needs less.
9. Include `depends-on: []` and `updates: []` even when empty.
10. For `how-to` and ordinary `explanation` docs, leave `kind` empty unless the request clearly requires a lifecycle role such as `plan` or `til`. Do not infer `kind: [spec]` for procedural guidance.
11. Follow existing project placement conventions first; if unclear, use standard docs structure or ask before placing the file.
12. Avoid copying canonical content from existing SSOT docs; link instead.
13. For plan docs, keep Gate conditions optional; after creating a plan, add a brief hint that the user may add gate conditions under Milestones if implementation should wait for prerequisites.
14. If the user asks for both a doc and a short summary, emit the doc first, then a compact report.

## Discovery gate

**This gate is non-negotiable and applies to ALL Create requests without exception.**

Discovery applies even when:
- The user provides a specific title
- The user names a doc type explicitly (e.g., "create a plan", "write a spec")
- The request seems obvious or straightforward
- The user has sketched out rough content
- The user is working on a follow-up to prior work

The skill must assume there may be hidden instability even when the request looks clear. Do not rationalize or skip questioning based on apparent clarity.

**Authoring is blocked until all of the following are true:**

- at least one discovery question has been asked;
- the skill understands the problem being solved;
- intended users or stakeholders are clear enough to write for;
- success criteria and scope boundary are clear enough to avoid writing the wrong doc;
- any material constraint, dependency, milestone-order issue, or gate condition that would change the document shape is clear;
- the chosen artifact type is justified by discovery rather than assumed;
- the user has confirmed the restatement.

Use the next question that most reduces ambiguity. Prefer questions that can change:

- the intended outcome;
- who the document is for;
- what is in scope or out of scope;
- whether the artifact should be a plan, spec, ADR, how-to, explanation, or a split set of docs;
- whether durable truth belongs in a spec;
- whether a technical decision requires an ADR first.

Do not carry unresolved ambiguity into the authored doc as assumptions, open questions, deferred questions, or placeholder certainty. If a fact is not known well enough to write safely, ask another question instead.

Before writing, use a compact restatement such as:

```markdown
I understand this request as:
- Problem: ...
- Success: ...
- Scope: ...
- Intended artifact: ...

Is this correct?
```

If the user changes or rejects the restatement, continue discovery.

## Create-specific guidance

- For plan requests, use discovery to confirm the intended outcome, measurable scope boundary, milestone shape, and whether the request is actually a spec or ADR problem first.
- For spec requests, use discovery to confirm the durable truth the spec should own and what belongs in related ADR or plan docs instead.
- For ADR requests, use discovery to confirm the decision to be made, alternatives worth recording, and whether the decision is actually stable enough for an ADR.
- For how-to or explanation requests, use discovery to confirm task/outcome and whether the request is actually asking for durable reference truth instead.
- For source-material conversions, do not normalize raw material into a doc until discovery has established the intended outcome and correct artifact shape.
- For large features, decide after discovery whether one doc is sufficient or whether a root index plus sub-docs is the better artifact shape.
- When a temporary plan references durable specs or SSOT docs that may need updates after completion, put those durable docs in the plan's own `updates`; do not add the temporary plan to the durable docs' `updates` or `depends-on` as a reciprocal link.
- For placement decisions when conventions are unclear: consult `contracts/organization.md` to recommend a pattern. Favor Pattern 1 (By Type) for small teams, Pattern 3 (Hybrid) for growing teams. Always confirm with user before recommending folder restructuring.

## Template mapping

- plan -> `workflows/create-plan/implementation-plan-template.md`
- spec -> `templates/authoring/spec.md`
- ADR -> `templates/authoring/adr.md`
- TIL -> `templates/authoring/til.md`
- how-to -> `templates/authoring/how-to.md`
- explanation -> `templates/authoring/explanation.md`

## Reporting

Use `templates/reports/mutation-report.md` only when the user explicitly asks for a report or summary in addition to the authored document.
