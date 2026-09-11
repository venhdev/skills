# Issue tracker: Local Markdown

Issues and tasks for this repository live as markdown files in `.agents/tasks/`.

## Conventions

- One feature per directory: `.agents/tasks/<feature-slug>/`
- Associated specifications typically live in `docs/` (e.g., `docs/<feature-slug>.md` or governed via `/ssot`).
- Implementation tasks are one file per task at `.agents/tasks/<feature-slug>/<NN>-<slug>.md`, numbered sequentially from `01`, never a single combined file.
- Task state is recorded as a `**Status**:` line (`ready`, `in-progress`, `done`) near the top of each file.
- Task dependencies are recorded as a `**Blocked by**:` line near the top of each file.
- Comments and execution notes append to the bottom of the file under a `## Comments` heading.

## When a skill says "publish to the issue tracker"

Create a new file under `.agents/tasks/<feature-slug>/` (creating the directory if needed).

## When a skill says "fetch the relevant ticket"

Read the file at the referenced path. The user will normally pass the path or the task number directly.

## Task Decomposition & Execution Operations

Used by `/to-tasks` and `/forge`:

- **Task Files**: `.agents/tasks/<feature-slug>/<NN>-<slug>.md`, numbered sequentially starting from `01`.
- **Task Header Format**:

  ```markdown
  # <NN>: <Task Title>

  **Status**: ready
  **Blocked by**: None | <NN> (<Title>)
  ```

- **Blocking**: A `**Blocked by**:` line near the top. A task is unblocked when `Blocked by: None` or every task listed in `Blocked by` has `**Status**: done`.
- **Frontier**: Scan `.agents/tasks/<feature-slug>/` for files where `Status` is `ready` and all predecessors in `Blocked by` are `done`. First in sequence order is the active frontier.
- **Claim**: Set `**Status**: in-progress` before active execution.
- **Resolve**: Ensure all acceptance criteria are verified `[x]`, set `**Status**: done`, and append execution notes under `## Comments`.
