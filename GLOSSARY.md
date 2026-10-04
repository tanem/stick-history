# stick-history

The tool prints the tracklist of a set from the History a Pioneer DJ player wrote to a USB stick.

## Language

**Stick**:
A mounted volume with a `PIONEER` folder at its root.
_Avoid_: USB on its own, drive, device

**History**:
The list a player writes of the tracks played in one set, named `HISTORY <nnn>`.
_Avoid_: playlist, history playlist

**Empty History**:
A History with no entries. The tool leaves it out, and `--list` says on standard error how many it left out.

**Newest History**:
The History with the highest number. Players record no date.

**Tracklist**:
The tool's output for one History, one numbered line per track.

**Set**:
One performance. It produces one History per player.
