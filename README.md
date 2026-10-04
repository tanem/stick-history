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

## Usage

With one stick mounted, run it with no arguments:

```sh
stick-history
```

This finds the stick and prints the newest non-empty History to standard output. Each track takes one line, numbered from 1 in play order:

```
1. Artist - Title
2. Artist - Title
```

Lines end with LF.

If a track's artist is missing from the stick's artist table, the line has an empty artist and keeps the separator. The numbering then still matches the History.

Some characters are printed as `�` (U+FFFD). This keeps each track on one line and stops a stick from sending escape sequences to the terminal. It applies to:

- Control characters in a title, an artist or a History name.
- The Unicode line and paragraph separators.
- Bytes that are not valid UTF-8.
- Control characters in the stick's path when an error message names it. A volume label can hold one.

Options:

- `--list` prints each non-empty History with its name and track count, tab-separated, newest last. If any History is empty, a line on standard error says how many of the Histories were left out.
- `--history <n>` prints `HISTORY <n>` instead of the newest one. `<n>` can be given with or without leading zeros: `--history 52` and `--history 052` are the same.
- `--volume <path>` chooses the stick when more than one is mounted. Without it, the command exits non-zero and lists the candidates on standard error.
- A trailing path writes the tracklist to that file instead of standard output. Options go before the path.
- `--version` prints the version and exits.

A stick is a mounted volume with a `PIONEER` folder at its root. The command looks under `/Volumes` on macOS, at the drive roots on Windows, and under `/media`, `/run/media` and `/mnt` on Linux.

Players name Histories `HISTORY 001` upward and record no date. The newest History is therefore the one with the highest number.

On an error, the command prints a message to standard error, prints nothing to standard output and exits non-zero. These are errors:

- No stick is found.
- No History is non-empty.
- The named History is empty or absent.
- `export.pdb` is unreadable or malformed.

### Cleaning up titles

The tool prints titles as the stick holds them. To clean up a title or strip characters from it, pipe the output through a tool such as `sed`.

This example strips a key at the front of a title in the form `Fm - Title`:

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

Each [GitHub Release](https://github.com/tanem/stick-history/releases) has archives for:

- macOS, Apple silicon and Intel
- Windows, x86-64
- Linux, x86-64

Their checksums are in `SHA256SUMS` on the same release.

The macOS binaries are not signed or notarised:

- Homebrew does not quarantine a formula's download, so the binary it installs runs without a Gatekeeper prompt.
- A binary from an archive downloaded with a browser is quarantined, and Gatekeeper can block it.

Each archive has a build provenance attestation. With the [GitHub CLI](https://cli.github.com), this checks that an archive was built by this repo's release workflow:

```sh
gh attestation verify stick-history_0.1.0_darwin_arm64.tar.gz --repo tanem/stick-history
```

`stick-history --version` prints:

- The version of the release for a release build, as in `stick-history 0.1.0`.
- The version of the module for a build installed with `go install github.com/tanem/stick-history@latest`.
- `stick-history (devel)` for any other build.

## What it reads

`export.pdb` is the database rekordbox writes to a stick when it exports a collection. The player writes each set's History into it.

The tool reads four tables: tracks, artists, history playlists and history entries. It follows the [Deep Symmetry analysis](https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html) of the format and depends on the Go standard library alone.

The `pdb` package can be used without the command. `pdb.Open(path)` parses a file, and `Histories()` returns every History with its tracks in play order. Strings are returned as the file stores them, control characters included.

`Open` returns an error for a file that exceeds any of these limits:

- **Strings**: the decoded strings total more than twice the file's size. Only a crafted file gets there, by pointing many rows at one string.
- **History entries**: more than one for each 12 bytes of the file's size. Only a crafted file gets there, by pointing many row offsets at one history entry.
- **History playlists**: more than 4,096 or the file's page count, whichever is larger. This limit is assumed, not physical: a file a player writes is expected to stay under it, but a crafted file can fill its pages with short history playlist rows and hold thousands for each page.

## Caveats

- Verified on an XDJ-700 with a stick exported by rekordbox 6. Sticks exported by rekordbox 7 have not been checked.
- Newer players also write a Device Library Plus database, `exportLibrary.db`. The tool does not read it.
- One stick per set. Histories from two sticks are not merged.
- A History can hold fewer tracks than were played, or none. On one stick, rekordbox had synced the collection after the sets. The tracks it removed from the stick were gone from the stick's track table, and so were the History entries that pointed at them, so 122 of 124 Histories were empty and the other two held 4 and 5 tracks. The tool prints what the stick holds. This was seen on that one stick and has not been checked against rekordbox directly.
- Artist rows of subtype `0x64` did not occur on the stick the tool was verified against. Their layout follows the Deep Symmetry analysis and is covered by the synthetic test fixture only.

## Development

Run the tests:

```sh
go test ./...
```

The tests build a synthetic `export.pdb` in memory, in `internal/pdbtest`. It holds no track, artist or History from a real collection. `*.pdb` is ignored by git, so a real `export.pdb` is not committed by accident.

The parser has a fuzz target. `go test` runs it on its seed, and this fuzzes it:

```sh
go test -fuzz=FuzzParse ./pdb
```

### Releases

The release workflow runs every Monday and can be started by hand. No tag is pushed by hand.

It uses [tanem/release-action](https://github.com/tanem/release-action) to derive the next version from the labels on the pull requests merged since the last release:

- `breaking` is a major bump.
- `enhancement` is a minor bump.
- Any other label is a patch bump.

When nothing was merged, it releases nothing. When there is something to release, the workflow:

1. Tags the commit and creates the GitHub Release, with notes GitHub generates from the pull requests.
2. Builds the archives with `scripts/build.sh` and attests them.
3. Attaches the archives and `SHA256SUMS` to the release.
4. Writes the formula with `scripts/formula.sh` and commits it to `tanem/homebrew-tap`.

Before it tags anything, the workflow checks that the tap's deploy key can push. It clones `tanem/homebrew-tap` with the key and runs `git push --dry-run` against it. GitHub refuses a read-only deploy key at that point. A key that is missing, cannot reach the tap or cannot write to it therefore fails the run before a tag or a release exists.

A run started by hand with the dry-run option logs the version it would release and changes nothing. It skips the deploy key check and does not need the key.

A run that fails after the tag leaves a release without some of its archives, its attestations or its formula. Running the workflow again does not finish it. To finish it, start the workflow by hand with the `version` input set to that version, without a leading `v`, as in `0.1.0`.

That run creates no tag and no release. It fails before building if the tag `v<version>` or its GitHub Release does not exist. Otherwise it:

1. Checks out the tag.
2. Builds and attests the archives.
3. Uploads them in place of any files already on the release.
4. Commits the formula to the tap when it differs from the one there. When the tap already holds the formula of a later version, the formula is left as it is.

Two builds of a version are not byte-identical. Resuming a release that was already complete therefore replaces its archives and commits a formula with the new checksums. The attestations of the replaced archives stay in the attestation store.

`version` and the dry-run option cannot be set together. A run with both fails.

[docs/adr/0001-keep-build-scripts-over-goreleaser.md](docs/adr/0001-keep-build-scripts-over-goreleaser.md) records why the build uses these scripts and not GoReleaser.

[CONTRIBUTING.md](CONTRIBUTING.md) covers sending a change, and [SECURITY.md](SECURITY.md) covers reporting a vulnerability.
