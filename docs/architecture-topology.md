---
title: ZenForge Daemon (zenfd) Architecture & Seams
description: Host-daemon topology, process group supervision, and communication seams.
status: active
created: 2026-11-03
updated: 2026-11-03
---

# ZenForge Daemon (zenfd) Architecture & Seams

ZenForge Daemon is an agent-native operating environment combining SQLite WAL as Single Source of Truth with Model Context Protocol (MCP) and POSIX process group supervision.

```mermaid
graph LR
  subgraph Host [Developer Host / Agent Runner]
    A[Agent CLI / IDE] -->|JSON-RPC 2.0 stdio| B(zenfd mcp)
    C[Web Cockpit :5173] -->|HTTP / SSE :1123| D[Gateway Server]
  end
  subgraph ControlPlane [Daemon Engine]
    B --> E[(SQLite WAL SSOT)]
    D --> E
    D --> F[DAG Scheduler]
    F --> G[POSIX Supervisor]
  end
```

## Detailed Subsystem Specifications

- **Domain & Interface Contracts**: [docs/zenfd/contracts.md](zenfd/contracts.md)
- **Web Cockpit UI Layout**: [docs/zenfd/ui-layout.md](zenfd/ui-layout.md)
