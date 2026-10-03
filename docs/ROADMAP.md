# Roadmap: `inspect-*` Suite

Read-only inspection and auditing skills (Archetype B) evaluating codebase health and semantic architectural risks before refactoring or change planning.

---

## 1. Skill Roster

| Skill | Status | Target Scope | Core Semantic Focus |
|---|---|---|---|
| [`inspect-code`](file:///home/rirong/src/owner/my-skills/skills/inspect-code/SKILL.md) | **Active** | Code Quality & OOP | God classes, cross-file DRY violations, cognitive complexity, coupling bleed (Feature Envy). |
| `inspect-security` | Planned | Application Security | Semantic auth flaws (IDOR/BOLA), multi-tenant data leaks, cross-boundary taint sinks. |
| `inspect-perf` | Planned | Performance & Resources | ORM N+1 query patterns, event-loop starvation, unbounded in-memory cache retention. |
| `inspect-api` | Planned | API & Interface Hygiene | Missing pagination, missing idempotency keys on mutations, breaking contract drift. |
| `inspect-concurrency` | Planned | Async & Concurrency | Check-then-act races, lock order inversions, zombie task/goroutine lifecycle leaks. |
| `inspect-migration` | Planned | Database Migrations | Zero-downtime violations (Expand-Contract), exclusive table locks on deployment. |
| `inspect-deps` | Planned | Dependency Hygiene | Domain layer contamination, copyleft license risks, micro-package/abandoned bloat. |

---

## 2. Core Invariants

- **Archetype B Exclusivity**: Audit and catalog only; forbid modifying files on disk. Route remediation to [`/simplify`](file:///home/rirong/src/owner/my-skills/skills/simplify/SKILL.md) (micro-level) or [`/changeset`](file:///home/rirong/src/owner/my-skills/skills/changeset/SKILL.md) (macro-level).
- **Semantic Over Mechanical**: Target multi-file reasoning and architectural cohesion; leave AST formatting, type checking, and regex scanning to native CLI tooling.
