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

A stick is a mounted volume with a `PIONEER` folder at its root. The command looks under `/Volumes` on macOS, at the drive roots on Windows, and under `/media`, `/run/media` and `/mnt` on Linux.

Players name Histories `HISTORY 001` upward and record no date, so the newest History is the one with the highest number.

Errors go to standard error with a non-zero exit: no stick found, no non-empty History, a named History that is empty or absent, or an `export.pdb` that is unreadable or malformed. Nothing is printed to standard output in that case.

### Titles with a key tag

The tool prints titles as the stick holds them. A collection tagged by Mixed In Key carries the key at the front of each title, as in `Fm - Title`. This pipe strips it:

```sh
stick-history | sed -E 's/ - [A-G][#b]?m? - / - /'
```

## Install

With Go 1.23 or later:

```sh
go install github.com/tanem/stick-history@latest
```

With Homebrew:

```sh
brew install tanem/tap/stick-history
```

Binaries for macOS (Apple silicon and Intel), Windows (x86-64) and Linux (x86-64) are attached to each [GitHub Release](https://github.com/tanem/stick-history/releases).

## What it reads

`export.pdb` is the database rekordbox writes to a stick when it exports a collection, and the player writes each set's History into it. The tool reads the tracks, artists, history playlists and history entries tables, following the [Deep Symmetry analysis](https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html) of the format. It depends on the Go standard library alone.

The `pdb` package is separate from the command. `pdb.Open(path)` parses a file and `Histories()` returns every History with its tracks in play order, so the reader can be used on its own. It returns strings as the file stores them, control characters included. It returns an error for a file whose strings decode to more than twice the file's size, which happens only when a crafted file points many rows at one string. It also returns an error for a file with more history entries than one for each 12 bytes of the file's size, which happens only when a crafted file points many row offsets at one history entry.

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

A tag of the form `v1.2.3` runs the release workflow, which builds the binaries, attaches them to a GitHub Release together with `SHA256SUMS`, and attaches a Homebrew formula for copying into the `tanem/homebrew-tap` repo.

[CONTRIBUTING.md](CONTRIBUTING.md) covers sending a change, and [SECURITY.md](SECURITY.md) covers reporting a vulnerability.
