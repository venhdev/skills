# Output: Doc Status Report

Use this shape for Read mode unless the user asks for another format.

```markdown
## Doc status — <title>

| Field | Value |
|---|---|
| Path | `<path>` |
| Type / kind | `<type>` / `<kind>` |
| Status | `<status or current>` |
| Freshness | `<current | stale | due soon>` |
| Depends on | `<current dependencies or none>` |
| Updates | `<downstream docs or none>` |
| Replacement | `<replacement path or none>` |

### Read decision

`READ` / `SKIP` / `READ WITH WARNING`

Reason: <one sentence>

### Current guidance

<Only summarize current guidance. If the target is superseded, summarize the replacement instead.>

### Downstream impact

- <affected docs, if relevant>
```
