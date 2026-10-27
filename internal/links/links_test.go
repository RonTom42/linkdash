package links

import (
	"reflect"
	"strings"
	"testing"
)

func mustAdd(t *testing.T, d *Dashboard, category, name, url string) Link {
	t.Helper()
	l, _, err := d.Add(category, name, url)
	if err != nil {
		t.Fatalf("Add(%q, %q, %q): %v", category, name, url, err)
	}
	return l
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		in, want string
		wantErr  bool
	}{
		{in: "https://go.dev/doc", want: "https://go.dev/doc"},
		{in: "  go.dev/doc ", want: "https://go.dev/doc"},
		{in: "localhost:8080/x", want: "https://localhost:8080/x"},
		{in: "http://localhost:8080", want: "http://localhost:8080"},
		{in: "mailto:me@example.com", want: "mailto:me@example.com"},
		{in: "obsidian://open?vault=notes", want: "obsidian://open?vault=notes"},
		{in: "file:///Users/me/notes.txt", want: "file:///Users/me/notes.txt"},
		{in: "", wantErr: true},
		{in: "javascript:alert(1)", wantErr: true},
		{in: "JavaScript:alert(1)", wantErr: true},
		{in: "data:text/html,<script>alert(1)</script>", wantErr: true},
		{in: "vbscript:msgbox", wantErr: true},
		{in: "java\tscript:alert(1)", wantErr: true},
		{in: "https://", wantErr: true},
	}
	for _, tt := range tests {
		got, err := NormalizeURL(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("NormalizeURL(%q) = %q, want error", tt.in, got)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("NormalizeURL(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
		}
	}
}

func TestAdd(t *testing.T) {
	d := New()
	mustAdd(t, d, "Dev", "Go docs", "https://go.dev/doc")

	l := mustAdd(t, d, " dev ", "Go  blog", "go.dev/blog")
	if l.Category != "Dev" {
		t.Errorf("category = %q, want existing spelling %q", l.Category, "Dev")
	}
	if l.Name != "Go blog" {
		t.Errorf("name = %q, want collapsed whitespace", l.Name)
	}

	l, updated, err := d.Add("DEV", "The Go docs", "https://go.dev/doc")
	if err != nil || !updated || l.Name != "The Go docs" {
		t.Errorf("re-adding URL: link %+v, updated %v, err %v; want rename", l, updated, err)
	}
	if len(d.Links) != 2 {
		t.Errorf("got %d links, want 2", len(d.Links))
	}

	// The same URL may appear in another category.
	if _, updated, _ := d.Add("Reading", "Go docs", "https://go.dev/doc"); updated || len(d.Links) != 3 {
		t.Errorf("same URL in another category: updated %v, %d links; want a new link", updated, len(d.Links))
	}

	for _, args := range [][3]string{{"", "x", "go.dev"}, {"Dev", " ", "go.dev"}, {"Dev", "x", "javascript:x"}} {
		if _, _, err := d.Add(args[0], args[1], args[2]); err == nil {
			t.Errorf("Add%q succeeded, want error", args)
		}
	}
}

func TestSortedAndGroup(t *testing.T) {
	d := New()
	mustAdd(t, d, "news", "zeit", "https://zeit.de")
	mustAdd(t, d, "Dev", "github", "https://github.com")
	mustAdd(t, d, "news", "BBC", "https://bbc.com")
	mustAdd(t, d, "Dev", "Go docs", "https://go.dev/doc")
	mustAdd(t, d, "dev", "atlassian", "https://atlassian.com")
	mustAdd(t, d, "Apps", "Mail", "https://mail.example")

	var got []string
	for _, c := range Group(d.Sorted("")) {
		names := []string{}
		for _, l := range c.Links {
			names = append(names, l.Name)
		}
		got = append(got, c.Name+": "+strings.Join(names, ", "))
	}
	want := []string{"Apps: Mail", "Dev: atlassian, github, Go docs", "news: BBC, zeit"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("groups =\n%q\nwant\n%q", got, want)
	}

	if n := len(d.Sorted("DEV")); n != 3 {
		t.Errorf("Sorted(DEV) returned %d links, want 3", n)
	}
}

func TestMatchAndRemove(t *testing.T) {
	d := New()
	mustAdd(t, d, "Dev", "Go docs", "https://go.dev/doc")
	mustAdd(t, d, "Reading", "Go docs", "https://go.dev/doc")
	mustAdd(t, d, "Dev", "GitHub", "https://github.com")

	tests := []struct {
		query, category string
		want            int
	}{
		{"https://go.dev/doc", "", 2},
		{"go.dev/doc", "", 2},
		{"go docs", "", 2},
		{"Go docs", "reading", 1},
		{"github", "", 1},
		{"https://github.com/", "", 0},
		{"Go", "", 0},
	}
	for _, tt := range tests {
		if got := d.Match(tt.query, tt.category); len(got) != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %d links", tt.query, tt.category, got, tt.want)
		}
	}

	if !d.Remove(d.Match("Go docs", "Reading")[0]) {
		t.Fatal("Remove returned false")
	}
	if d.Remove(Link{Category: "Reading", URL: "https://go.dev/doc"}) {
		t.Error("removing twice returned true")
	}
	for _, c := range Group(d.Sorted("")) {
		if c.Name == "Reading" {
			t.Error("empty category Reading is still listed")
		}
	}
}
