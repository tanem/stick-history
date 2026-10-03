package pdb

import (
	"encoding/binary"
	"math/rand"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/tanem/stick-history/internal/pdbtest"
)

const pageSize = 4096

// fixture is a file with every case the reader has to handle: both artist
// subtypes, short ASCII, long ASCII and UTF-16 strings, a track without an artist row, an
// empty History, a deleted row, an index page and a page chain longer than
// one page.
func fixture() []byte {
	return pdbtest.Build(pageSize, []pdbtest.Table{
		{Type: 2, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.ArtistRow(0x60, 1, pdbtest.ShortString("Near Artist")),
			pdbtest.ArtistRow(0x64, 2, pdbtest.LongASCII("Far Artist")),
			pdbtest.ArtistRow(0x60, 3, pdbtest.UTF16String("Ärtíst Þree")),
		}}}},
		{Type: 0, Pages: []pdbtest.Page{
			{Rows: [][]byte{
				pdbtest.TrackRow(10, 1, pdbtest.ShortString("Short Title")),
				pdbtest.TrackRow(11, 2, pdbtest.UTF16String("Tïtle – ☃")),
				pdbtest.TrackRow(12, 99, pdbtest.LongASCII("No Artist Row")),
				pdbtest.TrackRow(13, 1, pdbtest.ShortString("Deleted")),
			}, Deleted: []int{3}},
			{Index: true, Rows: [][]byte{
				pdbtest.TrackRow(14, 1, pdbtest.ShortString("On Index Page")),
			}},
			{Rows: [][]byte{
				pdbtest.TrackRow(15, 3, pdbtest.ShortString("Second Page")),
			}},
		}},
		{Type: 11, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryPlaylistRow(3, pdbtest.ShortString("HISTORY 003")),
			pdbtest.HistoryPlaylistRow(1, pdbtest.ShortString("HISTORY 001")),
			pdbtest.HistoryPlaylistRow(2, pdbtest.ShortString("HISTORY 002")),
		}}}},
		{Type: 12, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryEntryRow(12, 1, 3),
			pdbtest.HistoryEntryRow(10, 1, 1),
			pdbtest.HistoryEntryRow(15, 3, 1),
			pdbtest.HistoryEntryRow(11, 1, 2),
			pdbtest.HistoryEntryRow(15, 1, 4),
			pdbtest.HistoryEntryRow(13, 3, 2),
			pdbtest.HistoryEntryRow(14, 3, 3),
		}}}},
	})
}

func TestParseFixture(t *testing.T) {
	e, err := Parse(fixture())
	if err != nil {
		t.Fatal(err)
	}
	want := []History{
		{ID: 1, Name: "HISTORY 001", Tracks: []Track{
			{"Near Artist", "Short Title"},
			{"Far Artist", "Tïtle – ☃"},
			{"", "No Artist Row"},
			{"Ärtíst Þree", "Second Page"},
		}},
		{ID: 2, Name: "HISTORY 002"},
		{ID: 3, Name: "HISTORY 003", Tracks: []Track{
			{"Ärtíst Þree", "Second Page"},
			{"", ""}, // deleted track row
			{"", ""}, // track row on an index page
		}},
	}
	if got := e.Histories(); !reflect.DeepEqual(got, want) {
		t.Errorf("Histories() = %#v\nwant %#v", got, want)
	}
}

// Parse returns control characters as the file stores them. Replacing them is
// left to the caller that prints the strings.
func TestControlCharactersKept(t *testing.T) {
	const (
		artist = "Art\x1b[2Jist\x7f"
		title  = "Line one\nLine\ttwo\u009b"
		name   = "HISTORY\r001\x07"
	)
	data := pdbtest.Build(pageSize, []pdbtest.Table{
		{Type: 2, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.ArtistRow(0x60, 1, pdbtest.ShortString(artist)),
		}}}},
		{Type: 0, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.TrackRow(10, 1, pdbtest.UTF16String(title)),
		}}}},
		{Type: 11, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryPlaylistRow(1, pdbtest.LongASCII(name)),
		}}}},
		{Type: 12, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryEntryRow(10, 1, 1),
		}}}},
	})
	e, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	want := []History{{ID: 1, Name: name, Tracks: []Track{{artist, title}}}}
	if got := e.Histories(); !reflect.DeepEqual(got, want) {
		t.Errorf("Histories() = %q\nwant %q", got, want)
	}
}

// A 4096-byte page holds 284 history entry rows, more than fit in the byte at
// 0x18 alone.
func TestFullEntriesPage(t *testing.T) {
	const n = 284
	var rows [][]byte
	for i := uint32(0); i < n; i++ {
		rows = append(rows, pdbtest.HistoryEntryRow(1, 1, n-i))
	}
	data := pdbtest.Build(pageSize, []pdbtest.Table{
		{Type: 0, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.TrackRow(1, 0, pdbtest.ShortString("T")),
		}}}},
		{Type: 11, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryPlaylistRow(1, pdbtest.ShortString("HISTORY 001")),
		}}}},
		{Type: 12, Pages: []pdbtest.Page{{Rows: rows, Deleted: []int{0, 283}}}},
	})
	e, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	hs := e.Histories()
	if len(hs) != 1 || len(hs[0].Tracks) != n-2 {
		t.Fatalf("got %d Histories, first with %d tracks; want 1 with %d", len(hs), len(hs[0].Tracks), n-2)
	}
}

// A row may extend past the start of the last row group, into the slots that
// group does not use. A single 4050-byte row fills a 4096-byte page up to the
// one used slot of its only group.
func TestRowRunsIntoLastGroup(t *testing.T) {
	title := strings.Repeat("x", 4050-(0x5e+21*2)-1-4)
	data := pdbtest.Build(pageSize, []pdbtest.Table{
		{Type: 0, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.TrackRow(1, 0, pdbtest.LongASCII(title)),
		}}}},
		{Type: 11, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryPlaylistRow(1, pdbtest.ShortString("HISTORY 001")),
		}}}},
		{Type: 12, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryEntryRow(1, 1, 1),
		}}}},
	})
	e, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := e.Histories()[0].Tracks[0].Title; got != title {
		t.Errorf("title has %d bytes, want %d", len(got), len(title))
	}
}

// sharedName is a crafted file whose artists table has n rows that all point
// at one name of size bytes. The rows are 12 bytes each and the name follows
// the last one, so the file stores the name once and Parse decodes it n times.
func sharedName(n, size int) []byte {
	rows := make([][]byte, n)
	for i := range rows {
		rows[i] = pdbtest.ArtistRow(0x64, uint32(i+1), nil)
		binary.LittleEndian.PutUint16(rows[i][0x0a:], uint16(12*(n-i)))
	}
	rows[n-1] = append(rows[n-1], pdbtest.LongASCII(strings.Repeat("x", size))...)
	return pdbtest.Build(1<<16, []pdbtest.Table{
		{Type: 2, Pages: []pdbtest.Page{{Rows: rows}}},
	})
}

// checkAllocated fails the test when Parse allocates more than factor times
// the size of data.
func checkAllocated(t *testing.T, data []byte, factor uint64) {
	t.Helper()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, err := Parse(data)
	runtime.ReadMemStats(&after)
	allocated := after.TotalAlloc - before.TotalAlloc
	t.Logf("the file is %d bytes and Parse allocated %d bytes, %.1f times the file; Parse returned: %v",
		len(data), allocated, float64(allocated)/float64(len(data)), err)
	if allocated > factor*uint64(len(data)) {
		t.Errorf("Parse allocated more than %d times the file size", factor)
	}
}

// Parse must not allocate many times the size of the file it is given.
func TestSharedNameMemory(t *testing.T) {
	checkAllocated(t, sharedName(2000, 36000), 4)
}

// The file is 1,376,256 bytes, so it may decode to 2,752,512 bytes of strings:
// 76 names of 36,000 bytes fit and 77 do not.
func TestStringBudget(t *testing.T) {
	if _, err := Parse(sharedName(76, 36000)); err != nil {
		t.Errorf("76 names: %v", err)
	}
	_, err := Parse(sharedName(77, 36000))
	if err == nil || !strings.Contains(err.Error(), "decoded strings exceed") {
		t.Errorf("77 names: got %v, want an error containing %q", err, "decoded strings exceed")
	}
}

// sharedEntry is a crafted file whose history entries table has one page for
// each count. A page has one 12-byte entry row and that many present row
// offsets, all pointing at it, so the file stores the entry once per page and
// Parse reads it once per offset.
func sharedEntry(pageSize int, counts ...int) []byte {
	pages := make([]pdbtest.Page, len(counts))
	for i, n := range counts {
		pages[i] = pdbtest.Page{
			Rows:          [][]byte{pdbtest.HistoryEntryRow(1, 1, 1)},
			SharedOffsets: n - 1,
		}
	}
	return pdbtest.Build(pageSize, []pdbtest.Table{
		{Type: 11, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryPlaylistRow(1, pdbtest.ShortString("HISTORY 001")),
		}}}},
		{Type: 12, Pages: pages},
	})
}

// manySharedEntries is a 1,433,600-byte file with 50 pages in the history
// entries table, each with 8,191 row offsets, the most a page header can
// declare. That is 409,550 history entries for 50 entry rows.
func manySharedEntries() []byte {
	counts := make([]int, 50)
	for i := range counts {
		counts[i] = 8191
	}
	return sharedEntry(20480, counts...)
}

func TestManySharedEntries(t *testing.T) {
	_, err := Parse(manySharedEntries())
	if err == nil || !strings.Contains(err.Error(), "history entries exceed") {
		t.Errorf("got %v, want an error containing %q", err, "history entries exceed")
	}
}

// Parse reads 119,466 entries before it returns the error. With go1.23.3 on
// darwin/arm64 it allocated about 5.2 MB, 3.6 times the file, most of it in
// growing the slice of entries. Without the limit it allocated about 86.8 MB,
// 60.6 times the file. The bound of 8 times leaves room for a Go version that
// grows slices differently.
func TestManySharedEntriesMemory(t *testing.T) {
	checkAllocated(t, manySharedEntries(), 8)
}

// A file of 25 pages of 4,096 bytes is 102,400 bytes, so it may hold 8,533
// history entries.
func TestEntryLimit(t *testing.T) {
	if _, err := Parse(sharedEntry(pageSize, 1707, 1707, 1707, 1707, 1705)); err != nil {
		t.Errorf("8533 entries: %v", err)
	}
	_, err := Parse(sharedEntry(pageSize, 1707, 1707, 1707, 1707, 1706))
	if err == nil || !strings.Contains(err.Error(), "history entries exceed 8533") {
		t.Errorf("8534 entries: got %v, want an error containing %q", err, "history entries exceed 8533")
	}
}

func TestParseString(t *testing.T) {
	cases := []struct {
		name string
		b    []byte
		want string
		err  string
	}{
		{"short", pdbtest.ShortString("abc"), "abc", ""},
		{"short empty", pdbtest.ShortString(""), "", ""},
		{"long ascii", pdbtest.LongASCII("abc"), "abc", ""},
		{"utf16", pdbtest.UTF16String("ab☃"), "ab☃", ""},
		{"utf16 trailing nul", append(pdbtest.UTF16String("ab\x00"), 0), "ab", ""},
		{"short runs past row", []byte{0x09, 'a'}, "", "runs past"},
		{"short no length", []byte{0x01}, "", "no length"},
		{"long header cut", []byte{0x40, 0x05}, "", "header runs past"},
		{"long too short", []byte{0x40, 0x02, 0x00, 0x00}, "", "shorter than its header"},
		{"long runs past", []byte{0x40, 0x09, 0x00, 0x00, 'a'}, "", "runs past"},
		{"utf16 odd", []byte{0x90, 0x05, 0x00, 0x00, 'a'}, "", "odd byte"},
		{"empty row", nil, "", "outside"},
	}
	for _, c := range cases {
		got, err := parseString(c.b, 0)
		if c.err == "" {
			if err != nil || got != c.want {
				t.Errorf("%s: got %q, %v; want %q", c.name, got, err, c.want)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), c.err) {
			t.Errorf("%s: got %q, %v; want error containing %q", c.name, got, err, c.err)
		}
	}
}

func TestMalformed(t *testing.T) {
	good := fixture()
	cases := map[string][]byte{
		"empty":        {},
		"header only":  good[:0x1c],
		"half a file":  good[:len(good)/2],
		"odd length":   good[:len(good)-1],
		"first page":   good[:pageSize],
		"random bytes": randomBytes(len(good), 1),
	}
	for name, data := range cases {
		if _, err := Parse(data); err == nil {
			t.Errorf("%s: Parse returned no error", name)
		}
	}
}

// Every prefix and every single-byte corruption of the fixture must parse or
// return an error, never panic.
func TestNoPanic(t *testing.T) {
	good := fixture()
	for n := 0; n <= len(good); n += 37 {
		mustNotPanic(t, "prefix", good[:n])
	}
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 20000; i++ {
		data := append([]byte(nil), good...)
		for k := 0; k < 1+r.Intn(4); k++ {
			data[r.Intn(len(data))] = byte(r.Intn(256))
		}
		mustNotPanic(t, "corrupted", data)
	}
	for i := 0; i < 200; i++ {
		mustNotPanic(t, "random", randomBytes(pageSize*(1+r.Intn(4)), int64(i)))
	}
}

// FuzzParse checks that Parse returns either an Export or an error for any
// input. go test runs it on the seed; go test -fuzz=FuzzParse ./pdb fuzzes.
func FuzzParse(f *testing.F) {
	f.Add(fixture())
	f.Fuzz(func(t *testing.T, data []byte) {
		e, err := Parse(data)
		if (e == nil) == (err == nil) {
			t.Errorf("Parse returned %v, %v", e, err)
		}
	})
}

func mustNotPanic(t *testing.T, name string, data []byte) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s: Parse panicked: %v", name, r)
		}
	}()
	_, _ = Parse(data)
}

func randomBytes(n int, seed int64) []byte {
	b := make([]byte, n)
	rand.New(rand.NewSource(seed)).Read(b)
	return b
}
