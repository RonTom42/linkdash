package links

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
)

// The data block is located by these exact strings, so changing them breaks
// every existing page. Its content never contains dataClose: html/template
// marshals the JSON with <, > and & escaped as <, > and &.
const (
	dataOpen  = `<script type="application/json" id="linkdash-data">`
	dataClose = `</script>`
)

// ErrNotDashboard means a file exists but was not written by linkdash.
var ErrNotDashboard = errors.New("not a linkdash file")

//go:embed template.html
var templateHTML string

var page = template.Must(template.New("page").Funcs(template.FuncMap{
	"href": func(u string) template.URL {
		if !safeURL(u) {
			return "#"
		}
		return template.URL(u)
	},
}).Parse(templateHTML))

// Load reads the dashboard embedded in the HTML file at path. A missing file
// yields an empty dashboard.
func Load(path string) (*Dashboard, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return New(), nil
	}
	if err != nil {
		return nil, err
	}
	d, err := Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return d, nil
}

// Parse extracts the dashboard from a rendered page.
func Parse(html []byte) (*Dashboard, error) {
	_, rest, ok := bytes.Cut(html, []byte(dataOpen))
	if !ok {
		return nil, ErrNotDashboard
	}
	data, _, ok := bytes.Cut(rest, []byte(dataClose))
	if !ok {
		return nil, ErrNotDashboard
	}
	d := New()
	if err := json.Unmarshal(data, d); err != nil {
		return nil, fmt.Errorf("%w: invalid embedded data: %v", ErrNotDashboard, err)
	}
	if d.Version > currentVersion {
		return nil, fmt.Errorf("written by a newer linkdash (data version %d, this build supports %d)", d.Version, currentVersion)
	}
	if d.Links == nil {
		d.Links = []Link{}
	}
	return d, nil
}

// Render returns the complete HTML page for d. Output is deterministic.
func Render(d *Dashboard) ([]byte, error) {
	sorted := d.Sorted("")
	var buf bytes.Buffer
	err := page.Execute(&buf, struct {
		Title      string
		Categories []Category
		Data       Dashboard
	}{
		Title:      d.Title,
		Categories: Group(sorted),
		Data:       Dashboard{Version: currentVersion, Title: d.Title, Links: sorted},
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Save renders d and atomically replaces the file at path. If path is a
// symlink, its target is replaced and the link is kept.
func Save(path string, d *Dashboard) error {
	b, err := Render(d)
	if err != nil {
		return err
	}
	if target, err := filepath.EvalSymlinks(path); err == nil {
		path = target
	}
	return writeAtomic(path, b)
}

func writeAtomic(path string, data []byte) error {
	mode := fs.FileMode(0o644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".linkdash-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name()) // fails harmlessly once renamed
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
