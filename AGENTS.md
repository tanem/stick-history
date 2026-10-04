## Agent skills

### Issue tracker

Issues are GitHub Issues on `tanem/stick-history`. Use the `gh` CLI to work with them. See `docs/agents/issue-tracker.md`.

### Triage labels

The five triage roles use their default label names: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human` and `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

This repo is single-context. It has one `GLOSSARY.md` and `docs/adr/` at the repo root. See `docs/agents/domain.md`.

## Docs

- `README.md` is for the person running the command. Keep the detail in the pages below.
- `docs/cli.md` has the full behaviour of the command: stick detection, output rules and errors.
- `docs/install.md` has the release archives, how to verify one, Gatekeeper and the `--version` output.
- `docs/export-pdb.md` has the notes on `export.pdb`: what the tool reads, what has been verified and why a History can be empty.
- `docs/releasing.md` has the release runbook.
- The limits the parser puts on a file are in the `pdb` package doc comment, in `pdb/pdb.go`.
- `go test ./...` checks the relative links in every Markdown file, in `internal/doclinks`.

## Pull requests

### Release label

Every pull request carries exactly one release label. The `release-label` check fails without it. `safe to test` is not counted.

The label sets the version bump of the next release:

- `breaking` is major.
- `enhancement` is minor.
- Any other label is patch.

For a patch, use:

- `bug` for a fix to the shipped tool.
- `documentation` for docs.
- `internal` for a change that does not alter the shipped tool.

### Breaking changes while the version is `0.x`

Label a breaking change `enhancement`, not `breaking`. `tanem/release-action` has no special handling for `0.x`, so `breaking` would produce `1.0.0`.
