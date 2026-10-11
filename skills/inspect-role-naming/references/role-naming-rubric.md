# Architectural Role & Nomenclature Rubric (SSOT)

> [!IMPORTANT]
> **Project Authority Priority Rule**: The host project's existing architectural standards, repository conventions, and established codebase patterns **ALWAYS take precedence** over this rubric. This document serves as a foundational baseline and fallback reference when project-specific conventions are unstated, incomplete, or ambiguous.

---

## 1. Architectural Role Suffix Matrix

Classes and types generally declare their primary architectural responsibility via standard role suffixes. Pure business domain concepts (`Order`, `User`, `Account`) remain clean domain nouns without technical persistence suffixes (`*Entity`, `*VO`).

| Role Suffix | Architectural Responsibility & Scope | Recommended Behaviors | Smells to Avoid (Role Dishonesty) |
|---|---|---|---|
| **`*Repository`** | Collection-oriented mediation between domain aggregates and data persistence. | Querying, ORM/DB access, cache retrieval, entity reconstitution. | Performing out-of-process HTTP/vendor calls, sending email/push notifications, view formatting. |
| **`*Gateway`** | Encapsulation of external, third-party partner APIs or remote systems (Stripe, Twilio, CRM). | Network transport calls, protocol translation, external error normalization. | Direct database table querying, internal transaction management, persisting domain entities. |
| **`*Service`** | Application use-case coordination (Application Service) or multi-entity business logic (Domain Service). | Orchestrating repositories and gateways, transaction boundaries, domain rules. | Raw socket/HTTP handling, presentation rendering, inline raw SQL scripts. |
| **`*Controller`** | Inbound presentation adapter translating transport requests into application commands/queries. | Param validation, header parsing, returning transport status codes. | Inlining business domain logic, raw SQL queries, calling external vendor APIs directly. |
| **`*Policy`** | Encapsulation of business rules, validation criteria, or access quotas (Stateless Strategy). | Pure calculation, evaluation rules, boolean decisions, quota checks. | Performing side effects, network I/O, database writes, state mutations. |
| **`*Engine`** | High-algorithmic-density compute pipelines, state machines, or rule evaluation pipelines. | Complex state transitions, pricing algorithms, workflow execution. | Direct database persistence; delegates state persistence to Repositories. |
| **`*Worker`** | Asynchronous loop or message queue lifecycle coordinator. | Polling, message consumption, backoff, retry, acknowledgment. | Embedding core domain calculations directly in message ack loops. |
| **`*Adapter`** | Structural bridge implementing a port interface in Hexagonal/Clean Architecture. | Protocol conversion, third-party library translation. | Application use-case orchestration, core business invariant logic. |
| **`*Dto`** | Pure passive data container across process or network serialization boundaries. | Primitive fields, serialization annotations, boundary validation constraints. | Encapsulating business rules, database calls, stateful domain methods. |

---

## 2. Catch-All Suffix Antipatterns & Focused Alternatives

Generic suffixes often lack semantic boundaries and can mask God classes.

| Ambiguous Suffix | Common Structural Issue | Recommended Alternative |
|---|---|---|
| **`*Manager`** | Tends to accumulate disparate concerns over time. | Decompose by responsibility: `UserRepository` (persistence), `UserPolicy` (rules), `UserGateway` (external API). |
| **`*Helper` / `*Util`** | Low cohesion, risks cyclic dependencies across modules. | Colocate logic with domain entities, or extract focused single-task subroutines (`*Codec`, `*Parser`). |
| **`*Processor`** | Ambiguous intent (queue worker, state machine, or mediator). | Clarify role: `*Worker` (queue loop), `*Engine` (algorithm), or `*Handler` (mediator event). |

---

## 3. Cross-Boundary Casing & Schema Isolation

Maintain clean separation between external serialization formats and internal language identifiers, adhering to the project's established conventions.

### A. Boundary Casing Consistency & Convention Alignment
Different architectures adopt different conventions across tiers (e.g. `snake_case` in REST/PostgreSQL, `camelCase` in GraphQL/Node.js, or `PascalCase` in .NET/MS SQL). Production best practice focuses on **Consistency & Boundary Isolation**:
- **Wire Payloads & APIs**: Follow the established project API contract consistently (e.g. `snake_case` or `camelCase`).
- **Database & Storage Schemas**: Align with the project's persistence schema conventions without mixing conflicting casing schemes within the same database layer.
- **Internal Language Identifiers**: Adhere to the host language's idiomatic style (`camelCase` in TS/Java/Dart/Go, `snake_case` in Python/Rust, `PascalCase` in C#).
- **Core Principle**: Avoid leaking external wire or storage casing directly into internal domain models. Isolate external schemas at the boundary using DTOs, mappers, or serialization annotations.

### B. Anti-Corruption Boundary Practices
1. **Prefer Explicit Boundary DTOs**: Avoid exposing internal domain models or ORM entities directly over public network boundaries.
2. **Explicit Field Binding**: Annotate boundary DTOs with explicit serialization keys when crossing formats (e.g. `@JsonProperty("account_id")`, `json:"account_id"`).
3. **Guard Against Silent Deserialization Mismatches**: Ensure parsers reject or flag unmapped casing drift during contract testing to prevent uninitialized null/zero-value objects.

---

## 4. Framework & Runtime Collision Prevention

Domain entities should avoid shadowing framework, runtime, or ambient language primitives to prevent auto-import ambiguity.

### A. Shadowing Hotspots
Be cautious when naming domain models with bare primitives:
- **Framework Lifecycles & UI**: `State`, `Container`, `Context`, `Action`, `Event`, `Route`, `Component`.
- **Language & Runtime Primitives**: `Record`, `Model`, `Object`, `Node`, `Error`.

### B. The Ambient Shadowing Risk (TypeScript / Web)
In TypeScript/DOM environments, naming an entity `Event` or `Record` can cause missing imports to fall back silently to ambient globals in `lib.dom.d.ts`, passing compilation but failing at runtime.

### C. Recommended Domain Qualification
Qualify business concepts with domain-specific context:
- *Rather than bare `State`* ──> `OrderPaymentState` or `SessionState`.
- *Rather than bare `Container`* ──> `CargoContainer` or `LayoutContainer`.
- *Rather than bare `Event`* ──> `AuditLogEvent` or `DomainEvent`.

---

## 5. Namespace Purity & Prefix Boundaries

- **Avoid Obsolete Encodings**: Prefer standard language naming over Hungarian prefixes (`IUser`, `m_name`, `s_instance`) unless dictated by project convention (e.g., standard C# interface guidelines).
- **Avoid Redundant Stuttering**: Avoid repeating package or module names in exported identifiers where the package qualifier already provides context (`auth.Service` over `auth.AuthService`).
- **Project Brand Boundaries**: Organizational or brand prefixes are typically confined to:
  - Root package namespaces: `@company/package-name`.
  - Design system tokens / CSS classes: `--brand-color-primary`.
  - Public client SDK entrypoints: `BrandClient`.
  - *Internal domain models and repositories generally remain clean business nouns.*

---

## 6. Linguistic Action-Effect Consistency & Affirmative Booleans

### A. Command-Query Consistency
- **Query Operations (`get*`, `find*`, `calculate*`, `is*`, `has*`)**: Primarily read-only inquiries; avoid coupling hidden mutations, persistent writes, or remote side-effects into queries.
- **Mutation Operations (`create*`, `update*`, `delete*`, `sync*`, `persist*`)**: Use imperative command verbs to declare state modification.
- **Combined Intent**: If an operation both finds and mutates, declare it explicitly (`getOrCreateUser()`).

### B. Affirmative Predicate Booleans
- Prefer positive modal verbs for boolean properties and predicates: `is*`, `has*`, `can*`, `should*`.
- **Avoid Negative Markers**: Avoid negative prefixes (`isNot*`, `isDisabled`, `noCache`) where they force double-negative logic inversions (`!isDisabled`). Prefer positive phrasing (`isEnabled`, `useCache`).
