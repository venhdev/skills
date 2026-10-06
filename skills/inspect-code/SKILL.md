---
name: inspect-code
description: "Audit source code modules for structural smells, God classes, and duplication anti-patterns before refactoring."
disable-model-invocation: true
---

# inspect-code — Code Quality & Structural Debt Inspector

## Domain Rubric

### 1. Scope & Operational Boundaries

- **Structural Debt vs. Syntax Formatting**: Confine analysis strictly to structural, OOP, and cognitive debt across module, class, and method tiers; defer whitespace, semicolons, and styling nits entirely to native linters.
- **Architectural Form vs. Business Logic**: Audit structural cohesion and maintainability exclusively; never evaluate domain business rules, pricing logic, or math formulas without authoritative specs (route bug isolation to `/diagnose`; route spec drift to `/assay`).
- **Read-Only Audit vs. Code Mutation**: Deliver structured inspection dossiers only; forbid modifying files on disk, generating code diffs, or applying automatic refactorings (route remediation planning to `/changeset`).

### 2. Structural Inspection Dimensions

1. **Class Cohesion & SRP (God Class)**
   - Smell: Class aggregating multiple disparate domain concerns, bloated field/method counts, or low method cohesion.
   - Remedy: Decompose into focused single-responsibility services, value objects, or strategy delegates.
   - Micro-Snippet:
     ```text
     Smell:  class OrderManager { calculateTax(); sendEmail(); saveToDb(); renderPdf(); }
     Remedy: class OrderService delegates to TaxCalculator, OrderMailer, OrderRepository, PdfRenderer.
     ```

2. **Method & Flow Complexity (Long Method / Deep Nesting)**
   - Smell: Methods exceeding manageable length, control-flow nesting ≥ 3 levels deep, arrow anti-pattern, or high cognitive load.
   - Remedy: Invert conditions into guard clauses, flatten nested blocks, and decompose into focused single-task helper subroutines.
   - Micro-Snippet:
     ```text
     Smell:  void process() { if (a) { while (b) { if (c) { try { ... } } } } }
     Remedy: Use guard clauses for early returns; extract loop bodies into private subroutines.
     ```

3. **Systemic Duplication (Macro & Micro DRY)**
   - Smell: Copy-pasted algorithmic routines, multi-step validation sequences, or error handling pipelines spanning multiple classes, files, or methods.
   - Remedy: Centralize into shared domain helpers, base abstractions, or reusable middleware.
   - Micro-Snippet:
     ```text
     Smell:  ServiceA and ServiceB replicate identical 20-line auth header parsing and claim verification.
     Remedy: Extract to TokenAuthenticator.validateAndExtractClaims().
     ```

4. **Coupling & Encapsulation Bleed**
   - Smell: Methods manipulating internal data of external classes (Feature Envy); classes exposing mutable internal collections directly via getters.
   - Remedy: Apply "Tell, Don't Ask"; encapsulate state mutations behind domain methods; return immutable snapshots or defensive copies.
   - Micro-Snippet:
     ```text
     Smell:  if (order.getCustomer().getAddress().getCountry().equals("US")) { ... }
     Remedy: if (order.isDomesticDelivery()) { ... }
     ```

5. **Polymorphism & Strategy Deficits (Scattered Branching)**
   - Smell: Parallel `switch(type)` or `if (type == A)` cascades scattered across multiple classes inducing Shotgun Surgery.
   - Remedy: Replace scattered type conditionals with Polymorphism, Strategy Pattern, or dedicated Visitor dispatch.
   - Micro-Snippet:
     ```text
     Smell:  PaymentService, InvoiceService, and EmailService all maintain parallel switch (user.getPlanTier()).
     Remedy: Encapsulate tier-specific behaviors inside PlanTier strategy implementations.
     ```

6. **Robustness & Resource Hygiene**
   - Smell: Swallowing errors via empty catch blocks; unmanaged streams, database connection handles, or event listeners lacking cleanup hooks.
   - Remedy: Explicitly handle or rethrow with contextual metadata; encapsulate resource lifecycles within deterministic disposal blocks (`defer`, `finally`, `using`).
   - Micro-Snippet:
     ```text
     Smell:  try { connect(); } catch (Exception e) {}
     Remedy: try { connect(); } catch (Exception e) { logger.error("DB connection failed", e); throw new ServiceUnavailableException(e); }
     ```

## Canonical Output Contract

```markdown
# Code Inspection Dossier: <Target Scope / Module>

## 1. Executive Summary
- **Target Inspected**: `<file_or_directory_path>`
- **Health Rating**: <CLEAN | MODERATE_DEBT | CRITICAL_DEBT>
- **Primary Bottleneck**: <Single-sentence summary of greatest structural risk>

## 2. Technical Debt Catalog

### [<CRITICAL | MAJOR | MINOR>] <Dimension Name>: `<Target Entity / Symbol>`
- **Location**: [<file>#L<N>-L<M>](file:///path/to/file#L<N>-L<M>)
- **Smell**: <Observable anti-pattern description and maintainability impact>
- **Remedy**: <Exact architectural decomposition or extraction path>

*(Repeat for each distinct structural finding; emit 'No structural code smells observed across rubric dimensions' if clean)*

## 3. Pipeline Routing
- `/changeset` ── Multi-file structural refactoring, God class decomposition, or shared abstraction extraction.
- `/diagnose` ── Suspected runtime functional bugs or test failures.
```

## Execution Protocol

### Phase 1: Scope Triage & Macro Grounding

1. Ingest target path (`file`, `directory`, or `subsystem`).
2. Identify primary structural anchors (classes, module entrypoints, shared utilities) using symbol and file tools.

### Phase 2: Scale-Adaptive Inspection

1. Evaluate target scope scale:
   - *In-Turn Execution* (< 10 files or single cohesive module): Audit code directly in-turn against rubric dimensions.
   - *Subagent Fan-Out* (multi-package subsystems or broad repositories): Launch 2–3 `research` subagents (Role: `Structural Inspector (<Slice>)`).
2. Context Seeding Contract for Subagents:
   - Seed each subagent with target slice coordinates, discovered structural anchors, Domain Rubric dimensions, and the Canonical Output Contract.

### Phase 3: Synthesis & Turn-Halt Gate (Fan-in)

1. Reconcile and deduplicate subagent findings into the Canonical Output Contract.
2. Assign definitive Health Rating:
   - `CLEAN`: Zero structural defects breaching rubric.
   - `MODERATE_DEBT`: Isolated encapsulation leaks, scattered conditionals, or minor duplication.
   - `CRITICAL_DEBT`: Pervasive God classes, cross-subsystem copy-paste, or systemic resource leaks.
3. Emit the completed Inspection Dossier into conversation stream and halt turn immediately.
