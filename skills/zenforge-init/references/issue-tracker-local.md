# Issue tracker: Local Markdown

Issues and tasks for this repository live as markdown files in `.agents/tasks/`.

## Conventions

- One feature per directory: `.agents/tasks/<feature-slug>/`
- The spec is `docs/specs/<feature-slug>.md`
- Implementation tasks are one file per task at `.agents/tasks/<feature-slug>/<NN>-<slug>.md`, numbered sequentially from `01`, never a single combined file
- Task state is recorded as a `**Status**:` line (`ready`, `in-progress`, `done`) near the top of each file
- Comments and execution notes append to the bottom of the file under a `## Comments` heading

## When a skill says "publish to the issue tracker"

Create a new file under `.agents/tasks/<feature-slug>/` (creating the directory if needed).

## When a skill says "fetch the relevant ticket"

Read the file at the referenced path. The user will normally pass the path or the task number directly.

## Wayfinding operations

Used for autonomous frontier exploration. The **map** is a file with one **child** file per ticket:

- **Map**: `.agents/tasks/<effort>/map.md` (the Notes / Decisions-so-far / Fog body).
- **Child task**: `.agents/tasks/<effort>/<NN>-<slug>.md`, numbered from `01`, with the deliverable in the body. A `Type:` line records the task type (`research`/`prototype`/`task`); a `Status:` line records `ready`/`in-progress`/`done`.
- **Blocking**: a `Blocked by: <NN>, <NN>` line near the top. A task is unblocked when every task it lists has `Status: done`.
- **Frontier**: scan `.agents/tasks/<effort>/` for files that are `ready`, unblocked, and unclaimed; first by number wins.
- **Claim**: set `Status: in-progress` and save before active execution.
- **Resolve**: set `Status: done`, ensure acceptance criteria are checked `[x]`, then append a context pointer to the map in `map.md`.
