---
name: simplify
description: "Refactor complex, bloated, or deeply nested code into clean and idiomatic implementations while preserving exact functional behavior."
disable-model-invocation: true
---

# simplify — Code Clarity & Refactoring Engine

Refactor complex, bloated, or deeply nested code into clean, readable, and idiomatic implementations while preserving exact functional behavior.

## Operating Invariants

- **Behavioral Invariance**: Preserve 100% of existing functionality, test outcomes, error handling, and public API signatures; forbid altering program semantics or breaking contracts.
- **Scope Discipline**: Confine refactoring strictly to targeted files or functions; forbid unsolicited churn in untouched surrounding code.
- **Pre-Mutation Gate**: Stage changes exclusively via Changeset summary and halt turn immediately; forbid writing mutations to disk without affirmative human authorization.
- **Verification Circuit Breaker**: Stop execution immediately upon verification failure, report stderr with file citations, and prompt user whether to revert or keep debugging; forbid silent lossy reversions or unguided retry loops.

## Refactoring Rubric

Evaluate target code against 6 mutually exclusive structural dimensions:

1. **Control Flow (Block Hierarchy)**
   - Smell: Nesting ≥ 3 levels deep, arrow anti-pattern, trailing else blocks.
   - Remedy: Invert conditions into guard clauses and early returns.
   - Guardrail: Preserve simple 1-level conditionals and natural linear flow.

2. **Predicate Logic (Boolean Expressions)**
   - Smell: Compound conditions (≥ 3 clauses), double negatives.
   - Remedy: Extract explanatory boolean variables, apply De Morgan's laws.
   - Guardrail: Keep obvious 1–2 clause conditions (`user && user.isActive`) inline.

3. **Function Cohesion (Vertical Scope)**
   - Smell: Functions mixing high-level orchestration with low-level mechanics (parsing, formatting, regex).
   - Remedy: Extract low-level mechanics into focused single-task helper functions.
   - Guardrail: Forbid trivial micro-extractions used once; preserve cohesive linear logic.

4. **Code Repetition (Horizontal Duplication)**
   - Smell: Identical copy-pasted blocks (≥ 3 occurrences) within the target file.
   - Remedy: Consolidate common logic into a single private helper.
   - Guardrail: Never merge accidental similarities with boolean flag parameters; tolerate minor repetition over coupled abstractions.

5. **Expression Density (Inline Syntax)**
   - Smell: Nested ternaries (`a ? b : c ? d : e`), multiple statements packed onto one line.
   - Remedy: Unpack into explicit multi-line `if/else` or pattern-matching statements.
   - Guardrail: Preserve readable single-line binary ternaries (`isValid ? a : b`).

6. **Identifier Naming (Lexicon)**
   - Smell: Deceptive, misleading, or cryptic single-character variable names outside loop counters (`i, j`).
   - Remedy: Rename identifiers to reveal domain intent clearly.
   - Guardrail: Forbid cosmetic churn on standard conventions (`err`, `ctx`, `req`, `res`, `i`); never rename public API parameters.

## Execution Protocol

### Phase 1: Inspect & Clean-Pass Check

1. Inspect the target file, diff, or function using non-mutating file tools.
2. Evaluate target code against the Refactoring Rubric.
3. **Clean-Pass Exit**: If zero code smells breach the rubric or all candidate refactors violate guardrails, emit this assessment and halt turn immediately:

   ```text
   Assessment: Target code is already clean, idiomatic, and minimal. No refactoring required.
   ```

### Phase 2: Stage Changeset & Lean Delivery

1. Stage planned modifications in Changeset format:

   ```text
   # Changeset: Simplify <Target Scope>

   📁 <subsystem_or_directory>/
   └── 📄 <filename>
       └── [UPDATE] Simplify <component_or_function>:
           • <Dimension applied, e.g., Flatten nested conditionals via guard clauses>.
           • <Dimension applied, e.g., Extract low-level parsing into helper function>.
   ```

2. **Lean Delivery**: Present strictly the high-density Changeset summary and technical rationale. Forbid dumping voluminous raw diffs into chat by default.

### Phase 3: Authorization Gate

1. Present the Changeset summary and offer execution options:
   - Approve applying changes directly.
   - Request to inspect the full unified diff preview first.
   - Adjust scope or cancel.
2. Forbid modifying workspace files on disk or executing mutating commands within this turn.
3. Halt turn immediately and wait for explicit human authorization (e.g., 'proceed', 'approved').
4. If the user requests to see the diff preview, render the unified diff and halt turn again for final approval.

### Phase 4: Mutate & Verify

1. Upon receiving approval, apply edits to disk.
2. Run existing tests, linters, or syntax checks to verify behavioral invariance.
3. If verification fails, stop, report the error output, and ask the user whether to revert or keep debugging.
