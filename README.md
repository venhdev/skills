# ZenForge Skills

## Install

```bash
npx skills add venhdev/skills
```

## zenfd MCP

Build: `cd apps/daemon && make build`

Setup, client configuration, the 7-tool reference with parameters, and live
verification live in [MCP.md](MCP.md).

The MCP server manages plans and tasks only and is not bound to a repository.
Review is human-in-the-loop through the main agent: a worker calls
`zenfd_request_review` and exits, the main agent surfaces the question to the
human, then calls `zenfd_resolve_review`. The Web Cockpit only shows
`AWAITING_REVIEW` details read-only.
