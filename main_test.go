package main

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
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
	wantErr := "stick-history: 1 of 3 Histories is empty and not listed\n"
	if r.code != 0 || r.stdout != "HISTORY 001\t2\nHISTORY 010\t1\n" || r.stderr != wantErr {
		t.Errorf("got %+v\nwant stderr %q", r, wantErr)
	}
}

func TestHistoryFlag(t *testing.T) {
	withVolumes(t, stick(t, "USB"))
	for _, n := range []string{"1", "01", "001"} {
		if r := exec("--history", n); r.code != 0 || r.stdout != history1 || r.stderr != "" {
			t.Errorf("--history %s: got %+v", n, r)
		}
	}
	if r := exec("--history", "10"); r.code != 0 || r.stdout != history10 || r.stderr != "" {
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
}

// --list on a stick whose every History is empty prints no row and still
// exits 0.
func TestListWhenEveryHistoryIsEmpty(t *testing.T) {
	withVolumes(t, writeStick(t, "USB", pdbtest.Build(4096, []pdbtest.Table{
		{Type: 11, Pages: []pdbtest.Page{{Rows: [][]byte{
			pdbtest.HistoryPlaylistRow(1, pdbtest.ShortString("HISTORY 001")),
			pdbtest.HistoryPlaylistRow(2, pdbtest.ShortString("HISTORY 002")),
		}}}},
	})))
	r := exec("--list")
	wantErr := "stick-history: 2 of 2 Histories are empty and not listed\n"
	if r.code != 0 || r.stdout != "" || r.stderr != wantErr {
		t.Errorf("got %+v\nwant stderr %q", r, wantErr)
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

// A release build carries the version scripts/build.sh stamped into it.
// --version needs no stick.
func TestVersionOfReleaseBuild(t *testing.T) {
	withVolumes(t)
	withVersion(t, "1.2.3")
	r := exec("--version")
	if r.code != 0 || r.stdout != "stick-history 1.2.3\n" || r.stderr != "" {
		t.Errorf("got %+v", r)
	}
}

// A go install build has no stamped version. It prints the version of the
// module it was built from, without the leading v.
func TestVersionOfGoInstallBuild(t *testing.T) {
	withVolumes(t)
	withVersion(t, "")
	withBuildInfo(t, &debug.BuildInfo{Main: debug.Module{
		Path:    "github.com/tanem/stick-history",
		Version: "v0.3.1",
		Sum:     "h1:J1shsA93PJUEVaUSaay7UXAyE8aimq3GW0pjlolpa24=",
	}})
	r := exec("--version")
	if r.code != 0 || r.stdout != "stick-history 0.3.1\n" || r.stderr != "" {
		t.Errorf("got %+v", r)
	}
}

// Any other build prints a marker in place of a version: one whose module was
// not downloaded, so has no checksum, and one with no build information. Go
// 1.24 and later give a build from a checkout a version derived from the
// commit, and that is not a release either.
func TestVersionOfOtherBuilds(t *testing.T) {
	withVolumes(t)
	withVersion(t, "")
	for name, info := range map[string]*debug.BuildInfo{
		"go build, before Go 1.24": {Main: debug.Module{Version: "(devel)"}},
		"go build, from Go 1.24":   {Main: debug.Module{Version: "v0.3.2-0.20261004001122-16a9053f3c1d+dirty"}},
		"no build information":     nil,
	} {
		t.Run(name, func(t *testing.T) {
			withBuildInfo(t, info)
			r := exec("--version")
			if r.code != 0 || r.stdout != "stick-history (devel)\n" || r.stderr != "" {
				t.Errorf("got %+v", r)
			}
		})
	}
}

// withBuildInfo makes the command read info as its build information, or none
// when info is nil.
func withBuildInfo(t *testing.T, info *debug.BuildInfo) {
	t.Helper()
	old := readBuildInfo
	readBuildInfo = func() (*debug.BuildInfo, bool) { return info, info != nil }
	t.Cleanup(func() { readBuildInfo = old })
}

func withVersion(t *testing.T, v string) {
	t.Helper()
	old := version
	version = v
	t.Cleanup(func() { version = old })
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

// A path in an error message has its control characters replaced, as the
// strings of the tracklist do. On macOS an exFAT volume label can hold them,
// and the label is the name of the volume's root. An ordinary path prints as
// it is.
func TestPathInErrorMessages(t *testing.T) {
	for _, v := range []struct{ kind, name, shown string }{
		{"ordinary", "USB", "USB"},
		{"crafted", "A\x1b[2JB\n\x07C", "A�[2JB��C"},
	} {
		t.Run(v.kind, func(t *testing.T) {
			if v.kind == "crafted" && runtime.GOOS == "windows" {
				t.Skip("a Windows file name cannot hold a control character")
			}
			// shownRoot returns root as an error message prints it.
			shownRoot := func(root string) string {
				return filepath.Join(filepath.Dir(root), v.shown)
			}
			pdbPath := func(root string) string {
				return filepath.Join(root, "PIONEER", "rekordbox", "export.pdb")
			}
			// fails runs the command, which must fail and write want to
			// standard error.
			fails := func(t *testing.T, want string, args ...string) {
				t.Helper()
				if r := exec(args...); r.code != 1 || r.stdout != "" || r.stderr != want {
					t.Errorf("got %+v\nwant stderr %q", r, want)
				}
			}

			t.Run("missing export.pdb", func(t *testing.T) {
				root := stick(t, v.name)
				if err := os.Remove(pdbPath(root)); err != nil {
					t.Fatal(err)
				}
				withVolumes(t, root)
				var missing *fs.PathError
				if _, err := os.ReadFile(pdbPath(root)); !errors.As(err, &missing) {
					t.Fatalf("reading the removed file: %v", err)
				}
				fails(t, "stick-history: open "+pdbPath(shownRoot(root))+": "+missing.Err.Error()+"\n")
			})

			t.Run("malformed export.pdb", func(t *testing.T) {
				root := writeStick(t, v.name, nil)
				withVolumes(t, root)
				fails(t, "stick-history: "+pdbPath(shownRoot(root))+": file is shorter than its header\n")
			})

			t.Run("no non-empty History", func(t *testing.T) {
				root := writeStick(t, v.name, pdbtest.Build(4096, []pdbtest.Table{
					{Type: 11, Pages: []pdbtest.Page{{Rows: [][]byte{
						pdbtest.HistoryPlaylistRow(1, pdbtest.ShortString("HISTORY 001")),
					}}}},
				}))
				withVolumes(t, root)
				fails(t, "stick-history: "+shownRoot(root)+" has no non-empty History\n")
			})

			t.Run("no HISTORY n", func(t *testing.T) {
				root := stick(t, v.name)
				withVolumes(t, root)
				fails(t, "stick-history: "+shownRoot(root)+" has no HISTORY 009\n", "--history", "9")
			})

			t.Run("no PIONEER folder", func(t *testing.T) {
				root := filepath.Join(t.TempDir(), v.name)
				if err := os.Mkdir(root, 0o755); err != nil {
					t.Fatal(err)
				}
				fails(t, "stick-history: "+shownRoot(root)+" has no PIONEER folder\n", "--volume", root)
			})

			// Each root is on a line of its own, indented by two spaces.
			t.Run("more than one stick", func(t *testing.T) {
				a, b := stick(t, v.name), stick(t, v.name)
				withVolumes(t, a, b)
				fails(t, "stick-history: more than one stick is mounted; pick one with --volume:\n"+
					"  "+shownRoot(a)+"\n"+
					"  "+shownRoot(b)+"\n")
			})
		})
	}
}
