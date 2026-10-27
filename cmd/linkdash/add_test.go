package main

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/RonTom42/linkdash/internal/picker"
)

func TestAddAndList(t *testing.T) {
	h := newHarness(t)
	h.seed()
	if got := h.mustRun("add", "-c", "dev", "The Go docs", "https://go.dev/doc"); !strings.HasPrefix(got, "Renamed") {
		t.Errorf("re-adding a URL printed %q, want a rename", got)
	}

	want := "Dev\n  GitHub       https://github.com\n  The Go docs  https://go.dev/doc\n" +
		"News\n  Zeit  https://zeit.de\n" +
		"Reading\n  Go docs  https://go.dev/doc\n"
	if got := h.mustRun("list"); got != want {
		t.Errorf("list =\n%s\nwant\n%s", got, want)
	}
	if got := h.mustRun("ls", "-c", "news"); got != "News\n  Zeit  https://zeit.de\n" {
		t.Errorf("list -c news = %q", got)
	}
}

func TestAddUsageErrors(t *testing.T) {
	h := newHarness(t)
	for _, args := range [][]string{
		{"add", "Go docs", "https://go.dev"},
		{"add", "-c", "Dev", "https://go.dev"},
		{"add", "-x"},
		{"bogus"},
	} {
		if err := h.run(args...); !errors.Is(err, errUsage) {
			t.Errorf("linkdash %q: err = %v, want usage error", args, err)
		}
	}
	if err := h.run("add", "-c", "Dev", "x", "javascript:alert(1)"); err == nil || errors.Is(err, errUsage) {
		t.Errorf("javascript: URL: err = %v, want a plain error", err)
	}
	if _, err := os.Stat(h.file); !os.IsNotExist(err) {
		t.Error("failed commands created the file")
	}
}

func TestAddInteractive(t *testing.T) {
	h := newHarness(t)
	h.seed()

	// No arguments: pick an existing category, then answer the prompts. An
	// invalid URL is asked for again.
	var got picker.Options
	var gotItems []string
	h.pick = func(items []string, opt picker.Options) (int, string, error) {
		got, gotItems = opt, items
		return 1, "", nil
	}
	h.stdin = "  The Zeit \njavascript:alert(1)\nzeit.de/x\n"
	if out := h.mustRun("add"); out != "Added \"The Zeit\" (https://zeit.de/x) to News.\n" {
		t.Errorf("stdout = %q", out)
	}
	if !got.AllowNew || strings.Join(gotItems, ",") != "Dev,News,Reading" {
		t.Errorf("picker got %+v with items %q", got, gotItems)
	}
	for _, want := range []string{"Category: News\n", "Name: URL: ", `scheme "javascript" is not allowed`} {
		if !strings.Contains(h.stderr.String(), want) {
			t.Errorf("stderr %q lacks %q", h.stderr.String(), want)
		}
	}

	// A new category typed in the picker; name and URL from arguments.
	h.pick = func([]string, picker.Options) (int, string, error) { return -1, "Tools", nil }
	h.stdin = ""
	h.mustRun("add", "jq", "https://jqlang.org")
	if !strings.Contains(h.stderr.String(), "Category: Tools (new)") {
		t.Errorf("stderr = %q", h.stderr.String())
	}

	// -c skips the picker; only the prompts are asked.
	h.pick = func([]string, picker.Options) (int, string, error) {
		t.Error("picker opened although -c was given")
		return -1, "", picker.ErrCancelled
	}
	h.stdin = "\nRust\nrust-lang.org"
	h.mustRun("add", "-c", "dev")
	if !strings.Contains(h.stderr.String(), "the name cannot be empty") {
		t.Errorf("empty name was not rejected: %q", h.stderr.String())
	}

	// End of input or Esc adds nothing.
	h.stdin = "Half"
	if out := h.mustRun("add", "-c", "dev"); out != "" || !strings.Contains(h.stderr.String(), "Nothing added.") {
		t.Errorf("EOF at URL prompt: stdout %q, stderr %q", out, h.stderr.String())
	}
	h.pick = func([]string, picker.Options) (int, string, error) { return -1, "", picker.ErrCancelled }
	if out := h.mustRun("add"); out != "" || !strings.Contains(h.stderr.String(), "Nothing added.") {
		t.Errorf("cancelled picker: stdout %q, stderr %q", out, h.stderr.String())
	}

	if err := h.run("add", "only-one-arg"); !errors.Is(err, errUsage) {
		t.Errorf("one positional argument: err = %v, want usage error", err)
	}

	want := "Dev/GitHub,Dev/Go docs,Dev/Rust,News/The Zeit,News/Zeit,Reading/Go docs,Tools/jq"
	if got := strings.Join(h.names(), ","); got != want {
		t.Errorf("links = %s\nwant    %s", got, want)
	}
}
