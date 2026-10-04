# The command

`stick-history` prints the tracklist of a set from the History a Pioneer DJ player wrote to a stick.

```
stick-history [--list] [--history <n>] [--volume <path>] [<file>]
stick-history --version
```

## Finding the stick

A stick is a mounted volume with a `PIONEER` folder at its root. The command looks for one in these places:

- macOS: under `/Volumes`.
- Windows: at the drive roots, `A:\` to `Z:\`.
- Linux: under `/media`, `/run/media` and `/mnt`.

With one stick mounted, the command reads it. With more than one mounted, it exits non-zero and lists the candidates on standard error. `--volume <path>` chooses the stick in that case. The path is the root of the stick, the directory that holds the `PIONEER` folder.

The command reads `PIONEER/rekordbox/export.pdb` on the stick. [export-pdb.md](export-pdb.md) describes that file.

## Choosing the History

With no options, the command prints the newest non-empty History.

Players name Histories `HISTORY 001` upward and record no date. The newest History is therefore the one with the highest number.

`--history <n>` prints `HISTORY <n>` instead of the newest one. `<n>` can be given with or without leading zeros: `--history 52` and `--history 052` are the same.

## The tracklist

Each track takes one line, numbered from 1 in play order:

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

## Writing to a file

A trailing path writes the tracklist to that file instead of standard output. Options go before the path.

```sh
stick-history --history 52 tracklist.txt
```

`--list` takes no output file.

## Listing the Histories

`--list` prints each non-empty History with its name and track count, tab-separated, newest last. If any History is empty, a line on standard error says how many of the Histories were left out.

`--list` and `--history` cannot be used together.

## The version

`--version` prints the version and exits. [install.md](install.md) lists the forms the output takes.

## Errors

On an error, the command prints a message to standard error, prints nothing to standard output and exits non-zero. These are errors:

- No stick is found.
- More than one stick is mounted and `--volume` is not given.
- The path given to `--volume` has no `PIONEER` folder.
- No History is non-empty.
- The value given to `--history` is not a number.
- The named History is empty or absent.
- `export.pdb` is unreadable or malformed.

An option the command does not know, or a combination it does not accept, is a usage error. The command prints the usage text to standard error and exits with status 2.

## Cleaning up titles

The tool prints titles as the stick holds them. To clean up a title or strip characters from it, pipe the output through a tool such as `sed`.

This example strips a key at the front of a title in the form `Fm - Title`:

```sh
stick-history | sed -E 's/ - [A-G][#b]?m? - / - /'
```
