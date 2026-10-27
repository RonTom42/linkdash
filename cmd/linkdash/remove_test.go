package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/RonTom42/linkdash/internal/picker"
)

func TestRemoveByArgument(t *testing.T) {
	h := newHarness(t)
	h.seed()

	out := h.mustRun("remove", "https://zeit.de")
	if !strings.Contains(out, `Removed "Zeit"`) || !strings.Contains(out, "Undo: linkdash -f "+shellQuote(h.file)+" add -c News Zeit https://zeit.de") {
		t.Errorf("remove output = %q", out)
	}

	err := h.run("remove", "Go docs")
	if err == nil || !strings.Contains(err.Error(), "matches 2 links") {
		t.Errorf("ambiguous remove without a terminal: err = %v", err)
	}
	h.mustRun("rm", "-c", "reading", "go", "docs")
	h.mustRun("remove", "go.dev/doc")

	if err := h.run("remove", "nope"); err == nil || !strings.Contains(err.Error(), "no link has") {
		t.Errorf("remove of unknown link: err = %v", err)
	}
	if err := h.run("remove"); err == nil || !strings.Contains(err.Error(), "not a terminal") {
		t.Errorf("remove without arguments or terminal: err = %v", err)
	}
	if got := h.names(); len(got) != 1 || got[0] != "Dev/GitHub" {
		t.Errorf("remaining links = %v", got)
	}
}

func TestRemoveWithPicker(t *testing.T) {
	h := newHarness(t)
	h.seed()

	var gotItems []string
	var gotQuery string
	h.pick = func(items []string, opt picker.Options) (int, string, error) {
		gotItems, gotQuery = items, opt.Query
		for i, item := range items {
			if strings.Contains(item, "[Reading]") {
				return i, opt.Query, nil
			}
		}
		return -1, "", errors.New("no Reading item")
	}
	h.mustRun("remove", "Go docs")
	if gotQuery != "Go docs" || len(gotItems) != 4 || !strings.HasPrefix(gotItems[0], "[Dev]      GitHub   https://github.com") {
		t.Errorf("picker got query %q, items %q", gotQuery, gotItems)
	}

	// -c narrows the list; no query opens the picker too.
	h.pick = func(items []string, opt picker.Options) (int, string, error) {
		gotItems, gotQuery = items, opt.Query
		return 0, "", nil
	}
	h.mustRun("remove", "-c", "News")
	if gotQuery != "" || len(gotItems) != 1 {
		t.Errorf("picker got query %q, items %q", gotQuery, gotItems)
	}

	h.pick = func([]string, picker.Options) (int, string, error) { return -1, "", picker.ErrCancelled }
	if out := h.mustRun("remove"); out != "" || !strings.Contains(h.stderr.String(), "Nothing removed") {
		t.Errorf("cancelled picker: stdout %q, stderr %q", out, h.stderr.String())
	}

	if got := strings.Join(h.names(), ","); got != "Dev/Go docs,Dev/GitHub" && got != "Dev/GitHub,Dev/Go docs" {
		t.Errorf("remaining links = %v", got)
	}
}
