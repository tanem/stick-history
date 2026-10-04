package doclinks_test

import (
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/tanem/stick-history/internal/doclinks"
)

// check fails the test unless Broken reports exactly the links in want.
func check(t *testing.T, fsys fstest.MapFS, want ...doclinks.Link) {
	t.Helper()
	got, err := doclinks.Broken(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Broken() = %v, want %v", got, want)
	}
}

func file(text string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(text)}
}

func TestMissingFile(t *testing.T) {
	fsys := fstest.MapFS{
		"README.md": file("# Title\n\nSee [the guide](docs/guide.md).\n"),
	}
	check(t, fsys, doclinks.Link{File: "README.md", Line: 3, Target: "docs/guide.md"})
}

func TestFileOrDirectory(t *testing.T) {
	fsys := fstest.MapFS{
		"README.md":     file("[guide](docs/guide.md) and [docs](docs) and [docs](docs/)\n"),
		"docs/guide.md": file("# Guide\n"),
	}
	check(t, fsys)
}

func TestRelativeToFile(t *testing.T) {
	fsys := fstest.MapFS{
		"README.md":     file("# Title\n"),
		"docs/guide.md": file("[up](../README.md), [beside](other.md), [wrong](README.md)\n"),
		"docs/other.md": file("# Other\n"),
	}
	check(t, fsys, doclinks.Link{File: "docs/guide.md", Line: 1, Target: "README.md"})
}

func TestFragment(t *testing.T) {
	fsys := fstest.MapFS{
		"README.md":     file("[a](docs/guide.md#install)\n[b](#usage)\n[c](docs/gone.md#install)\n"),
		"docs/guide.md": file("# Guide\n"),
	}
	check(t, fsys, doclinks.Link{File: "README.md", Line: 3, Target: "docs/gone.md#install"})
}

func TestExternalURLs(t *testing.T) {
	fsys := fstest.MapFS{
		"README.md": file("[a](https://example.com/gone.md)\n[b](mailto:someone@example.com)\n[c](//example.com/gone.md)\n"),
	}
	check(t, fsys)
}

func TestReferenceDefinitions(t *testing.T) {
	fsys := fstest.MapFS{
		"README.md":     file("See [the guide][guide] and [the other][other].\n\n[guide]: docs/guide.md\n[other]: docs/gone.md\n"),
		"docs/guide.md": file("# Guide\n"),
	}
	check(t, fsys, doclinks.Link{File: "README.md", Line: 4, Target: "docs/gone.md"})
}

func TestLeadingSlash(t *testing.T) {
	fsys := fstest.MapFS{
		"README.md":     file("# Title\n"),
		"docs/guide.md": file("[a](/README.md)\n[b](/guide.md)\n"),
	}
	check(t, fsys, doclinks.Link{File: "docs/guide.md", Line: 2, Target: "/guide.md"})
}

func TestGitDirectory(t *testing.T) {
	fsys := fstest.MapFS{
		".git/notes.md": file("[a](gone.md)\n"),
	}
	check(t, fsys)
}
