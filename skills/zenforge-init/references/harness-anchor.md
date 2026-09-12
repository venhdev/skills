# Test Harness Anchor — <Project Name>

Authoritative testing SSOT routing and hermetic primitives registry.

## 1. Governing Testing SSOTs

Navigate directly to these documents for testing standards and conventions (do not duplicate content here):

- `<relative-path, e.g., docs/testing.md | test/AGENTS.md | internal>`

## 2. Hermetic Primitives Registry

Foundation helpers providing hermetic test isolation (specify relative path or `none` if not applicable):

| Primitive | Helper Relative Path |
| :--- | :--- |
| **Deterministic Clock** | `<relative-path, e.g., test/helpers/clock.* | none>` |
| **Storage Sandbox** | `<relative-path, e.g., test/helpers/storage.* | :memory: | none>` |
| **Fixtures / Factories** | `<relative-path, e.g., test/fixtures/ | test/factories/ | none>` |
| **Transport / Network Mock** | `<relative-path, e.g., test/helpers/mock_transport.* | none>` |
| **Lifecycle / Cleanup** | `<relative-path, e.g., test/helpers/cleanup.* | none>` |
