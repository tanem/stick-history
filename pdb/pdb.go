// Package pdb reads the History playlists of a rekordbox export.pdb file,
// the database a Pioneer DJ player writes to a USB stick.
//
// Only the tracks, artists, history playlists and history entries tables are
// read. The layout follows the Deep Symmetry analysis of the format:
// https://djl-analysis.deepsymmetry.org/rekordbox-export-analysis/exports.html
package pdb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"
	"unicode/utf16"
)

// Table types, as recorded in the table pointers of the file header.
const (
	tableTracks           = 0
	tableArtists          = 2
	tableHistoryPlaylists = 11
	tableHistoryEntries   = 12
)

const (
	fileHeaderLen = 0x1c // bytes before the table pointers
	tablePtrLen   = 16   // type, empty candidate, first page, last page
	pageHeaderLen = 0x28 // the heap starts here
	rowGroupLen   = 0x24 // 16 row offsets, a presence bitmask and two unknown bytes
	rowsPerGroup  = 16
	maxPageSize   = 1 << 20

	trackRowLen        = 0x5e + 21*2 // fixed fields then 21 string offsets
	trackTitleIndex    = 17
	artistRowNearLen   = 0x0a
	artistRowFarLen    = 0x0c
	artistSubtypeFar   = 0x64
	historyEntryRowLen = 12

	stringUTF16 = 0x90 // kind byte of a long string holding UTF-16LE
)

// stringBudget is how many bytes of decoded strings Parse accepts for each
// byte of the file. A file that stores every string once stays under 1.5: the
// largest growth is a 2-byte UTF-16 unit that becomes 3 bytes of UTF-8. A file
// that points many rows at one string can decode to thousands of times its
// size.
const stringBudget = 2

// minPlaylistLimit is the lowest limit Parse puts on history playlists. A file
// with few tracks has few pages and may have more history playlists than
// pages, since each set on a player adds a history playlist.
const minPlaylistLimit = 4096

// Track is one entry of a History.
type Track struct {
	// Artist is empty when the artist table has no row for the track's artist.
	Artist string
	Title  string
}

// History is one History playlist with its tracks in play order.
type History struct {
	ID     uint32
	Name   string
	Tracks []Track
}

// Export is a parsed export.pdb.
type Export struct {
	histories []History
}

// Open reads and parses the export.pdb at path.
func Open(path string) (*Export, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	e, err := Parse(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return e, nil
}

// Histories returns every History in the file, including the empty ones,
// ordered by id. The slice is the caller's to reorder.
func (e *Export) Histories() []History {
	return append([]History(nil), e.histories...)
}

// Parse parses the contents of an export.pdb. A malformed file produces an
// error, never a panic. So does a file whose strings decode to more than
// twice its size, which no file that stores each string once can do, and a
// file with more history entries than one for each 12 bytes of its size,
// which no file that stores each entry once can have. So does a file with
// more history playlists than 4,096 or its page count, whichever is larger.
// That limit is not a physical one: it assumes a file a player writes stays
// under it.
func Parse(data []byte) (*Export, error) {
	f, err := parseFile(data)
	if err != nil {
		return nil, err
	}

	limit := stringBudget * uint64(len(data))
	var decoded uint64
	spend := func(s string) error {
		decoded += uint64(len(s))
		if decoded > limit {
			return fmt.Errorf("decoded strings exceed %d bytes, %d times the file size", limit, stringBudget)
		}
		return nil
	}

	artists := map[uint32]string{}
	err = f.eachRow(tableArtists, func(row []byte) error {
		id, name, err := parseArtistRow(row)
		if err != nil {
			return err
		}
		artists[id] = name
		return spend(name)
	})
	if err != nil {
		return nil, err
	}

	type trackRow struct {
		artistID uint32
		title    string
	}
	tracks := map[uint32]trackRow{}
	err = f.eachRow(tableTracks, func(row []byte) error {
		id, artistID, title, err := parseTrackRow(row)
		if err != nil {
			return err
		}
		tracks[id] = trackRow{artistID, title}
		return spend(title)
	})
	if err != nil {
		return nil, err
	}

	var histories []History
	byID := map[uint32]int{}
	// A file that fills its pages with 5-byte rows, each with its own id and
	// an empty name, can have thousands of history playlists for each page,
	// and the Export keeps a History for each one. The limit is the page
	// count, or minPlaylistLimit for a file with fewer pages than that. It
	// assumes a file a player writes stays under it.
	playlistLimit := max(len(data)/f.pageSize, minPlaylistLimit)
	var numPlaylists int
	err = f.eachRow(tableHistoryPlaylists, func(row []byte) error {
		numPlaylists++
		if numPlaylists > playlistLimit {
			return fmt.Errorf("history playlists exceed %d, the limit for a file of this size", playlistLimit)
		}
		id, name, err := parseHistoryPlaylistRow(row)
		if err != nil {
			return err
		}
		if err := spend(name); err != nil {
			return err
		}
		if _, dup := byID[id]; dup {
			return nil
		}
		byID[id] = len(histories)
		histories = append(histories, History{ID: id, Name: name})
		return nil
	})
	if err != nil {
		return nil, err
	}

	type entry struct {
		trackID uint32
		index   uint32
	}
	entries := map[uint32][]entry{}
	// A file that stores every history entry once has at most one for each
	// historyEntryRowLen bytes. A file that points many row offsets at one
	// entry row can have thousands of entries for each row.
	entryLimit := len(data) / historyEntryRowLen
	var numEntries int
	err = f.eachRow(tableHistoryEntries, func(row []byte) error {
		numEntries++
		if numEntries > entryLimit {
			return fmt.Errorf("history entries exceed %d, the most the file can hold", entryLimit)
		}
		if len(row) < historyEntryRowLen {
			return errors.New("history entry row is short")
		}
		trackID := binary.LittleEndian.Uint32(row[0:])
		playlistID := binary.LittleEndian.Uint32(row[4:])
		index := binary.LittleEndian.Uint32(row[8:])
		entries[playlistID] = append(entries[playlistID], entry{trackID, index})
		return nil
	})
	if err != nil {
		return nil, err
	}

	for i := range histories {
		es := entries[histories[i].ID]
		sort.SliceStable(es, func(a, b int) bool { return es[a].index < es[b].index })
		for _, e := range es {
			t := tracks[e.trackID]
			histories[i].Tracks = append(histories[i].Tracks, Track{
				Artist: artists[t.artistID],
				Title:  t.title,
			})
		}
	}
	sort.Slice(histories, func(a, b int) bool { return histories[a].ID < histories[b].ID })
	return &Export{histories: histories}, nil
}

// file is the page structure of an export.pdb.
type file struct {
	data     []byte
	pageSize int
	tables   []tablePtr
}

type tablePtr struct {
	typ       uint32
	firstPage uint32
	lastPage  uint32
}

func parseFile(data []byte) (*file, error) {
	if len(data) < fileHeaderLen {
		return nil, errors.New("file is shorter than its header")
	}
	pageSize := binary.LittleEndian.Uint32(data[0x04:])
	numTables := binary.LittleEndian.Uint32(data[0x08:])
	if pageSize < pageHeaderLen+rowGroupLen || pageSize > maxPageSize {
		return nil, fmt.Errorf("page size %d is not usable", pageSize)
	}
	if len(data)%int(pageSize) != 0 {
		return nil, fmt.Errorf("file size %d is not a multiple of the page size %d", len(data), pageSize)
	}
	if numTables == 0 || fileHeaderLen+uint64(numTables)*tablePtrLen > uint64(pageSize) {
		return nil, fmt.Errorf("table count %d does not fit the header page", numTables)
	}
	f := &file{data: data, pageSize: int(pageSize)}
	numPages := uint32(len(data) / f.pageSize)
	for i := uint32(0); i < numTables; i++ {
		p := data[fileHeaderLen+i*tablePtrLen:]
		t := tablePtr{
			typ:       binary.LittleEndian.Uint32(p[0:]),
			firstPage: binary.LittleEndian.Uint32(p[8:]),
			lastPage:  binary.LittleEndian.Uint32(p[12:]),
		}
		if t.firstPage == 0 || t.firstPage >= numPages || t.lastPage == 0 || t.lastPage >= numPages {
			return nil, fmt.Errorf("table %d points outside the file", t.typ)
		}
		f.tables = append(f.tables, t)
	}
	return f, nil
}

// eachRow calls fn with every present row of the table of the given type,
// walking its page chain from the first page to the last.
func (f *file) eachRow(typ uint32, fn func(row []byte) error) error {
	var t *tablePtr
	for i := range f.tables {
		if f.tables[i].typ == typ {
			t = &f.tables[i]
			break
		}
	}
	if t == nil {
		return fmt.Errorf("the file has no table of type %d", typ)
	}
	visited := map[uint32]bool{}
	index := t.firstPage
	for {
		if visited[index] {
			return fmt.Errorf("table %d: page chain loops at page %d", typ, index)
		}
		visited[index] = true
		p, err := f.page(index)
		if err != nil {
			return fmt.Errorf("table %d: %w", typ, err)
		}
		if err := p.eachRow(fn); err != nil {
			return fmt.Errorf("table %d, page %d: %w", typ, index, err)
		}
		if index == t.lastPage {
			return nil
		}
		index = p.next
	}
}

type page struct {
	data    []byte
	next    uint32
	numRows int  // row offsets in the row groups, present or not
	isData  bool // a page with flag 0x40 set holds no rows
}

func (f *file) page(index uint32) (page, error) {
	start := uint64(index) * uint64(f.pageSize)
	if index == 0 || start+uint64(f.pageSize) > uint64(len(f.data)) {
		return page{}, fmt.Errorf("page %d is outside the file", index)
	}
	b := f.data[start : start+uint64(f.pageSize)]
	// The low 13 bits of the three bytes at 0x18 hold the row count. The
	// high bits of the byte at 0x19 are flags of unknown meaning.
	n := uint32(b[0x18]) | uint32(b[0x19])<<8 | uint32(b[0x1a])<<16
	return page{
		data:    b,
		next:    binary.LittleEndian.Uint32(b[0x0c:]),
		numRows: int(n & 0x1fff),
		isData:  b[0x1b]&0x40 == 0,
	}, nil
}

// eachRow calls fn with every row whose presence bit is set. The row slice
// runs from the row's offset to the end of the heap, so a row parser must
// check its own length.
func (p page) eachRow(fn func(row []byte) error) error {
	if !p.isData || p.numRows == 0 {
		return nil
	}
	// The heap may run into the last row group as far as its lowest used
	// row offset: a group with fewer than 16 rows leaves its unused slots to
	// row data.
	groups := (p.numRows + rowsPerGroup - 1) / rowsPerGroup
	lastGroupEnd := len(p.data) - (groups-1)*rowGroupLen
	heapEnd := lastGroupEnd - 6 - 2*((p.numRows-1)%rowsPerGroup)
	if heapEnd < pageHeaderLen {
		return fmt.Errorf("%d row offsets do not fit the page", p.numRows)
	}
	for i := 0; i < p.numRows; i++ {
		g, j := i/rowsPerGroup, i%rowsPerGroup
		end := len(p.data) - g*rowGroupLen
		present := binary.LittleEndian.Uint16(p.data[end-4:])
		if present>>j&1 == 0 {
			continue
		}
		off := pageHeaderLen + int(binary.LittleEndian.Uint16(p.data[end-6-2*j:]))
		if off >= heapEnd {
			return fmt.Errorf("row %d starts outside the heap", i)
		}
		if err := fn(p.data[off:heapEnd]); err != nil {
			return fmt.Errorf("row %d: %w", i, err)
		}
	}
	return nil
}

func parseTrackRow(row []byte) (id, artistID uint32, title string, err error) {
	if len(row) < trackRowLen {
		return 0, 0, "", errors.New("track row is short")
	}
	artistID = binary.LittleEndian.Uint32(row[0x44:])
	id = binary.LittleEndian.Uint32(row[0x48:])
	titleOff := binary.LittleEndian.Uint16(row[0x5e+trackTitleIndex*2:])
	title, err = parseString(row, int(titleOff))
	if err != nil {
		return 0, 0, "", fmt.Errorf("track %d title: %w", id, err)
	}
	return id, artistID, title, nil
}

func parseArtistRow(row []byte) (id uint32, name string, err error) {
	if len(row) < artistRowNearLen {
		return 0, "", errors.New("artist row is short")
	}
	subtype := binary.LittleEndian.Uint16(row[0:])
	id = binary.LittleEndian.Uint32(row[0x04:])
	var nameOff int
	if subtype == artistSubtypeFar {
		if len(row) < artistRowFarLen {
			return 0, "", errors.New("artist row is short")
		}
		nameOff = int(binary.LittleEndian.Uint16(row[0x0a:]))
	} else {
		nameOff = int(row[0x09])
	}
	name, err = parseString(row, nameOff)
	if err != nil {
		return 0, "", fmt.Errorf("artist %d name: %w", id, err)
	}
	return id, name, nil
}

func parseHistoryPlaylistRow(row []byte) (id uint32, name string, err error) {
	if len(row) < 4 {
		return 0, "", errors.New("history playlist row is short")
	}
	id = binary.LittleEndian.Uint32(row[0:])
	name, err = parseString(row, 4)
	if err != nil {
		return 0, "", fmt.Errorf("history playlist %d name: %w", id, err)
	}
	return id, name, nil
}

// parseString decodes the string at off in b. An odd first byte is a short
// ASCII string whose length, including that byte, is the byte shifted right
// once. Otherwise a u16 at off+1 gives the length including a 4-byte header,
// and a first byte of 0x90 means the body is UTF-16LE.
func parseString(b []byte, off int) (string, error) {
	if off < 0 || off >= len(b) {
		return "", errors.New("string starts outside the row")
	}
	kind := b[off]
	if kind&1 == 1 {
		n := int(kind >> 1)
		if n < 1 {
			return "", errors.New("short string has no length byte")
		}
		if off+n > len(b) {
			return "", errors.New("short string runs past the row")
		}
		return trimNUL(b[off+1 : off+n]), nil
	}
	if off+4 > len(b) {
		return "", errors.New("long string header runs past the row")
	}
	n := int(binary.LittleEndian.Uint16(b[off+1:]))
	if n < 4 {
		return "", errors.New("long string is shorter than its header")
	}
	if off+n > len(b) {
		return "", errors.New("long string runs past the row")
	}
	body := b[off+4 : off+n]
	if kind != stringUTF16 {
		return trimNUL(body), nil
	}
	if len(body)%2 != 0 {
		return "", errors.New("UTF-16 string has an odd byte count")
	}
	u := make([]uint16, len(body)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(body[2*i:])
	}
	for len(u) > 0 && u[len(u)-1] == 0 {
		u = u[:len(u)-1]
	}
	return string(utf16.Decode(u)), nil
}

func trimNUL(b []byte) string {
	return string(bytes.TrimRight(b, "\x00"))
}
