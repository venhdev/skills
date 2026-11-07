# ZenForge Skills

## Install

```bash
npx skills add venhdev/skills
```

## zenfd

`zenfd` is the single-binary control plane and multi-agent tracking daemon for ZenForge.

### Install

Install via **ZenCLI**:
```bash
zen i zenforge
```
Or download pre-compiled binaries directly from [GitHub Releases](https://github.com/venhdev/zenforge/releases).

### Build from source

```bash
cd apps/daemon && make build
# Binary is produced at apps/daemon/bin/zenfd
```

### CLI Quickstart

```bash
zenfd              # Show quickstart overview and active daemon state
zenfd daemon       # Start background daemon & cockpit UI at http://127.0.0.1:1123
zenfd status       # Comprehensive status report (daemon runtime + all AI MCP clients)
zenfd mcp          # Audit MCP setups & view integration guide (zero hang)
```

Client setup, tool reference, write surfaces, and verification: [MCP.md](MCP.md).

Agent-facing cards:

- [zenfd-tools](skills/zenfd-tools/SKILL.md) — MCP tool surface and task lifecycle.
- [zenf-project](skills/zenf-project/SKILL.md) — project/plan/task hierarchy front door.
- [zenf-docs](skills/zenf-docs/SKILL.md) — SSOT document registry front door.
- [zenf-agent](skills/zenf-agent/SKILL.md) — worker personas and observed sessions.
