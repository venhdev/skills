# Issue tracker: GitLab

Issues and specs for this repo live as GitLab issues. Use the [`glab`](https://gitlab.com/gitlab-org/cli) CLI for all operations.

## Conventions

- **Create an issue**: `glab issue create --title "..." --description "..."`. Use a heredoc for multi-line descriptions. Pass `--description -` to open an editor.
- **Read an issue**: `glab issue view <number> --comments`. Use `-F json` for machine-readable output.
- **List issues**: `glab issue list -F json` with appropriate `--label` filters.
- **Comment on an issue**: `glab issue note <number> --message "..."`. GitLab calls comments "notes".
- **Apply / remove labels**: `glab issue update <number> --label "..."` / `--unlabel "..."`. Multiple labels can be comma-separated or by repeating the flag.
- **Close**: `glab issue close <number>`. `glab issue close` does not accept a closing comment, so post the explanation first with `glab issue note <number> --message "..."`, then close.
- **Merge requests**: GitLab calls PRs "merge requests". Use `glab mr create`, `glab mr view`, `glab mr note`, etc., the same shape as `gh pr ...` with `mr` in place of `pr` and `note`/`--message` in place of `comment`/`--body`.
- **Number space**: Unlike GitHub, GitLab numbers issues and MRs separately, so `#42` refers specifically to issue `#42` (MRs are referenced as `!42`).

Infer the repo from `git remote -v`; `glab` does this automatically when run inside a clone.

## When a skill says "publish to the issue tracker"

Create a GitLab issue via `glab issue create`.

## When a skill says "fetch the relevant ticket"

Run `glab issue view <number> --comments`.

## Task Decomposition & Execution Operations

Used by `/to-tasks` and `/forge`:

- **Publish Task**: `glab issue create --title "<NN>: <Task Title>" --description "..."` with embedded changeset and acceptance criteria.
- **Blocking Dependencies**: GitLab's **native blocking link**, the canonical, UI-visible representation. Add it with the `/blocked_by #<n>` quick action, posted as a note (`glab issue note <child> --message "/blocked_by #<blocker>"`). Native blocking links are a Premium/Ultimate feature; on the free tier (or where unavailable) fall back to a `**Blocked by**: #<n>` line at the top of the description. A ticket is unblocked when every blocker is closed.
- **Frontier Query**: `glab issue list -F json`, drop any with an open blocker: a native `blocked_by` link to an open issue (`glab api projects/:id/issues/:iid/links`), or an open issue in the `Blocked by` line, or an assignee; first in sequence order wins.
- **Claim**: `glab issue update <number> --assignee @me`.
- **Resolve**: Ensure all acceptance criteria are verified `[x]`, post resolution comment, and close: `glab issue note <number> --message "<resolution summary>"` then `glab issue close <number>`.
