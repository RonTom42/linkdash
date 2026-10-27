package links

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const hostile = `</script><img src=x onerror=alert(1)>`

func TestSaveLoadRoundTrip(t *testing.T) {
	d := New()
	d.Title = `My "Links" & ` + hostile
	mustAdd(t, d, "Dev", "Go docs", "https://go.dev/doc")
	mustAdd(t, d, "Dev", hostile, "https://example.com/?a=1&b='2'")
	mustAdd(t, d, "Ärzte & <Praxis>", "Dr. Müller 🩺", "https://example.ch/müller")
	mustAdd(t, d, hostile, "line\u2028separator", "mailto:a@example.com")

	path := filepath.Join(t.TempDir(), "links.html")
	if err := Save(path, d); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want := &Dashboard{Version: currentVersion, Title: d.Title, Links: d.Sorted("")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip:\n got %+v\nwant %+v", got, want)
	}

	html, _ := os.ReadFile(path)
	if bytes.Contains(html, []byte("<img")) {
		t.Error("rendered page contains an unescaped <img tag")
	}
	if n := bytes.Count(html, []byte(dataOpen)); n != 1 {
		t.Errorf("page contains %d data blocks, want 1", n)
	}
	for _, s := range []string{`<a href="https://go.dev/doc" title="https://go.dev/doc" target="_blank" rel="noopener noreferrer">Go docs</a>`, "&lt;/script&gt;&lt;img"} {
		if !bytes.Contains(html, []byte(s)) {
			t.Errorf("page does not contain %q", s)
		}
	}
}

func TestRenderIsStable(t *testing.T) {
	d := New()
	mustAdd(t, d, "b", "two", "https://two.example")
	mustAdd(t, d, "a", "one", "https://one.example")
	first, err := Render(d)
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := Parse(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Render(reloaded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Error("re-rendering a loaded page changed it")
	}
}

func TestRenderEmpty(t *testing.T) {
	html, err := Render(New())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(html, []byte("No links yet")) || !bytes.Contains(html, []byte(`"links":[]`)) {
		t.Errorf("empty page lacks the placeholder or an empty links array:\n%s", html)
	}
}

func TestRenderNeutralizesUnsafeStoredURL(t *testing.T) {
	d := New()
	d.Links = append(d.Links, Link{Category: "x", Name: "evil", URL: "javascript:alert(1)"})
	html, err := Render(d)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(html, []byte(`<a href="#"`)) {
		t.Errorf("unsafe URL was not replaced by #:\n%s", html)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()

	d, err := Load(filepath.Join(dir, "missing.html"))
	if err != nil || len(d.Links) != 0 || d.Title != defaultTitle {
		t.Errorf("Load(missing) = %+v, %v; want empty dashboard", d, err)
	}

	foreign := filepath.Join(dir, "foreign.html")
	writeFile(t, foreign, "<html><body>hello</body></html>")
	if _, err := Load(foreign); !errors.Is(err, ErrNotDashboard) {
		t.Errorf("Load(foreign) error = %v, want ErrNotDashboard", err)
	}

	future := filepath.Join(dir, "future.html")
	writeFile(t, future, dataOpen+`{"version":99}`+dataClose)
	if _, err := Load(future); err == nil || !strings.Contains(err.Error(), "newer") {
		t.Errorf("Load(future) error = %v, want newer-version error", err)
	}
}

func TestSaveKeepsSymlinkAndMode(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.html")
	link := filepath.Join(dir, "link.html")
	if err := Save(target, New()); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks unsupported:", err)
	}

	d := New()
	mustAdd(t, d, "a", "one", "https://one.example")
	if err := Save(link, d); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("link.html is no longer a symlink (%v)", err)
	}
	if fi, _ := os.Stat(target); fi.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", fi.Mode().Perm())
	}
	if got, _ := Load(target); len(got.Links) != 1 {
		t.Errorf("target has %d links, want 1", len(got.Links))
	}
	if leftovers, _ := filepath.Glob(filepath.Join(dir, ".linkdash-*")); len(leftovers) > 0 {
		t.Errorf("temporary files left behind: %v", leftovers)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
