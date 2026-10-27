package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/RonTom42/linkdash/internal/links"
	"github.com/RonTom42/linkdash/internal/picker"
)

type harness struct {
	t              *testing.T
	file           string
	vars           map[string]string
	stdin          string
	stdout, stderr bytes.Buffer
	pick           func(items []string, opt picker.Options) (int, string, error)
}

func newHarness(t *testing.T) *harness {
	return &harness{t: t, file: filepath.Join(t.TempDir(), "page.html"), vars: map[string]string{}}
}

// run executes linkdash with -f pointing at the harness file.
func (h *harness) run(args ...string) error {
	return h.runRaw(append([]string{"-f", h.file}, args...)...)
}

func (h *harness) runRaw(args ...string) error {
	h.stdout.Reset()
	h.stderr.Reset()
	return run(args, env{
		stdin:  strings.NewReader(h.stdin),
		stdout: &h.stdout,
		stderr: &h.stderr,
		getenv: func(k string) string { return h.vars[k] },
		pick:   h.pick,
	})
}

func (h *harness) mustRun(args ...string) string {
	h.t.Helper()
	if err := h.run(args...); err != nil {
		h.t.Fatalf("linkdash %q: %v\nstderr: %s", args, err, h.stderr.String())
	}
	return h.stdout.String()
}

func (h *harness) names() []string {
	h.t.Helper()
	d, err := links.Load(h.file)
	if err != nil {
		h.t.Fatal(err)
	}
	var out []string
	for _, l := range d.Sorted("") {
		out = append(out, l.Category+"/"+l.Name)
	}
	return out
}

func (h *harness) seed() {
	h.t.Helper()
	h.mustRun("add", "-c", "Dev", "Go docs", "go.dev/doc")
	h.mustRun("add", "GitHub", "https://github.com", "-c", "dev") // flags after positionals
	h.mustRun("add", "--category=Reading", "Go docs", "https://go.dev/doc")
	h.mustRun("add", "-c", "News", "Zeit", "https://zeit.de")
}

func TestFileResolution(t *testing.T) {
	h := newHarness(t)
	envFile := filepath.Join(t.TempDir(), "env.html")
	h.vars[fileEnv] = envFile

	if err := h.runRaw("add", "-c", "A", "one", "one.example"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(envFile); err != nil {
		t.Errorf("$%s was not used: %v", fileEnv, err)
	}

	// -f wins over the environment, before or after the command.
	if err := h.runRaw("add", "-c", "A", "two", "two.example", "-f", h.file); err != nil {
		t.Fatal(err)
	}
	if got := h.names(); len(got) != 1 || got[0] != "A/two" {
		t.Errorf("-f file has %v", got)
	}
}

func TestVersion(t *testing.T) {
	h := newHarness(t)
	for _, args := range [][]string{{"version"}, {"--version"}, {"-version", "add"}} {
		if err := h.runRaw(args...); err != nil {
			t.Fatalf("linkdash %q: %v", args, err)
		}
		if got := h.stdout.String(); got != "linkdash (devel)\n" {
			t.Errorf("linkdash %q printed %q", args, got)
		}
	}
	version = "v1.2.3"
	t.Cleanup(func() { version = "" })
	if err := h.runRaw("version"); err != nil || h.stdout.String() != "linkdash v1.2.3\n" {
		t.Errorf("with -X main.version: %q, %v", h.stdout.String(), err)
	}
}
