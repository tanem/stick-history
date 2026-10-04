# What the tool reads

`export.pdb` is the database rekordbox writes to a stick when it exports a collection. It is at `PIONEER/rekordbox/export.pdb` on the stick. The player writes each set's History into it.

The tool reads four tables: tracks, artists, history playlists and history entries. It follows the [Deep Symmetry analysis](https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html) of the format and depends on the Go standard library alone.

The `pdb` package does the reading and can be used without the command. Its documentation is on [pkg.go.dev](https://pkg.go.dev/github.com/tanem/stick-history/pdb), with the limits it puts on a file.

## What has been verified

- The tool was verified on an XDJ-700 with a stick exported by rekordbox 6. Sticks exported by rekordbox 7 have not been checked.
- Artist rows of subtype `0x64` did not occur on that stick. Their layout follows the Deep Symmetry analysis and is covered by the synthetic test fixture only.

## What the tool does not do

- Newer players also write a Device Library Plus database, `exportLibrary.db`. The tool does not read it.
- One stick per set. Histories from two sticks are not merged.

## A History can be short or empty

A History can hold fewer tracks than were played, or none. The tool prints what the stick holds.

On one stick, rekordbox had synced the collection after the sets. The tracks it removed from the stick were gone from the stick's track table, and so were the History entries that pointed at them. 122 of 124 Histories were empty, and the other two held 4 and 5 tracks.

This was seen on that one stick and has not been checked against rekordbox directly.
