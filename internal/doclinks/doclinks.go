// Package doclinks finds the relative links in Markdown files that point at a
// file or directory that does not exist.
package doclinks

import (
	"io/fs"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// Link is a link in a Markdown file.
type Link struct {
	// File is the path of the Markdown file, relative to the root.
	File string
	// Line is the line the link is on, counted from 1.
	Line int
	// Target is the link's destination as the file writes it.
	Target string
}

// linkTarget matches the destination of an inline link, [text](destination),
// and of a reference definition, [label]: destination.
var linkTarget = regexp.MustCompile(`\]\(([^)\s]+)|^\s*\[[^\]]+\]:\s+(\S+)`)

// Broken returns the relative links in the *.md files of fsys whose target
// does not exist.
func Broken(fsys fs.FS) ([]Link, error) {
	var broken []Link
	err := fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if path.Ext(name) != ".md" {
			return nil
		}
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(data), "\n") {
			for _, m := range linkTarget.FindAllStringSubmatch(line, -1) {
				// One of the two groups is empty: m[1] is an inline link's
				// destination and m[2] is a reference definition's.
				target := m[1] + m[2]
				if isBroken(fsys, name, target) {
					broken = append(broken, Link{File: name, Line: i + 1, Target: target})
				}
			}
		}
		return nil
	})
	return broken, err
}

// isBroken reports whether target, a link in the file name, is a relative link
// to a file or directory that is not in fsys. An external URL and a link that
// is only a fragment are not checked, so they are not broken.
func isBroken(fsys fs.FS, name, target string) bool {
	u, err := url.Parse(target)
	if err != nil {
		return true
	}
	if u.Scheme != "" || u.Host != "" || u.Path == "" {
		return false
	}
	// A leading slash starts at the root, as it does on GitHub.
	resolved := path.Join(".", u.Path)
	if !strings.HasPrefix(u.Path, "/") {
		resolved = path.Join(path.Dir(name), u.Path)
	}
	// A target that leaves the root is not a valid path in fsys, so Stat
	// fails and the link is reported.
	_, err = fs.Stat(fsys, resolved)
	return err != nil
}
