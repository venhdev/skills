# unified-docs

A single skill for documentation lifecycle work: create, read, maintain, and audit docs through one frontmatter contract.

## What it does

- Creates and updates Diátaxis-style docs.
- Reads docs with lifecycle awareness before trusting the body.
- Manages ADR and plan transitions.
- Normalizes frontmatter.
- Audits stale, orphan, duplicate, broken dependency, superseded dependency, and cascade issues.
- Repairs `depends-on` / `updates` metadata when requested.

## Modes

- **Read mode**: check whether a doc is current, superseded, stale, or dependent on other docs.
- **Maintain mode**: create or update docs while keeping metadata and cascade links valid.
- **Audit mode**: report corpus health and propose or apply repairs.

## Core rule

Frontmatter is the contract. `depends-on` is current prerequisite context; `updates` is downstream cascade impact; ADR lineage uses `supersedes` and `supersededBy`.
