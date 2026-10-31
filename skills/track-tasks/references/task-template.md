# Task Body Contract

Record outcome, acceptance criteria, direction constraints, and proof/reference
provenance for later implementation. Keep executable implementation details out
of the task unless they define a binding interface or schema.

````markdown
# <NN>: <Task Title>

**Status**: ready
**Blocked by**: None | <Task IDs>

## What to Build
<Observable outcome and retained design constraints.>

## Acceptance Criteria
- [ ] <Verifiable criterion>

## Basis
<Solution proof or official references; embed concise proof inline for remote trackers.>

## Embedded Changeset
<Planned impact; reconcile against current checkout before implementation.>

```text
|-- <subsystem>/
|   `-- <file>
|       `-- [UPDATE] <Required change and invariant>.
`-- <deep/path/to/file>
    `-- [CREATE] <Required component or check>.

Summary: <N> files affected (<C> created, <U> updated, <M> moved, <D> deleted).
```

## Comments
<Append publication/execution notes when present; omit empty notes.>
````

For remote trackers, adapt lifecycle to the configured tracker operations:
open/unclaimed corresponds to `ready`, claimed active work to `in-progress`,
and verified completion to `done`. Closing without verified acceptance does
not prove completion. Use one effective lifecycle representation per tracker;
keep any redundant body status synchronized rather than treating it as a
competing authority.
