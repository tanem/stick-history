package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tanem/stick-history/internal/pdbtest"
	"github.com/tanem/stick-history/pdb"
)

// stick writes a synthetic export.pdb with three Histories under a new volume
// root and returns that root. HISTORY 002 is empty.
func stick(t *testing.T, name string) string {
	t.Helper()
	data := pdbtest.Build(4096, []pdbtest.Table{
		{Type: 2, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.ArtistRow(0x60, 1, pdbtest.ShortString("Artist One")),
		}}}},
		{Type: 0, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.TrackRow(10, 1, pdbtest.ShortString("Title A")),
			pdbtest.TrackRow(11, 7, pdbtest.UTF16String("Tïtle B")),
		}}}},
		{Type: 11, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryPlaylistRow(1, pdbtest.ShortString("HISTORY 001")),
			pdbtest.HistoryPlaylistRow(2, pdbtest.ShortString("HISTORY 002")),
			pdbtest.HistoryPlaylistRow(3, pdbtest.ShortString("HISTORY 010")),
		}}}},
		{Type: 12, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryEntryRow(11, 1, 2),
			pdbtest.HistoryEntryRow(10, 1, 1),
			pdbtest.HistoryEntryRow(10, 3, 1),
		}}}},
	})
	return writeStick(t, name, data)
}

// writeStick writes data as the export.pdb under a new volume root and
// returns that root.
func writeStick(t *testing.T, name string, data []byte) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	dir := filepath.Join(root, "PIONEER", "rekordbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "export.pdb"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func withVolumes(t *testing.T, roots ...string) {
	t.Helper()
	old := listVolumes
	listVolumes = func() []string { return roots }
	t.Cleanup(func() { listVolumes = old })
}

type result struct {
	code   int
	stdout string
	stderr string
}

func exec(args ...string) result {
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return result{code, stdout.String(), stderr.String()}
}

const (
	history1  = "1. Artist One - Title A\n2.  - Tïtle B\n"
	history10 = "1. Artist One - Title A\n"
)

func TestNewestByDefault(t *testing.T) {
	withVolumes(t, filepath.Join(t.TempDir(), "Macintosh HD"), stick(t, "USB"))
	r := exec()
	if r.code != 0 || r.stdout != history10 || r.stderr != "" {
		t.Errorf("got %+v", r)
	}
}

func TestList(t *testing.T) {
	withVolumes(t, stick(t, "USB"))
	r := exec("--list")
	if r.code != 0 || r.stdout != "HISTORY 001\t2\nHISTORY 010\t1\n" || r.stderr != "" {
		t.Errorf("got %+v", r)
	}
}

func TestHistoryFlag(t *testing.T) {
	withVolumes(t, stick(t, "USB"))
	for _, n := range []string{"1", "01", "001"} {
		if r := exec("--history", n); r.code != 0 || r.stdout != history1 {
			t.Errorf("--history %s: got %+v", n, r)
		}
	}
	if r := exec("--history", "10"); r.code != 0 || r.stdout != history10 {
		t.Errorf("--history 10: got %+v", r)
	}
	for n, msg := range map[string]string{
		"2":  "HISTORY 002 is empty",
		"9":  "no HISTORY 009",
		"x":  "needs a number",
		"-1": "needs a number",
	} {
		r := exec("--history", n)
		if r.code != 1 || r.stdout != "" || !strings.Contains(r.stderr, msg) {
			t.Errorf("--history %s: got %+v, want stderr containing %q", n, r, msg)
		}
	}
}

func TestOutputFile(t *testing.T) {
	withVolumes(t, stick(t, "USB"))
	path := filepath.Join(t.TempDir(), "set.txt")
	r := exec("--history", "1", path)
	if r.code != 0 || r.stdout != "" || r.stderr != "" {
		t.Fatalf("got %+v", r)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != history1 {
		t.Errorf("file holds %q", got)
	}
}

func TestVolumeFlag(t *testing.T) {
	a, b := stick(t, "A"), stick(t, "B")
	withVolumes(t, a, b)

	r := exec()
	if r.code != 1 || r.stdout != "" || !strings.Contains(r.stderr, a) || !strings.Contains(r.stderr, b) {
		t.Errorf("two sticks: got %+v", r)
	}
	if r := exec("--volume", b); r.code != 0 || r.stdout != history10 {
		t.Errorf("--volume: got %+v", r)
	}
	r = exec("--volume", t.TempDir())
	if r.code != 1 || !strings.Contains(r.stderr, "no PIONEER folder") {
		t.Errorf("--volume without PIONEER: got %+v", r)
	}
}

func TestNoStick(t *testing.T) {
	withVolumes(t, t.TempDir())
	r := exec()
	if r.code != 1 || r.stdout != "" || !strings.Contains(r.stderr, "no stick found") {
		t.Errorf("got %+v", r)
	}
}

func TestNoNonEmptyHistory(t *testing.T) {
	withVolumes(t, writeStick(t, "USB", pdbtest.Build(4096, []pdbtest.Table{
		{Type: 11, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryPlaylistRow(1, pdbtest.ShortString("HISTORY 001")),
		}}}},
	})))
	r := exec()
	if r.code != 1 || r.stdout != "" || !strings.Contains(r.stderr, "no non-empty History") {
		t.Errorf("got %+v", r)
	}
	if r := exec("--list"); r.code != 0 || r.stdout != "" {
		t.Errorf("--list: got %+v", r)
	}
}

// A newline in a title must not start a new line of the tracklist.
func TestNewlineInTitle(t *testing.T) {
	withVolumes(t, writeStick(t, "USB", pdbtest.Build(4096, []pdbtest.Table{
		{Type: 2, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.ArtistRow(0x60, 1, pdbtest.ShortString("Artist One")),
		}}}},
		{Type: 0, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.TrackRow(10, 1, pdbtest.ShortString("Line one\n2. Fake - Track\r\nLine three")),
			pdbtest.TrackRow(11, 1, pdbtest.ShortString("Title B")),
		}}}},
		{Type: 11, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryPlaylistRow(1, pdbtest.ShortString("HISTORY 001")),
		}}}},
		{Type: 12, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryEntryRow(10, 1, 1),
			pdbtest.HistoryEntryRow(11, 1, 2),
		}}}},
	})))
	want := "1. Artist One - Line one\ufffd2. Fake - Track\ufffd\ufffdLine three\n2. Artist One - Title B\n"
	if r := exec(); r.code != 0 || r.stdout != want || r.stderr != "" {
		t.Errorf("got %+v\nwant stdout %q", r, want)
	}
}

// A crafted stick with control characters in an artist, in titles and in a
// History name. Each one prints as U+FFFD, in the tracklist and in --list.
func TestControlCharactersReplaced(t *testing.T) {
	withVolumes(t, writeStick(t, "USB", pdbtest.Build(4096, []pdbtest.Table{
		{Type: 2, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.ArtistRow(0x60, 1, pdbtest.ShortString("Art\x1b[2Jist")),
		}}}},
		{Type: 0, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.TrackRow(10, 1, pdbtest.ShortString("Bell\x07 Tab\t Del\x7f")),
			pdbtest.TrackRow(11, 1, pdbtest.UTF16String("C1 \u009b31m, separators \u2028\u2029, kept ï☃")),
			pdbtest.TrackRow(12, 1, pdbtest.ShortString("Raw byte \x9b31m")),
		}}}},
		{Type: 11, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryPlaylistRow(1, pdbtest.LongASCII("HISTORY 001\x1b]0;title\x07\n")),
			pdbtest.HistoryPlaylistRow(2, pdbtest.ShortString("HISTORY 002")),
		}}}},
		{Type: 12, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryEntryRow(10, 1, 1),
			pdbtest.HistoryEntryRow(10, 2, 1),
			pdbtest.HistoryEntryRow(11, 2, 2),
			pdbtest.HistoryEntryRow(12, 2, 3),
		}}}},
	})))
	want := "1. Art\ufffd[2Jist - Bell\ufffd Tab\ufffd Del\ufffd\n" +
		"2. Art\ufffd[2Jist - C1 \ufffd31m, separators \ufffd\ufffd, kept ï☃\n" +
		"3. Art\ufffd[2Jist - Raw byte \ufffd31m\n"
	if r := exec(); r.code != 0 || r.stdout != want || r.stderr != "" {
		t.Errorf("tracklist: got %+v\nwant stdout %q", r, want)
	}
	want = "HISTORY 001\ufffd]0;title\ufffd\ufffd\t1\nHISTORY 002\t3\n"
	if r := exec("--list"); r.code != 0 || r.stdout != want || r.stderr != "" {
		t.Errorf("--list: got %+v\nwant stdout %q", r, want)
	}
}

func TestBadFile(t *testing.T) {
	root := stick(t, "USB")
	path := filepath.Join(root, "PIONEER", "rekordbox", "export.pdb")
	good, _ := os.ReadFile(path)
	withVolumes(t, root)
	for name, data := range map[string][]byte{
		"truncated": good[:len(good)/2],
		"random":    bytes.Repeat([]byte{0x5a, 0xa5, 0x01}, 4096),
		"empty":     {},
	} {
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
		r := exec()
		if r.code != 1 || r.stdout != "" || !strings.Contains(r.stderr, "export.pdb") {
			t.Errorf("%s: got %+v", name, r)
		}
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if r := exec(); r.code != 1 || r.stdout != "" || r.stderr == "" {
		t.Errorf("missing: got %+v", r)
	}
}

func TestUsageErrors(t *testing.T) {
	withVolumes(t, stick(t, "USB"))
	for _, args := range [][]string{
		{"a", "b"},
		{"--list", "out.txt"},
		{"--list", "--history", "1"},
		{"--bogus"},
	} {
		if r := exec(args...); r.code != 2 || r.stdout != "" || r.stderr == "" {
			t.Errorf("%v: got %+v", args, r)
		}
	}
	if r := exec("--help"); r.code != 0 || !strings.Contains(r.stderr, "Usage") {
		t.Errorf("--help: got %+v", r)
	}
}

func TestOlder(t *testing.T) {
	names := []string{"HISTORY 010", "Renamed", "HISTORY 002", "HISTORY 1"}
	got := make([]string, len(names))
	for _, a := range names {
		rank := 0
		for _, b := range names {
			if a != b && older(hist(b), hist(a)) {
				rank++
			}
		}
		got[rank] = a
	}
	want := "Renamed HISTORY 1 HISTORY 002 HISTORY 010"
	if strings.Join(got, " ") != want {
		t.Errorf("order %q, want %q", strings.Join(got, " "), want)
	}
}

func hist(name string) (h pdb.History) {
	h.Name = name
	return h
}
