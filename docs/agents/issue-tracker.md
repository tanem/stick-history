# Issue tracker: GitHub

Issues and specs for this repo are GitHub issues. Use the `gh` CLI for every operation.

## Conventions

- **Create an issue**: `gh issue create --title "..." --body "..."`. Use a heredoc for a multi-line body.
- **Read an issue**: `gh issue view <number> --comments`. Filter the comments with `jq` and fetch the labels too.
- **List issues**: `gh issue list --state open --json number,title,body,labels,comments --jq '[.[] | {number, title, body, labels: [.labels[].name], comments: [.comments[].body]}]'`, with the `--label` and `--state` filters the task needs.
- **Comment on an issue**: `gh issue comment <number> --body "..."`
- **Apply / remove labels**: `gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **Close**: `gh issue close <number> --comment "..."`

Infer the repo from `git remote -v`. `gh` does this itself when run inside a clone.

## Pull requests as a triage surface

**PRs as a request surface: no.** _(Set to `yes` if this repo treats external PRs as feature requests. `/triage` reads this flag.)_

When the flag is `yes`, PRs use the same labels and states as issues, through the `gh pr` commands:

- **Read a PR**: `gh pr view <number> --comments`, and `gh pr diff <number>` for the diff.
- **List external PRs for triage**: `gh pr list --state open --json number,title,body,labels,author,authorAssociation,comments`. Keep only an `authorAssociation` of `CONTRIBUTOR`, `FIRST_TIME_CONTRIBUTOR` or `NONE`. Drop `OWNER`, `MEMBER` and `COLLABORATOR`.
- **Comment / label / close**: `gh pr comment`, `gh pr edit --add-label`/`--remove-label`, `gh pr close`.

GitHub numbers issues and PRs in one sequence, so a bare `#42` can be either. Try `gh pr view 42` first, then `gh issue view 42`.

## When a skill says "publish to the issue tracker"

Create a GitHub issue.

## When a skill says "fetch the relevant ticket"

Run `gh issue view <number> --comments`.

## Wayfinding operations

`/wayfinder` uses these. The **map** is one issue, and its **child** issues are the tickets.

- **Map**: one issue labelled `wayfinder:map`. Its body holds Notes, Decisions-so-far and Fog. Create it with `gh issue create --label wayfinder:map`.
- **Child ticket**: an issue linked to the map as a GitHub sub-issue, using `gh api` on the sub-issues endpoint.
  - Where sub-issues are not enabled, add the child to a task list in the map body and put `Part of #<map>` at the top of the child body.
  - Its label is `wayfinder:<type>`, where the type is `research`, `prototype`, `grilling` or `task`.
  - Once claimed, the ticket is assigned to the driving dev.
- **Blocking**: GitHub's **native issue dependencies**. They are the canonical representation and show in the UI.
  - Add an edge with `gh api --method POST repos/<owner>/<repo>/issues/<child>/dependencies/blocked_by -F issue_id=<blocker-db-id>`.
  - `<blocker-db-id>` is the blocker's numeric **database id**, from `gh api repos/<owner>/<repo>/issues/<n> --jq .id`. It is _not_ the `#number` or the `node_id`.
  - GitHub reports `issue_dependencies_summary.blocked_by`. It counts open blockers only, so it is the live gate.
  - Where dependencies are not available, put a `Blocked by: #<n>, #<n>` line at the top of the child body.
  - A ticket is unblocked when every blocker is closed.
- **Frontier query**: list the map's open children with `gh issue list --state open`, scoped to the map's sub-issues or task list. Drop any child with an assignee or an open blocker. An open blocker is `issue_dependencies_summary.blocked_by > 0`, or an open issue in the `Blocked by` line. The first child left, in map order, wins.
- **Claim**: `gh issue edit <n> --add-assignee @me`. This is the session's first write.
- **Resolve**: run `gh issue comment <n> --body "<answer>"`, then `gh issue close <n>`. Then append a context pointer (a gist and a link) to the map's Decisions-so-far.
