package pdb

import (
	"math/rand"
	"reflect"
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

func mustNotPanic(t *testing.T, name string, data []byte) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s: Parse panicked: %v", name, r)
		}
	}()
	Parse(data)
}

func randomBytes(n int, seed int64) []byte {
	b := make([]byte, n)
	rand.New(rand.NewSource(seed)).Read(b)
	return b
}
