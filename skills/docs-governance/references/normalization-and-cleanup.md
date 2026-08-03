# Normalization and Cleanup

Read this reference when conventions differ or work includes normalization,
merge, move, or deletion.

## Classify Differences

| Classification | Action |
| --- | --- |
| `aligned` | Preserve. |
| `compatible extension` | Preserve. |
| `missing` | Add only when required; otherwise recommend it when materially useful. |
| `conflict` | Explain trade-offs and migration impact. |
| `legacy` | Propose normalization or cleanup. |
| `unknown` | Preserve and request a decision. |

## Normalize

1. Inventory affected files, templates, generators, indexes, linters, and
   consumers.
2. Define exact old-to-new field and value mappings.
3. Preserve compatible extensions and unknown values.
4. Separate current-task changes from optional corpus migration.
5. Propose exact scope and validation; never mass-update untouched documents
   without explicit approval.

## Merge and Cleanup

```text
preserve durable information
-> merge into the approved long-term owner
-> propose and obtain approval for an owner if none exists
-> validate the owner
-> propose deleting sources with no independent value
```

Keep completed documents only when useful. Propose rejected, obsolete, replaced,
or temporary documents for deletion only when their durable information has an
approved owner and they retain no independent value. Preserve decision history
by default; retain durable decisions and rationale before proposing deletion of
a historical document.

## Report Before Mutation

Report:

- every mapping and compatible extension;
- each source file and destination owner;
- retained durable information;
- exact edits, moves, and deletion candidates;
- independent value or history that would be lost;
- affected tooling, validation, risks, and recovery method.

Wait for explicit confirmation of the exact mutation set.

## Apply and Validate

- Complete and validate approved merges before deleting sources.
- Delete only exact files confirmed by the user; do not use wildcards,
  unresolved variables, or broad targets.
- Preserve read-only and generated artifacts unless their owning workflow is
  explicitly in scope and approved.
- Stop when new conflicts or out-of-scope files appear.
- Run repository-prescribed checks; verify metadata, links, indexes, generators,
  consumers, stale references, duplication, and authority conflicts.
- Confirm that deleted documents' durable information remains available and
  report recovery options.
