// Package picker is a minimal fzf-like terminal list: type to filter, move
// with the arrow keys, Enter to choose. Key handling and drawing are pure
// (state.update, state.render); only Pick touches the terminal.
package picker

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/term"
)

// ErrCancelled is returned when the user leaves the picker without choosing.
var ErrCancelled = errors.New("cancelled")

// Options configure Pick.
type Options struct {
	Title string
	Query string // initial search text
	// AllowNew offers the search text as an extra entry when it is not
	// exactly one of the items. Choosing that entry returns index -1.
	AllowNew bool
}

// Pick shows items on the terminal and returns the index of the chosen item
// (-1 for the AllowNew entry) and the final search text, whitespace-collapsed.
func Pick(items []string, opt Options) (index int, text string, err error) {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return -1, "", fmt.Errorf("opening terminal: %w", err)
	}
	defer tty.Close()
	fd := int(tty.Fd())
	saved, err := term.MakeRaw(fd)
	if err != nil {
		return -1, "", err
	}
	defer func() {
		_, leaveErr := io.WriteString(tty, "\x1b[?1049l") // main screen
		if restoreErr := term.Restore(fd, saved); err == nil {
			err = errors.Join(leaveErr, restoreErr)
		}
	}()
	if _, err := io.WriteString(tty, "\x1b[?1049h"); err != nil { // alternate screen
		return -1, "", err
	}

	s := newState(items, opt.Query, opt.AllowNew)
	var pending []byte
	buf := make([]byte, 1024)
	for {
		width, height, err := term.GetSize(fd)
		if err != nil || width <= 0 || height <= 0 {
			width, height = 80, 24
		}
		if _, err := io.WriteString(tty, s.render(opt.Title, width, height)); err != nil {
			return -1, "", err
		}

		n, err := tty.Read(buf)
		if err != nil {
			return -1, "", err
		}
		events, rest := decode(append(pending, buf[:n]...))
		pending = append([]byte(nil), rest...)
		for _, ev := range events {
			if done, picked := s.update(ev); done {
				if !picked {
					return -1, "", ErrCancelled
				}
				return s.matches[s.cursor], s.text(), nil
			}
		}
	}
}

// Filter returns the indices of items containing every whitespace-separated
// term of query, case-insensitively, in their original order.
func Filter(items []string, query string) []int {
	terms := strings.Fields(strings.ToLower(query))
	out := []int{}
	for i, item := range items {
		item = strings.ToLower(item)
		if allContained(item, terms) {
			out = append(out, i)
		}
	}
	return out
}

func allContained(s string, terms []string) bool {
	for _, t := range terms {
		if !strings.Contains(s, t) {
			return false
		}
	}
	return true
}

type keyKind int

const (
	keyRune keyKind = iota
	keyBackspace
	keyClear
	keyUp
	keyDown
	keyEnter
	keyCancel
)

type event struct {
	kind keyKind
	r    rune
}

// decode turns raw terminal input into events. It returns any trailing bytes
// of an incomplete UTF-8 sequence so the caller can prepend them to the next
// read. Unknown control characters and escape sequences are dropped.
func decode(b []byte) (events []event, rest []byte) {
	for len(b) > 0 {
		c := b[0]
		switch {
		case c == 0x1b:
			n, ev, ok := decodeEscape(b)
			if ok {
				events = append(events, ev)
			}
			b = b[n:]
			continue
		case c == '\r' || c == '\n':
			events = append(events, event{kind: keyEnter})
		case c == 0x7f || c == 0x08:
			events = append(events, event{kind: keyBackspace})
		case c == 0x15: // Ctrl-U
			events = append(events, event{kind: keyClear})
		case c == 0x10: // Ctrl-P
			events = append(events, event{kind: keyUp})
		case c == 0x0e: // Ctrl-N
			events = append(events, event{kind: keyDown})
		case c == 0x03 || c == 0x04: // Ctrl-C, Ctrl-D
			events = append(events, event{kind: keyCancel})
		case c < 0x20:
		default:
			if !utf8.FullRune(b) {
				return events, b
			}
			r, size := utf8.DecodeRune(b)
			if r != utf8.RuneError && unicode.IsPrint(r) {
				events = append(events, event{kind: keyRune, r: r})
			}
			b = b[size:]
			continue
		}
		b = b[1:]
	}
	return events, nil
}

// decodeEscape decodes an escape sequence at the start of b, which begins
// with ESC. A lone ESC cancels; ↑/↓ arrive as CSI or SS3 sequences.
func decodeEscape(b []byte) (n int, ev event, ok bool) {
	if len(b) == 1 || (b[1] != '[' && b[1] != 'O') {
		if len(b) > 1 && b[1] >= 0x20 && b[1] != 0x7f {
			return 2, event{}, false // Alt+printable key
		}
		return 1, event{kind: keyCancel}, true
	}
	// Skip parameter bytes up to the final byte (0x40–0x7e).
	i := 2
	for i < len(b) && (b[i] < 0x40 || b[i] > 0x7e) {
		i++
	}
	if i == len(b) {
		return len(b), event{}, false
	}
	switch b[i] {
	case 'A':
		return i + 1, event{kind: keyUp}, true
	case 'B':
		return i + 1, event{kind: keyDown}, true
	}
	return i + 1, event{}, false
}

// newEntry stands in matches for the typed text offered as a new item.
const newEntry = -1

type state struct {
	items    []string
	allowNew bool
	query    []rune
	matches  []int // indices into items, or newEntry (always last)
	cursor   int   // index into matches
	offset   int   // first match shown
}

func newState(items []string, query string, allowNew bool) *state {
	s := &state{items: items, allowNew: allowNew, query: []rune(query)}
	s.refilter()
	return s
}

func (s *state) refilter() {
	s.matches = Filter(s.items, string(s.query))
	if text := s.text(); s.allowNew && text != "" && !s.isItem(text) {
		s.matches = append(s.matches, newEntry)
	}
	s.cursor, s.offset = 0, 0
}

// text is the search text with whitespace collapsed.
func (s *state) text() string {
	return strings.Join(strings.Fields(string(s.query)), " ")
}

func (s *state) isItem(text string) bool {
	for _, item := range s.items {
		if strings.EqualFold(item, text) {
			return true
		}
	}
	return false
}

// update applies one key. done is true when the picker should close; picked
// is then true if s.matches[s.cursor] was chosen.
func (s *state) update(ev event) (done, picked bool) {
	switch ev.kind {
	case keyRune:
		s.query = append(s.query, ev.r)
		s.refilter()
	case keyBackspace:
		if len(s.query) > 0 {
			s.query = s.query[:len(s.query)-1]
			s.refilter()
		}
	case keyClear:
		s.query = s.query[:0]
		s.refilter()
	case keyUp:
		s.cursor = max(s.cursor-1, 0)
	case keyDown:
		s.cursor = max(min(s.cursor+1, len(s.matches)-1), 0)
	case keyEnter:
		return len(s.matches) > 0, len(s.matches) > 0
	case keyCancel:
		return true, false
	}
	return false, false
}

const (
	reverse = "\x1b[7m"
	dim     = "\x1b[2m"
	bold    = "\x1b[1m"
	reset   = "\x1b[0m"
)

// render draws the whole screen from the top-left corner and leaves the
// cursor at the end of the search text. Lines end in \r\n for raw mode.
func (s *state) render(title string, width, height int) string {
	rows := max(height-4, 1)
	if s.cursor < s.offset {
		s.offset = s.cursor
	} else if s.cursor >= s.offset+rows {
		s.offset = s.cursor - rows + 1
	}

	var b strings.Builder
	b.WriteString("\x1b[H")
	line := func(style, text string) {
		b.WriteString("\x1b[2K" + style + truncate(text, width-1) + reset + "\r\n")
	}
	line(bold, title)
	prompt := "> " + string(s.query)
	line("", prompt)
	found := len(s.matches)
	if found > 0 && s.matches[found-1] == newEntry {
		found--
	}
	line(dim, fmt.Sprintf("  %d/%d", found, len(s.items)))
	for i := s.offset; i < len(s.matches) && i < s.offset+rows; i++ {
		text := "+ New: " + s.text()
		if s.matches[i] != newEntry {
			text = s.items[s.matches[i]]
		}
		if i == s.cursor {
			line(reverse, "▶ "+text)
		} else {
			line("", "  "+text)
		}
	}
	b.WriteString("\x1b[J") // clear leftover rows
	fmt.Fprintf(&b, "\x1b[%d;1H%s↑/↓ move · Enter select · Esc cancel%s", height, dim, reset)
	fmt.Fprintf(&b, "\x1b[2;%dH", min(utf8.RuneCountInString(prompt)+1, width))
	return b.String()
}

func truncate(s string, width int) string {
	if width < 1 || utf8.RuneCountInString(s) <= width {
		return s
	}
	r := []rune(s)
	return string(r[:width-1]) + "…"
}
