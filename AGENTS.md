docs/ZENFORGE_CONSTRUCTION_RULES.md

## Environment Specification
- Infrastructure & Toolchain Manifest: `.agents/environment.md`
- Diagnostic & Health Audit: via `/env-doctor`

## Agent Workflow
- **Environment**: Governed by `.agents/environment.md` (local runners, zero-touch for `.env*` secrets).
- **Task Tracker**: Governed by `.agents/task-tracker.md`.
- **Harness Anchor**: Governed by `.agents/harness-anchor.md` (via `/harness-audit`).
- **Documentation**: Governed by `docs/README.md` (via `/ssot`).
