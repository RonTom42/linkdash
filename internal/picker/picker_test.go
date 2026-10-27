package picker

import (
	"reflect"
	"strings"
	"testing"
)

var items = []string{
	"[Dev]   Go docs       https://go.dev/doc",
	"[Dev]   godoc.org     https://pkg.go.dev",
	"[Misc]  Google Docs   https://docs.google.com",
	"[News]  Zeit          https://zeit.de",
}

func TestFilter(t *testing.T) {
	tests := []struct {
		query string
		want  []int
	}{
		{"", []int{0, 1, 2, 3}},
		{"go", []int{0, 1, 2}},
		{"GO DOC", []int{0, 1, 2}},
		{"docs dev", []int{0}},
		{"zeit.de", []int{3}},
		{"nothing", []int{}},
	}
	for _, tt := range tests {
		if got := Filter(items, tt.query); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Filter(%q) = %v, want %v", tt.query, got, tt.want)
		}
	}
}

func TestDecode(t *testing.T) {
	events, rest := decode([]byte("aé\x1b[A\x1bOB\x7f\x15\r\x1b[5~\x10\x0e\x1b\x03\xc3"))
	want := []event{
		{kind: keyRune, r: 'a'},
		{kind: keyRune, r: 'é'},
		{kind: keyUp},
		{kind: keyDown},
		{kind: keyBackspace},
		{kind: keyClear},
		{kind: keyEnter},
		// \x1b[5~ (Page Up) is ignored
		{kind: keyUp},
		{kind: keyDown},
		{kind: keyCancel},
		{kind: keyCancel},
	}
	if !reflect.DeepEqual(events, want) {
		t.Errorf("events = %v\nwant     %v", events, want)
	}
	if string(rest) != "\xc3" {
		t.Errorf("rest = %q, want the incomplete UTF-8 byte", rest)
	}
	if events, _ := decode(append(rest, 0xbc)); !reflect.DeepEqual(events, []event{{kind: keyRune, r: 'ü'}}) {
		t.Errorf("completed rune decoded as %v", events)
	}
}

func TestUpdate(t *testing.T) {
	type step struct {
		keys       string
		wantQuery  string
		wantChoice int // index into items under the cursor, -1 for no matches
		wantDone   bool
		wantPicked bool
	}
	tests := []struct {
		name  string
		query string
		steps []step
	}{
		{"type and pick", "", []step{
			{keys: "go", wantQuery: "go", wantChoice: 0},
			{keys: "\x1b[B\x1b[B\x1b[B", wantQuery: "go", wantChoice: 2},
			{keys: "\x1b[A", wantQuery: "go", wantChoice: 1},
			{keys: "\r", wantQuery: "go", wantChoice: 1, wantDone: true, wantPicked: true},
		}},
		{"query resets cursor", "", []step{
			{keys: "\x0e\x0e", wantChoice: 2},
			{keys: "z", wantQuery: "z", wantChoice: 3},
		}},
		{"prefilled query and backspace", "zeitx", []step{
			{keys: "", wantQuery: "zeitx", wantChoice: -1},
			{keys: "\r", wantQuery: "zeitx", wantChoice: -1}, // Enter with no match does nothing
			{keys: "\x7f", wantQuery: "zeit", wantChoice: 3},
			{keys: "\x15", wantQuery: "", wantChoice: 0},
			{keys: "\x7f\x10", wantQuery: "", wantChoice: 0},
		}},
		{"escape cancels", "", []step{
			{keys: "\x1b", wantChoice: 0, wantDone: true},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newState(items, tt.query, false)
			for _, st := range tt.steps {
				events, _ := decode([]byte(st.keys))
				var done, picked bool
				for _, ev := range events {
					if done, picked = s.update(ev); done {
						break
					}
				}
				choice := -1
				if len(s.matches) > 0 {
					choice = s.matches[s.cursor]
				}
				if string(s.query) != st.wantQuery || choice != st.wantChoice || done != st.wantDone || picked != st.wantPicked {
					t.Fatalf("after %q: query %q, choice %d, done %v, picked %v; want %q, %d, %v, %v",
						st.keys, string(s.query), choice, done, picked, st.wantQuery, st.wantChoice, st.wantDone, st.wantPicked)
				}
			}
		})
	}
}

func TestRenderScrollsToCursor(t *testing.T) {
	s := newState(items, "", false)
	s.cursor = 3
	out := s.render("Remove a link", 30, 6) // room for 2 rows
	if !strings.Contains(out, "▶ [News]") || strings.Contains(out, "[Dev]   Go docs") {
		t.Errorf("window does not follow the cursor:\n%q", out)
	}
	for _, line := range strings.Split(out, "\r\n") {
		if strings.Contains(line, "https://") && !strings.Contains(line, "…") {
			t.Errorf("line not truncated to width: %q", line)
		}
	}
}

func TestAllowNew(t *testing.T) {
	categories := []string{"Dev", "News", "Reading"}
	tests := []struct {
		query string
		want  []int
	}{
		{"", []int{0, 1, 2}},             // nothing typed, nothing to create
		{"e", []int{0, 1, 2, newEntry}},  // partial matches come first
		{"  dev ", []int{0}},             // exact match (any case) offers no new entry
		{"Tools", []int{newEntry}},       // only the new entry
		{"read ing", []int{2, newEntry}}, // terms match, but the text is not an item
	}
	for _, tt := range tests {
		if got := newState(categories, tt.query, true).matches; !reflect.DeepEqual(got, tt.want) {
			t.Errorf("query %q: matches = %v, want %v", tt.query, got, tt.want)
		}
	}

	s := newState(categories, "", true)
	for _, ev := range []event{{kind: keyRune, r: 'M'}, {kind: keyRune, r: 'y'}, {kind: keyRune, r: ' '}, {kind: keyRune, r: ' '}, {kind: keyRune, r: 'X'}} {
		s.update(ev)
	}
	if done, picked := s.update(event{kind: keyEnter}); !done || !picked || s.matches[s.cursor] != newEntry || s.text() != "My X" {
		t.Errorf("Enter on new entry: done %v, picked %v, choice %d, text %q", done, picked, s.matches[s.cursor], s.text())
	}
	if out := s.render("Category", 40, 10); !strings.Contains(out, "▶ + New: My X") || !strings.Contains(out, "0/3") {
		t.Errorf("render does not show the selected new entry:\n%q", out)
	}
}
