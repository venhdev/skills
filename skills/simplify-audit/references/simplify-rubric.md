# Simplification Rubric

Evaluate worthwhile simplification against the following dimensions; classify candidates by their load-bearing effect.

1. **Control Flow (Block Hierarchy)**
   - Smell: Deep nesting that obscures the path, arrow anti-pattern, trailing else blocks.
   - Remedy: Invert conditions into guard clauses and early returns.
   - Guardrail: Preserve simple 1-level conditionals and natural linear flow.

2. **Predicate Logic (Boolean Expressions)**
   - Smell: Compound conditions that obscure domain meaning, double negatives.
   - Remedy: Extract explanatory boolean variables, apply De Morgan's laws.
   - Guardrail: Keep obvious concise checks (e.g., existence or status validation) inline.

3. **Function Cohesion (Vertical Scope)**
   - Smell: Functions mixing high-level orchestration with low-level mechanics (parsing, formatting, regex).
   - Remedy: Extract low-level mechanics into focused single-task helper functions.
   - Guardrail: Preserve cohesive linear logic; extract only when the helper clarifies a real responsibility.

4. **Code Repetition (Horizontal Duplication)**
   - Smell: Repeated load-bearing logic with the same contract and rule ownership.
   - Remedy: Consolidate into a shared helper or import the canonical owner; preserve state verification at each transaction boundary.
   - Guardrail: Keep logic separate when coincidental similarity spans independent policy owners; avoid boolean-flag helpers coupling unrelated change vectors.

5. **Expression Density (Inline Syntax)**
   - Smell: Nested ternary expressions or chained inline conditionals, multiple statements packed onto one line.
   - Remedy: Unpack into explicit multi-line conditional or pattern-matching statements.
   - Guardrail: Preserve readable single-line binary conditionals (e.g., simple fallback or value selection).

6. **Identifier Naming (Lexicon)**
   - Smell: Deceptive, misleading, or cryptic single-character variable names outside loop counters (`i, j`).
   - Remedy: Rename identifiers to reveal domain intent clearly.
   - Guardrail: Preserve standard conventions (`err`, `ctx`, `req`, `res`, `i`); never rename public API parameters.

7. **State Mutation (Variable Mutability)**
   - Smell: Reassigning temporary variables across disparate scopes, reusing flags for unrelated states, or mutating function parameter bindings.
   - Remedy: Split into discrete single-assignment or immutable variables reflecting distinct lifecycle values.
   - Guardrail: Preserve standard loop accumulators (`sum += x`), stream buffers, and performance-critical in-place mutations.

8. **Magic Literals (Literal Obfuscation)**
   - Smell: Hardcoded magic numbers, status strings, or configuration thresholds embedded directly inside operational logic.
   - Remedy: Import the canonical constant owner when one exists; extract to a private constant only when the value belongs to local logic.
   - Guardrail: Keep self-evident literals (`0`, `1`, `""`, `-1`) inline and constants private unless the approved design needs a public contract.

9. **Dead & Superseded Logic (Zombie Pruning)**
   - Smell: Unreachable code, uncalled private helpers, unused parameters after refactoring, or tests testing discarded mechanisms.
   - Remedy: Delete cleanly; never retain dead code or commented-out blocks.
   - Guardrail: Preserve interface contracts and polymorphic hooks with a confirmed consumer or active requirement; remove speculative extension points without demonstrated need.
