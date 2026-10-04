# stick-history

[![build status](https://img.shields.io/github/actions/workflow/status/tanem/stick-history/ci.yml?branch=main&style=flat-square)](https://github.com/tanem/stick-history/actions/workflows/ci.yml)
[![go reference](https://img.shields.io/badge/go-reference-007d9c?style=flat-square)](https://pkg.go.dev/github.com/tanem/stick-history/pdb)

> 🎧 The stick remembers what you played. This prints it.

```console
$ stick-history
1. Warm Up Slot - Nobody Here Yet
2. Red Lights - Trim It Back
3. Sync Button - Never Touched It
4. Booth Monitor - Louder Than The Room
5. Front Left - What Is This One
6. Headliner - Running Late (Extended Mix)
7. Lights On - One More (Promoter Says No Edit)
```

The morning after a set, someone asks for a track ID, and the answer is on a USB stick.

`stick-history` prints the tracklist of a set from the History a Pioneer DJ player wrote to a USB stick. It reads `PIONEER/rekordbox/export.pdb` on the stick directly, so it does not need rekordbox to turn a History into a tracklist.

## Install

- With Homebrew, on macOS or Linux: `brew install tanem/tap/stick-history`
- With Go 1.23 or later: `go install github.com/tanem/stick-history@latest`
- From a release archive for macOS, Windows or Linux: [GitHub Releases](https://github.com/tanem/stick-history/releases)

The macOS binaries are not signed or notarised, so Gatekeeper can block one that came from an archive downloaded with a browser.

## Usage

With one stick mounted, run it with no arguments:

```sh
stick-history
```

This finds the stick and prints the newest non-empty History to standard output, one numbered line per track in play order.

Options:

- `--list` prints each non-empty History with its name and track count, newest last.
- `--history <n>` prints `HISTORY <n>` instead of the newest one.
- `--volume <path>` chooses the stick to read.
- `--version` prints the version and exits.

With more than one stick mounted and no `--volume`, the command exits non-zero and lists the sticks on standard error.

A trailing path writes the tracklist to that file instead of standard output:

```sh
stick-history tracklist.txt
```

## Caveats

- Verified on an XDJ-700 with a stick exported by rekordbox 6, and not checked with a stick exported by rekordbox 7.
- One stick per set: Histories from two sticks are not merged.
- A History can hold fewer tracks than were played, or none, as [docs/export-pdb.md](docs/export-pdb.md) describes.

## Docs

- [docs/cli.md](docs/cli.md): the full behaviour of the command, with its output rules and its errors.
- [docs/install.md](docs/install.md): the release archives, how to verify one, Gatekeeper and what `--version` prints.
- [docs/export-pdb.md](docs/export-pdb.md): what the tool reads from the stick and what it does not.
- [docs/releasing.md](docs/releasing.md): how a release is made.

[CONTRIBUTING.md](CONTRIBUTING.md) covers sending a change, and [SECURITY.md](SECURITY.md) covers reporting a vulnerability.
