# Architecture Fitness Functions & Boundary Enforcement

Authoritative reference on architectural constraint families, codification anatomy, and automated enforcement tiers across universal programming environments.

---

## 1. The Two Constraint Families

### Family A: Horizontal Constraints (Banned Raw APIs vs. Sanctioned Primitives)
- **Concept**: Preventing direct utilization of unmanaged runtime or third-party APIs that lack unified governance, telemetry, resilience, or resource optimization.
- **Universal Archetypes**:
  - *Network & Transport*: Banning raw HTTP/RPC clients in favor of authenticated, instrumented wrappers with circuit breakers and tracing.
  - *Heavy Allocations & Decoders*: Banning raw image/file decoders or memory buffers in favor of bounded, downscaling primitives preventing memory exhaustion.
  - *Date & Clocks*: Banning direct calls to non-deterministic system clocks in favor of virtualized clock interfaces for reproducible execution.
  - *Storage & Sandboxing*: Banning raw file path concatenation in favor of path resolvers enforcing application/tenant sandboxes.

### Family B: Vertical Constraints (Layer & Module Boundary Directionality)
- **Concept**: Preserving unidirectional dependency flow and domain isolation across layered or hexagonal architectures.
- **Universal Archetypes**:
  - *Layer Inversion*: Forbidding persistence, database drivers, or network transports from being imported directly inside presentation or domain logic.
  - *Package Encapsulation*: Restricting access to internal packages (`internal/`, `private/`, `_lib/`) outside declared module seams.
  - *Cyclic Couplings*: Preventing circular dependencies between independent domain packages.

---

## 2. The Constraint Codification Anatomy

Every enforceable constraint declaration must define 4 explicit attributes:

1. **Banned (Prohibited Symbol / Import)**:
   - The exact regex, symbol pattern, or import package string that is forbidden.
   - *Example (Horizontal)*: Raw uninstrumented HTTP client instantiation.
   - *Example (Vertical)*: Importing database driver packages inside presentation handlers.
2. **Required (Sanctioned Route)**:
   - The official abstraction, wrapper class, or architectural seam mandated by the project.
   - *Example (Horizontal)*: Internal HTTP client wrapper with telemetry.
   - *Example (Vertical)*: Domain repository interface.
3. **Whitelist (Exemptions)**:
   - The exact file or module allowed to reference the banned symbol (typically the implementation file of the required primitive).
   - *Example*: The internal wrapper module itself.
4. **Failure Risk**:
   - The concrete operational failure avoided by this rule (e.g., *Memory exhaustion from unmanaged buffers*, *Unauthenticated network calls*, *Circular build deadlock*).

---

## 3. The 3-Tier Enforcement Taxonomy & Selection Guide

Choose the enforcement mechanism that provides the tightest feedback loop with the lowest tooling friction:

| Tier | Mechanism | Implementation Style | Best For | Typical Latency |
| :--- | :--- | :--- | :--- | :--- |
| **Tier 1** | **Script Guard** | Shell script (`ripgrep` / regex) in pre-commit hook or CI step | Direct syntax/API bans, zero runtime dependencies, monorepo CI gates | < 100 ms |
| **Tier 2** | **Architecture Unit Test** | Native test runner reading filesystem AST or import paths (`go test`, `pytest`, ArchUnit, test runner) | Layer directionality, package boundary verification, cross-platform CI | 1–3 seconds |
| **Tier 3** | **AST Linter Plugin** | Compiler plugin or static analyzer rule | Instant in-editor squiggly feedback, large engineering teams | Real-time (IDE) |

### Tier Selection Decision Heuristic

Follow this priority order when selecting the enforcement tier:

1. **Default to Tier 2 (Architecture Unit Test)** for 80% of constraints:
   - *Select when*: The project has an active test runner, or the rule evaluates import directionality across layers/packages.
   - *Advantage*: Runs seamlessly across all developer operating systems (Windows, macOS, Linux) without extra tooling.
2. **Select Tier 1 (Script Guard)** ONLY when:
   - The rule is a pure literal string/regex ban (e.g., banning a specific function name or API call).
   - Immediate pre-commit blocking (< 100ms) is required before code reaches Git staging.
   - The target project lacks a unified test runner.
3. **Select Tier 3 (AST Linter Plugin)** ONLY when:
   - The rule requires semantic type resolution (e.g., forbidding a method only on specific object types, where regex produces false positives).
   - The repository already has active custom linter infrastructure (e.g., `eslint-plugin-boundaries`, `custom_lint`).
