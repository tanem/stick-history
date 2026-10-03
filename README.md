# stick-history

[![build status](https://img.shields.io/github/actions/workflow/status/tanem/stick-history/ci.yml?branch=main&style=flat-square)](https://github.com/tanem/stick-history/actions/workflows/ci.yml)
[![go reference](https://img.shields.io/badge/go-reference-007d9c?style=flat-square)](https://pkg.go.dev/github.com/tanem/stick-history/pdb)

🎧 The stick remembers what you played. This prints it.

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

`stick-history` prints the tracklist of a set from the History a Pioneer DJ player wrote to a USB stick. It reads `PIONEER/rekordbox/export.pdb` on the stick directly, so rekordbox is not needed to turn a History into a tracklist.

## Usage

With one stick mounted, the common case needs no arguments:

```sh
stick-history
```

This finds the stick, picks the newest non-empty History and prints it to standard output, one line per track, numbered from 1 in play order:

```
1. Artist - Title
2. Artist - Title
```

Lines end with LF. A track whose artist is missing from the stick's artist table prints with an empty artist and the separator kept, so the numbering stays aligned with the History.

Control characters in a title, an artist or a History name are printed as `�` (U+FFFD), so a track always takes one line and a stick cannot send escape sequences to the terminal. The Unicode line and paragraph separators and bytes that are not valid UTF-8 are printed the same way. So is a control character in the stick's path when an error message names it, since a volume label can hold one.

Options:

- `--list` prints each non-empty History with its name and track count, tab-separated, newest last.
- `--history <n>` prints `HISTORY <n>` instead of the newest one. `<n>` can be given with or without leading zeros: `--history 52` and `--history 052` are the same.
- `--volume <path>` chooses the stick when more than one is mounted. Without it, the command exits non-zero and lists the candidates on standard error.
- A trailing path writes the tracklist to that file instead of standard output. Options go before the path.
- `--version` prints the version and exits.

A stick is a mounted volume with a `PIONEER` folder at its root. The command looks under `/Volumes` on macOS, at the drive roots on Windows, and under `/media`, `/run/media` and `/mnt` on Linux.

Players name Histories `HISTORY 001` upward and record no date, so the newest History is the one with the highest number.

Errors go to standard error with a non-zero exit: no stick found, no non-empty History, a named History that is empty or absent, or an `export.pdb` that is unreadable or malformed. Nothing is printed to standard output in that case.

### Titles with a key tag

The tool prints titles as the stick holds them. A collection tagged by Mixed In Key carries the key at the front of each title, as in `Fm - Title`. This pipe strips it:

```sh
stick-history | sed -E 's/ - [A-G][#b]?m? - / - /'
```

## Install

With Homebrew, on macOS or Linux:

```sh
brew install tanem/tap/stick-history
```

The formula is in [tanem/homebrew-tap](https://github.com/tanem/homebrew-tap). Each release updates it, so `brew upgrade` installs the latest release.

With Go 1.23 or later:

```sh
go install github.com/tanem/stick-history@latest
```

Archives for macOS (Apple silicon and Intel), Windows (x86-64) and Linux (x86-64) are attached to each [GitHub Release](https://github.com/tanem/stick-history/releases), with their checksums in `SHA256SUMS`. The macOS binaries are not signed or notarised. Homebrew does not quarantine what it downloads, so the binary it installs runs without a Gatekeeper prompt. A binary from an archive downloaded with a browser is quarantined, and Gatekeeper can block it.

Each archive has a build provenance attestation. With the [GitHub CLI](https://cli.github.com), this checks that an archive was built by this repo's release workflow:

```sh
gh attestation verify stick-history_0.1.0_darwin_arm64.tar.gz --repo tanem/stick-history
```

`stick-history --version` prints the version of a release, as in `stick-history 0.1.0`. A build made with `go install` prints the version of the module, and any other build prints `stick-history (devel)`.

## What it reads

`export.pdb` is the database rekordbox writes to a stick when it exports a collection, and the player writes each set's History into it. The tool reads the tracks, artists, history playlists and history entries tables, following the [Deep Symmetry analysis](https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html) of the format. It depends on the Go standard library alone.

The `pdb` package is separate from the command. `pdb.Open(path)` parses a file and `Histories()` returns every History with its tracks in play order, so the reader can be used on its own. It returns strings as the file stores them, control characters included. It returns an error for a file whose strings decode to more than twice the file's size, which happens only when a crafted file points many rows at one string. It also returns an error for a file with more history entries than one for each 12 bytes of the file's size, which happens only when a crafted file points many row offsets at one history entry. A file with more history playlists than 4,096 or its page count, whichever is larger, is an error too. That limit is not a physical one: it assumes a file a player writes stays under it. A crafted file can fill its pages with short history playlist rows and have thousands for each page.

## Caveats

- Verified on an XDJ-700 with a stick exported by rekordbox 6. Sticks exported by rekordbox 7 have not been checked.
- Newer players also write a Device Library Plus database, `exportLibrary.db`. The tool does not read it.
- One stick per set. Histories from two sticks are not merged.
- Artist rows of subtype `0x64` did not occur on the stick the tool was verified against. Their layout follows the Deep Symmetry analysis and is covered by the synthetic test fixture only.

## Development

```sh
go test ./...
```

The tests build a synthetic `export.pdb` in memory, in `internal/pdbtest`. It holds no track, artist or History from a real collection. `*.pdb` is ignored by git, so a real `export.pdb` is not committed by accident.

The parser has a fuzz target. `go test` runs it on its seed, and this fuzzes it:

```sh
go test -fuzz=FuzzParse ./pdb
```

### Releases

The release workflow runs every Monday and can be started by hand. It uses [tanem/release-action](https://github.com/tanem/release-action) to derive the next version from the labels on the pull requests merged since the last release: `breaking` is a major bump, `enhancement` is a minor bump and any other label is a patch bump. When nothing was merged, it releases nothing.

When there is something to release, the workflow tags the commit and creates the GitHub Release with notes GitHub generates from the pull requests. It then builds the archives with `scripts/build.sh`, attests them, attaches them to the release together with `SHA256SUMS`, and commits the formula that `scripts/formula.sh` writes to `tanem/homebrew-tap`. No tag is pushed by hand.

A run started by hand with the dry-run option logs the version it would release and changes nothing.

[docs/adr/0001-keep-build-scripts-over-goreleaser.md](docs/adr/0001-keep-build-scripts-over-goreleaser.md) records why the build uses these scripts and not GoReleaser.

[CONTRIBUTING.md](CONTRIBUTING.md) covers sending a change, and [SECURITY.md](SECURITY.md) covers reporting a vulnerability.
