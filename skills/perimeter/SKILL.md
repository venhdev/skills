---
name: perimeter
description: "Detect raw runtime API bypasses circumventing internal abstractions and codify enforceable boundary constraints."
disable-model-invocation: true
---

# perimeter — Architectural Boundary & Bypass Detection Engine

## Domain Rubric

### 1. Boundary Audit Dimensions

- **Sanctioned Primitives vs. Raw Bypass (Horizontal)**: Identify internal wrappers governing I/O, network clients, resources, or lifecycles; flag call sites bypassing them for raw runtime APIs.
- **Layer & Dependency Directionality (Vertical)**: Trace import graphs across architectural layers (Domain, App, Infra, UI); flag inverted or cross-boundary imports.
- **Resource & Sandbox Hygiene**: Audit storage paths, concurrency pools, and memory buffers; flag unanchored filesystem access or unthrottled resources.

### 2. The Load-Bearing Qualification Gate

Qualify candidate constraints strictly against 3 conditions:
1. *Existing Sanctioned Primitive*: Codebase defines an internal wrapper, base class, or canonical path addressing the boundary.
2. *Empirical Bypass Evidence*: At least one active call site violates the boundary by calling raw APIs directly.
3. *Quantifiable Failure Risk*: Bypass induces architectural drift, resource leaks, memory exhaustion, or security exposure.
- **Negative Boundary**: Forbid declaring constraints for cosmetic styling, trivial utilities (`Math.*`, collections), or speculative rules lacking active primitives.

## Canonical Output Contract

```markdown
# Perimeter Report: <Target Subsystem / Scope>

## 1. Executive Verdict
- **Target Inspected**: `<directory_or_package>`
- **Verdict**: <HARDENED | PARTIAL | UNPROTECTED> (<N> active boundaries)
- **Primary Risk**: <Single-sentence technical judgment on highest boundary exposure>

## 2. Qualified Boundary Declarations
*(If HARDENED: 'All architectural boundaries are enforced by active test suites; zero raw bypasses detected.')*

### Boundary: <Constraint Name, e.g., Direct Network Client Bypass>
- **Offending Evidence**: [<file>#L<N>](file:///path/to/file#L<N>) ── <Bypass snippet or call site>
- **Failure Risk**: <Concrete risk: memory leak / auth bypass / architectural drift>
- **Rule**: Required: `<Sanctioned Primitive>` ── Banned: `<Forbidden Raw API>`
- **Enforcement**: Tier: `<Arch Unit Test | AST Linter | Script Guard>` | Whitelist: `<Exempt files | 'None'>`

## 3. Disqualified Invariants (Pruned) *(omit if no candidate patterns pruned)*
- **[PRUNED] <Candidate Pattern>**: <Reason for rejection, e.g., No existing codebase primitive / Trivial style issue>

## 4. Pipeline Routing
- `/forge` ── Authorize and implement automated architecture tests for qualified boundaries.
- `/clarify` ── Boundary ownership, whitelist exceptions, or architectural trade-offs require human deliberation.
```

## Execution Protocol

### Phase 1: Macro Anchor Grounding

1. Ingest target directory or subsystem.
2. Scout high-level architectural anchors (internal wrappers, sanctioned clients, resolvers, base interfaces) using symbol and file search tools.
3. Establish grounded anchor context to seed downstream inspection.

### Phase 2: Scale-Adaptive Boundary Audit

1. Evaluate target scope scale:
   - *In-Turn Execution* (< 10 files or localized module): Audit call sites against discovered anchors directly in-turn.
   - *Subagent Fan-Out* (broad codebase or cross-layer scopes): Launch 1–3 `research` subagents (Role: `Perimeter Reviewer (<Slice>)`). Seed each with:
     - Target slice directory.
     - Discovered architectural anchors from Phase 1.
     - The Load-Bearing Qualification Gate and path to `references/fitness-functions.md`.
     - Canonical sub-report contract requiring offending code pointers (`file:///...#L<N>`).

### Phase 3: Codification & Turn-Halt Gate (Fan-in)

1. Reconcile findings into the Canonical Output Contract, moving ungrounded rules to Section 3 (Pruned; omit Section 3 if none).
2. If boundary ownership or whitelist exemptions contain unresolved trade-offs: halt and deliberate (via `/clarify`).
3. Emit the report into the conversation stream and halt turn immediately.
