package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/RonTom42/linkdash/internal/links"
)

func TestRenderAndTitle(t *testing.T) {
	h := newHarness(t)
	h.seed()
	h.mustRun("title", "My", "Start", "Page")
	before, _ := os.ReadFile(h.file)
	if !bytes.Contains(before, []byte("<title>My Start Page</title>")) {
		t.Error("title was not rendered")
	}
	h.mustRun("render")
	after, _ := os.ReadFile(h.file)
	if !bytes.Equal(before, after) {
		t.Error("render changed an up-to-date page")
	}

	foreign := filepath.Join(t.TempDir(), "index.html")
	if err := os.WriteFile(foreign, []byte("<html>mine</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := h.runRaw("-f", foreign, "render"); !errors.Is(err, links.ErrNotDashboard) {
		t.Errorf("render of a foreign file: err = %v", err)
	}
	if b, _ := os.ReadFile(foreign); string(b) != "<html>mine</html>" {
		t.Error("foreign file was overwritten")
	}
}
