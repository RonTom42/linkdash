package main

import (
	"testing"
)

func TestShellQuote(t *testing.T) {
	for in, want := range map[string]string{
		"Dev":                 "Dev",
		"https://go.dev/doc":  "https://go.dev/doc",
		"Go docs":             "'Go docs'",
		"https://x.ch/?a=1&b": "'https://x.ch/?a=1&b'",
		"it's":                `'it'\''s'`,
		"":                    "''",
	} {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %s, want %s", in, got, want)
		}
	}
}
