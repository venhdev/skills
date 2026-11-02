# Test Harness Anchor — ZenForge Ecosystem

Authoritative testing SSOT routing and hermetic primitives registry.

## 1. Governing Testing SSOTs

Navigate directly to these documents for testing standards and conventions (do not duplicate content here):

- `docs/ZENFORGE_CONSTRUCTION_RULES.md`
- `docs/ROADMAP.md`

## 2. Hermetic Primitives Registry

Foundation helpers providing hermetic test isolation:

| Primitive | Helper Relative Path | Status |
| :--- | :--- | :--- |
| **Deterministic Clock** | `none` | Marked for remediation in `/harness-audit` |
| **Storage Sandbox** | `apps/daemon/internal/storage/db.go` | In-memory WASM SQLite (`:memory:`) & `t.TempDir()` |
| **Fixtures / Factories** | `none` | Marked for remediation in `/harness-audit` |
| **Transport / Network Mock** | `none` | Local loopback gateway (`127.0.0.1:0` in test suites) |
| **Lifecycle / Cleanup** | `apps/daemon/internal/gateway/server.go` | Graceful `Shutdown` & `t.Cleanup` |
