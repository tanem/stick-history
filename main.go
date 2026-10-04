// Command stick-history prints the tracklist of a set from the History a
// Pioneer DJ player wrote to a USB stick.
//
// With no arguments it finds the mounted stick, reads
// PIONEER/rekordbox/export.pdb and prints the newest non-empty History, one
// line per track as "N. Artist - Title".
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/tanem/stick-history/pdb"
)

const usage = `Usage: stick-history [--list] [--history <n>] [--volume <path>] [<file>]
       stick-history --version

Prints the newest non-empty History on the mounted stick as a tracklist, one
line per track, "N. Artist - Title". With <file>, writes it there instead of
standard output.

`

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// version is the version of a release build, without a leading v.
// scripts/build.sh sets it with -ldflags. It is empty in any other build.
var version string

// readBuildInfo returns the build information of the running binary. Tests
// replace it.
var readBuildInfo = debug.ReadBuildInfo

// listVolumes returns the directories that may be the root of a stick. Tests
// replace it.
var listVolumes = volumeRoots

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("stick-history", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		_, _ = fmt.Fprint(stderr, usage)
		fs.PrintDefaults()
	}
	list := fs.Bool("list", false, "print each non-empty History with its track count, newest last")
	number := fs.String("history", "", "print HISTORY <n> instead of the newest non-empty History")
	volume := fs.String("volume", "", "the stick to read when more than one is mounted")
	showVersion := fs.Bool("version", false, "print the version and exit")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *showVersion {
		if _, err := fmt.Fprintf(stdout, "stick-history %s\n", buildVersion()); err != nil {
			return fail(stderr, err)
		}
		return 0
	}
	var outPath string
	switch rest := fs.Args(); {
	case len(rest) > 1:
		return usageError(fs, "at most one output file")
	case len(rest) == 1 && *list:
		return usageError(fs, "--list takes no output file")
	case len(rest) == 1:
		outPath = rest[0]
	}
	if *list && *number != "" {
		return usageError(fs, "--list and --history exclude each other")
	}

	root, err := findStick(*volume)
	if err != nil {
		return fail(stderr, err)
	}
	path := filepath.Join(root, "PIONEER", "rekordbox", "export.pdb")
	export, err := pdb.Open(path)
	if err != nil {
		return fail(stderr, openError(err, path))
	}
	histories := export.Histories()
	sort.SliceStable(histories, func(a, b int) bool { return older(histories[a], histories[b]) })

	var out bytes.Buffer
	switch {
	case *list:
		for _, h := range histories {
			if len(h.Tracks) > 0 {
				fmt.Fprintf(&out, "%s\t%d\n", printable(h.Name), len(h.Tracks))
			}
		}
	default:
		h, err := choose(histories, *number, root)
		if err != nil {
			return fail(stderr, err)
		}
		for i, t := range h.Tracks {
			fmt.Fprintf(&out, "%d. %s - %s\n", i+1, printable(t.Artist), printable(t.Title))
		}
	}

	if outPath == "" {
		_, err = stdout.Write(out.Bytes())
	} else {
		err = os.WriteFile(outPath, out.Bytes(), 0o644)
	}
	if err != nil {
		return fail(stderr, err)
	}
	return 0
}

// buildVersion returns the version --version prints: the stamped version of a
// release build, the module version of a go install build, and "(devel)" for
// any other build. A go install build is told apart by the checksum of its
// main module, which only a downloaded module has. A build from a checkout has
// none, whatever version Go derived for it from the commit.
func buildVersion() string {
	if version != "" {
		return version
	}
	if info, ok := readBuildInfo(); ok && info.Main.Sum != "" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return "(devel)"
}

func usageError(fs *flag.FlagSet, msg string) int {
	_, _ = fmt.Fprintf(fs.Output(), "stick-history: %s\n", msg)
	fs.Usage()
	return 2
}

func fail(stderr io.Writer, err error) int {
	_, _ = fmt.Fprintf(stderr, "stick-history: %v\n", err)
	return 1
}

// openError returns the error pdb.Open returned for path, with the path in its
// text passed through printable. The error is either the *fs.PathError of a
// failed read or a parse error that pdb.Open prefixed with the path.
func openError(err error, path string) error {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return fmt.Errorf("%s %s: %w", pathErr.Op, printable(pathErr.Path), pathErr.Err)
	}
	return fmt.Errorf("%s: %w", printable(path), errors.Unwrap(err))
}

// printable returns s with each control character replaced by U+FFFD. The
// strings come from the stick, as does the name of its root, and a control
// character printed as stored could start a new line or send an escape sequence
// to the terminal. The Unicode line and paragraph separators are replaced as
// well, since some programs break a line at them. Bytes that are not valid
// UTF-8 also come out as U+FFFD.
func printable(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			return unicode.ReplacementChar
		}
		return r
	}, s)
}

// choose picks the History to print: HISTORY <number> when a number is given,
// otherwise the newest one that has tracks.
func choose(histories []pdb.History, number, root string) (pdb.History, error) {
	if number == "" {
		for i := len(histories) - 1; i >= 0; i-- {
			if len(histories[i].Tracks) > 0 {
				return histories[i], nil
			}
		}
		return pdb.History{}, fmt.Errorf("%s has no non-empty History", printable(root))
	}
	n, err := strconv.Atoi(number)
	if err != nil || n < 0 {
		return pdb.History{}, fmt.Errorf("--history needs a number, got %q", number)
	}
	for _, h := range histories {
		if m, ok := historyNumber(h.Name); ok && m == n {
			if len(h.Tracks) == 0 {
				return pdb.History{}, fmt.Errorf("%s is empty", h.Name)
			}
			return h, nil
		}
	}
	return pdb.History{}, fmt.Errorf("%s has no HISTORY %03d", printable(root), n)
}

// historyNumber returns the n of a History named "HISTORY n".
func historyNumber(name string) (int, bool) {
	digits, ok := strings.CutPrefix(name, "HISTORY ")
	if !ok || digits == "" {
		return 0, false
	}
	n, err := strconv.Atoi(digits)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// older orders Histories by the number in their name. A player names them
// "HISTORY 001" upward and records no date, so the highest number is the
// newest. Histories without such a name sort before the numbered ones, by id.
func older(a, b pdb.History) bool {
	na, oka := historyNumber(a.Name)
	nb, okb := historyNumber(b.Name)
	switch {
	case oka && okb:
		return na < nb
	case oka != okb:
		return okb
	default:
		return a.ID < b.ID
	}
}

// findStick returns the root of the stick to read. A stick is a volume with a
// PIONEER folder at its root.
func findStick(volume string) (string, error) {
	if volume != "" {
		if !isStick(volume) {
			return "", fmt.Errorf("%s has no PIONEER folder", printable(volume))
		}
		return volume, nil
	}
	var sticks []string
	for _, v := range listVolumes() {
		if isStick(v) {
			sticks = append(sticks, v)
		}
	}
	switch len(sticks) {
	case 0:
		return "", errors.New("no stick found: no mounted volume has a PIONEER folder")
	case 1:
		return sticks[0], nil
	default:
		for i, s := range sticks {
			sticks[i] = printable(s)
		}
		return "", fmt.Errorf("more than one stick is mounted; pick one with --volume:\n  %s",
			strings.Join(sticks, "\n  "))
	}
}

func isStick(root string) bool {
	info, err := os.Stat(filepath.Join(root, "PIONEER"))
	return err == nil && info.IsDir()
}

// volumeRoots lists the mount points of the running OS.
func volumeRoots() []string {
	var patterns []string
	switch runtime.GOOS {
	case "darwin":
		patterns = []string{"/Volumes/*"}
	case "windows":
		var roots []string
		for c := 'A'; c <= 'Z'; c++ {
			roots = append(roots, string(c)+`:\`)
		}
		return roots
	default:
		patterns = []string{"/media/*", "/media/*/*", "/run/media/*/*", "/mnt/*"}
	}
	var roots []string
	for _, p := range patterns {
		matches, _ := filepath.Glob(p)
		roots = append(roots, matches...)
	}
	return roots
}
