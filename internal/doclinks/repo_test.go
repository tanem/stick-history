package doclinks_test

import (
	"os"
	"testing"

	"github.com/tanem/stick-history/internal/doclinks"
)

// TestRepoLinks checks the Markdown files of this repo. The test runs in
// internal/doclinks, so the root is two directories up.
func TestRepoLinks(t *testing.T) {
	broken, err := doclinks.Broken(os.DirFS("../.."))
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range broken {
		t.Errorf("%s:%d: link target %s does not exist", l.File, l.Line, l.Target)
	}
}
