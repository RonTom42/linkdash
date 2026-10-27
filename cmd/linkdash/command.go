package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type command struct {
	env
	file *string
	fs   *flag.FlagSet
	in   *bufio.Reader // stdin for prompts, created on first use
}

// flags starts a subcommand's flag set; every subcommand also accepts -f.
func (c *command) flags(name, synopsis string) *flag.FlagSet {
	c.fs = flag.NewFlagSet(name, flag.ContinueOnError)
	c.fs.SetOutput(c.stderr)
	c.fs.Usage = func() {
		fmt.Fprintf(c.stderr, "Usage: linkdash %s %s\n", name, synopsis)
		c.fs.PrintDefaults()
	}
	fileFlag(c.fs, c.file)
	return c.fs
}

// parse parses the subcommand's flags, which may appear before, between or
// after the positional arguments, and returns the positional arguments.
func (c *command) parse(args []string) ([]string, error) {
	var positional []string
	for {
		if err := c.fs.Parse(args); err != nil {
			return nil, parseError(err)
		}
		rest := c.fs.Args()
		if consumed := len(args) - len(rest); consumed > 0 && args[consumed-1] == "--" {
			return append(positional, rest...), nil
		}
		if len(rest) == 0 {
			return positional, nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}

func (c *command) usageError(format string, a ...any) error {
	fmt.Fprintf(c.stderr, "linkdash %s: %s\n", c.fs.Name(), fmt.Sprintf(format, a...))
	c.fs.Usage()
	return errUsage
}

// path resolves the page location: -f, then $LINKDASH_FILE, then the default.
func (c *command) path() string {
	p := *c.file
	if p == "" {
		p = c.getenv(fileEnv)
	}
	if p == "" {
		return defaultFile
	}
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, rest)
		}
	}
	return p
}

// fileFlag defines -f/-file on fs. The current value is kept as the default
// because defining a flag resets its variable, and a global -f may be set.
func fileFlag(fs *flag.FlagSet, file *string) {
	fs.StringVar(file, "f", *file, "the HTML `file` to maintain")
	fs.StringVar(file, "file", *file, "the HTML `file` to maintain")
}

func categoryFlag(fs *flag.FlagSet, category *string, help string) {
	fs.StringVar(category, "c", "", help)
	fs.StringVar(category, "category", "", help)
}

// parseError converts flag errors, which the flag package has already
// printed together with the usage text, to errUsage.
func parseError(err error) error {
	if errors.Is(err, flag.ErrHelp) {
		return err
	}
	return errUsage
}
