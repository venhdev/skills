# Test Smells & Flakiness Anti-Patterns

Authoritative catalog of testing anti-patterns that induce flakiness, false confidence, and maintenance debt.

## 1. Clock Leak & Sleep Trap
- **Smell**: Calling runtime system clocks (`DateTime.now()`, `time.Now()`, `Date.now()`) or hardcoded pauses (`sleep()`, `delay()`, `setTimeout()`).
- **Failure Mode**: Tests fail unpredictably across CI runners due to CPU scheduling jitter, timezone shifts, and leap-second drifts.
- **Remedy**:
  - *Instead of* calling system time directly, inject a controlled virtual clock interface.
  - *Instead of* sleeping for async operations to complete, poll deterministic completion predicates or advance virtual clock ticks manually.

## 2. Mock Mania (Fake Fidelity)
- **Smell**: Mocking internal domain logic, data models, repositories, or SQLite/SQL engines instead of running real components.
- **Failure Mode**: Tests pass green while production crashes due to SQL syntax errors, foreign key constraints, or null-safety violations that mocks silently permitted.
- **Remedy**:
  - *Instead of* mocking internal persistence layers, execute tests against real ephemeral in-memory engines (SQLite `:memory:`, temporary directory databases, or local container instances).
  - Confine mocks strictly to third-party network boundaries (payment gateways, external SMS APIs).

## 3. Assertion Roulette
- **Smell**: Multiple raw assertions (`expect()`, `assert()`) placed sequentially in a single test without diagnostic messages or contextual explanations.
- **Failure Mode**: When test fails on CI, stderr reports only unannotated boolean failures without indicating which invariant collapsed or why.
- **Remedy**:
  - *Instead of* chained raw assertions, provide descriptive diagnostic failure strings on every assertion.
  - *Instead of* verifying multiple unrelated behaviors in one test, split into focused single-behavior test cases.

## 4. Branch Inversion (Happy-Path Bias)
- **Smell**: Authoring tests that exclusively exercise success paths while ignoring error branches, boundary edges, and failure returns.
- **Failure Mode**: High line coverage with low defect prevention; system panics or enters corrupt state upon encountering malformed inputs.
- **Remedy**:
  - *Instead of* testing only valid inputs, pair happy path tests with negative test cases verifying typed error returns and state preservation.

## 5. Duplicate Scaffolding Bloat
- **Smell**: Copy-pasting repetitive entity instantiation or environment setup across multiple test files.
- **Failure Mode**: Modifying a required constructor parameter breaks unrelated test files, creating massive refactoring drag.
- **Remedy**:
  - *Instead of* inline payload construction, centralize entity setup into an Object Mother / Fixture Factory with immutable valid defaults and partial override hooks.
