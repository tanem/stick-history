## Agent skills

### Issue tracker

Issues live in GitHub Issues on `tanem/stick-history`, accessed with the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

The five canonical triage roles use their default label names: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

## Pull requests

### Release label

Every pull request carries exactly one release label, and the `release-label` check fails without it. The label sets the version bump of the next release: `breaking` is major, `enhancement` is minor, and any other label is patch. Use `bug` for a fix to the shipped tool, `documentation` for docs and `internal` for a change that does not alter the shipped tool. `safe to test` is not counted.

### Breaking changes while the version is `0.x`

Label a breaking change `enhancement`, not `breaking`. `tanem/release-action` has no special handling for `0.x`, so `breaking` would produce `1.0.0`.
