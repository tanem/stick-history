// Package pdbtest builds synthetic export.pdb files for tests. The files hold
// only the data the caller passes in, so no track, artist or History from a
// real collection ends up in the repo.
package pdbtest

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// NumTables is the table count written to the file header. A real export.pdb
// from rekordbox 6 has 20 tables.
const NumTables = 20

// Page is one page of a table.
type Page struct {
	// Rows are the row bodies in slot order. Each gets a row offset and a
	// presence bit.
	Rows [][]byte
	// Deleted lists slot indexes whose presence bit is cleared.
	Deleted []int
	// Index marks the page as not a data page: flag 0x40 is set and the
	// reader must skip it. The page still carries its rows, so a reader that
	// ignores the flag sees them.
	Index bool
}

// Table is one table of the file. A table not passed to Build gets one empty
// data page.
type Table struct {
	Type  uint32
	Pages []Page
}

// Build returns an export.pdb with the given tables. Page 0 is the file
// header; the tables' pages follow in the order given, chained through their
// next-page field. It panics when a page's rows do not fit, since that is a
// mistake in the test.
func Build(pageSize int, tables []Table) []byte {
	byType := map[uint32]Table{}
	for _, t := range tables {
		byType[t.Type] = t
	}
	var all []Table
	for typ := uint32(0); typ < NumTables; typ++ {
		t, ok := byType[typ]
		if !ok {
			t = Table{Type: typ, Pages: []Page{{}}}
		}
		if len(t.Pages) == 0 {
			t.Pages = []Page{{}}
		}
		all = append(all, t)
	}

	header := make([]byte, pageSize)
	binary.LittleEndian.PutUint32(header[0x04:], uint32(pageSize))
	binary.LittleEndian.PutUint32(header[0x08:], NumTables)
	var pages []byte
	next := uint32(1)
	for i, t := range all {
		first := next
		last := first + uint32(len(t.Pages)) - 1
		ptr := header[0x1c+16*i:]
		binary.LittleEndian.PutUint32(ptr[0:], t.Type)
		binary.LittleEndian.PutUint32(ptr[8:], first)
		binary.LittleEndian.PutUint32(ptr[12:], last)
		for j, p := range t.Pages {
			index := first + uint32(j)
			nextPage := index + 1
			if j == len(t.Pages)-1 {
				nextPage = 0
			}
			pages = append(pages, buildPage(pageSize, t.Type, index, nextPage, p)...)
		}
		next = last + 1
	}
	return append(header, pages...)
}

func buildPage(pageSize int, typ, index, next uint32, p Page) []byte {
	b := make([]byte, pageSize)
	binary.LittleEndian.PutUint32(b[0x04:], index)
	binary.LittleEndian.PutUint32(b[0x08:], typ)
	binary.LittleEndian.PutUint32(b[0x0c:], next)
	n := len(p.Rows)
	groups := (n + 15) / 16
	// The row count is 13 bits wide across the bytes at 0x18 and 0x19. The
	// high bits of 0x19 are flags on real sticks, so set some here to make
	// sure the reader masks them off.
	b[0x18] = byte(n)
	b[0x19] = byte(n>>8)&0x1f | 0xa0
	if p.Index {
		b[0x1b] = 0x64
	} else {
		b[0x1b] = 0x34
	}
	// Real sticks carry small, unrelated values at 0x22, where another
	// layout puts a 16-bit row count. Write one so a reader that trusts it
	// gets the wrong count.
	binary.LittleEndian.PutUint16(b[0x22:], 4)

	// Row data may run into the last row group up to its lowest used slot,
	// as on real sticks.
	heapEnd := pageSize - (groups-1)*0x24 - 6 - 2*((n-1)%16)
	if n == 0 {
		heapEnd = pageSize
	}
	off := 0x28
	deleted := map[int]bool{}
	for _, d := range p.Deleted {
		deleted[d] = true
	}
	for i, row := range p.Rows {
		if off+len(row) > heapEnd {
			panic(fmt.Sprintf("pdbtest: %d rows do not fit a %d-byte page", n, pageSize))
		}
		copy(b[off:], row)
		g, j := i/16, i%16
		end := pageSize - g*0x24
		binary.LittleEndian.PutUint16(b[end-6-2*j:], uint16(off-0x28))
		if !deleted[i] {
			present := binary.LittleEndian.Uint16(b[end-4:])
			binary.LittleEndian.PutUint16(b[end-4:], present|1<<j)
		}
		off += len(row)
	}
	return b
}

// ShortString encodes s as a short ASCII string: an odd first byte holding
// the length, including itself, shifted left once.
func ShortString(s string) []byte {
	if len(s) > 126 {
		panic("pdbtest: short string is too long")
	}
	return append([]byte{byte((len(s)+1)<<1 | 1)}, s...)
}

// LongASCII encodes s as a long ASCII string: kind byte 0x40, a u16 length
// including the 4-byte header, and an unknown byte.
func LongASCII(s string) []byte {
	b := []byte{0x40, 0, 0, 0}
	binary.LittleEndian.PutUint16(b[1:], uint16(len(s)+4))
	return append(b, s...)
}

// UTF16String encodes s as a long UTF-16LE string: kind byte 0x90.
func UTF16String(s string) []byte {
	units := utf16.Encode([]rune(s))
	b := []byte{0x90, 0, 0, 0}
	binary.LittleEndian.PutUint16(b[1:], uint16(len(units)*2+4))
	for _, u := range units {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	return b
}

// TrackRow builds a row of the tracks table. The 21 string offsets all point
// at an empty string except the title, which is the encoded string given.
func TrackRow(id, artistID uint32, title []byte) []byte {
	const fixed = 0x5e + 21*2
	row := make([]byte, fixed)
	binary.LittleEndian.PutUint32(row[0x44:], artistID)
	binary.LittleEndian.PutUint32(row[0x48:], id)
	empty := ShortString("")
	for i := 0; i < 21; i++ {
		binary.LittleEndian.PutUint16(row[0x5e+2*i:], uint16(fixed))
	}
	binary.LittleEndian.PutUint16(row[0x5e+2*17:], uint16(fixed+len(empty)))
	row = append(row, empty...)
	return append(row, title...)
}

// ArtistRow builds a row of the artists table. Subtype 0x60 keeps the name
// offset in a byte at 0x09; subtype 0x64 keeps it in a u16 at 0x0a and the
// byte at 0x09 holds a value that points outside the row.
func ArtistRow(subtype uint16, id uint32, name []byte) []byte {
	var row []byte
	switch subtype {
	case 0x60:
		row = make([]byte, 0x0a)
		row[0x09] = 0x0a
	case 0x64:
		row = make([]byte, 0x0c)
		row[0x09] = 0xff
		binary.LittleEndian.PutUint16(row[0x0a:], 0x0c)
	default:
		panic("pdbtest: unknown artist subtype")
	}
	binary.LittleEndian.PutUint16(row[0:], subtype)
	binary.LittleEndian.PutUint32(row[0x04:], id)
	return append(row, name...)
}

// HistoryPlaylistRow builds a row of the history playlists table.
func HistoryPlaylistRow(id uint32, name []byte) []byte {
	row := make([]byte, 4)
	binary.LittleEndian.PutUint32(row, id)
	return append(row, name...)
}

// HistoryEntryRow builds a row of the history entries table.
func HistoryEntryRow(trackID, playlistID, index uint32) []byte {
	row := make([]byte, 12)
	binary.LittleEndian.PutUint32(row[0:], trackID)
	binary.LittleEndian.PutUint32(row[4:], playlistID)
	binary.LittleEndian.PutUint32(row[8:], index)
	return row
}
