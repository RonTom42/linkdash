package main

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/RonTom42/linkdash/internal/links"
)

func inCategory(category string) string {
	if category == "" {
		return ""
	}
	return fmt.Sprintf(" in category %q", category)
}

// writeLinks prints sorted links grouped under their categories.
func writeLinks(w io.Writer, sorted []links.Link) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	for _, cat := range links.Group(sorted) {
		fmt.Fprintln(tw, cat.Name)
		for _, l := range cat.Links {
			fmt.Fprintf(tw, "  %s\t%s\n", l.Name, l.URL)
		}
	}
	return tw.Flush()
}

// labels formats sorted links as aligned picker rows: category, name, URL.
func labels(sorted []links.Link) []string {
	catWidth, nameWidth := 0, 0
	for _, l := range sorted {
		catWidth = max(catWidth, len([]rune(l.Category))+2)
		nameWidth = min(max(nameWidth, len([]rune(l.Name))), 40)
	}
	out := make([]string, len(sorted))
	for i, l := range sorted {
		out[i] = fmt.Sprintf("%-*s  %-*s  %s", catWidth, "["+l.Category+"]", nameWidth, l.Name, l.URL)
	}
	return out
}

// shellQuote quotes s for POSIX shells when it contains special characters.
func shellQuote(s string) string {
	if s != "" && !strings.ContainsFunc(s, func(r rune) bool { return !shellSafe(r) }) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// shellSafe reports whether r needs no quoting in POSIX shells.
func shellSafe(r rune) bool {
	return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9' || strings.ContainsRune("-_./:@%+=,", r)
}
