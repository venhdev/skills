# Roadmap: `inspect-*` Suite

Read-only inspection and auditing skills (Archetype B) evaluating codebase health and semantic architectural risks before refactoring or change planning.

---

## 1. Skill Roster (5 Orthogonal Pillars)

| Skill | Status | Target Scope | Core Semantic Focus |
|---|---|---|---|
| [`inspect-structure`](../skills/inspect-structure/SKILL.md) | **Active** | Code Structure & OOP | God classes, cross-file DRY violations, cognitive complexity, coupling bleed (Feature Envy). |
| [`inspect-role-naming`](../skills/inspect-role-naming/SKILL.md) | **Active** | Role & Nomenclature | Role suffix fidelity, wire-domain casing isolation, framework primitive collisions, CQS booleans. |
| [`inspect-perf`](../skills/inspect-perf/SKILL.md) | **Active** | Performance & Resources | ORM N+1 query patterns, event-loop starvation, unbounded in-memory cache retention, algorithmic hot-paths. |
| [`inspect-security`](../skills/inspect-security/SKILL.md) | **Active** | Application Security | Semantic auth flaws (IDOR/BOLA), multi-tenant isolation leaks, check-then-act race conditions, cross-boundary taint sinks. |
| `inspect-migration` | Planned | Database Migrations | Zero-downtime violations (Expand-Contract), exclusive table locks on deployment. |

---

## 2. Roster Convergence & Boundary Rationales

Following the architectural distillation of the suite (enforcing *ZenForge Construction Rules § 2: Tool-Aware Zero-Waste & Anti-No-Op*):

- **Concurrency Convergence**: Throughput bottlenecks, event-loop lag, and thread pool exhaustion are fully governed by [`inspect-perf`](../skills/inspect-perf/SKILL.md) (Dimension 3). Semantic race conditions (check-then-act, TOCTOU) are absorbed into `inspect-security`. Standalone `inspect-concurrency` is merged to eliminate cognitive routing friction.
- **API Hygiene Boundary**: Data-tier pagination and mutation idempotency keys are governed by [`inspect-perf`](../skills/inspect-perf/SKILL.md) (Dimensions 1 & 4); cross-boundary wire casing is governed by [`inspect-role-naming`](../skills/inspect-role-naming/SKILL.md). Contract drift is addressed by `/assay`.
- **Dependency Tooling Delegation**: Package vulnerabilities and copyleft license checks are strictly delegated to native CLI tooling (`npm audit`, `cargo audit`, `trivy`); Clean Architecture boundary violations remain governed by [`/perimeter`](../skills/perimeter/SKILL.md). Standalone `inspect-deps` is pruned as redundant.

---

## 3. Core Invariants

- **Archetype B Exclusivity**: Audit and catalog only; forbid modifying files on disk. Route remediation to [`/simplify-audit`](../skills/simplify-audit/SKILL.md) (behavior-preserving simplification audit/staging) or [`/changeset`](../skills/changeset/SKILL.md) (macro-level).
- **Semantic Over Mechanical**: Target multi-file reasoning and architectural cohesion; leave AST formatting, type checking, and regex scanning to native CLI tooling.
