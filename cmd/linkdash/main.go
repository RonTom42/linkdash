// Command linkdash maintains a self-contained HTML page of categorized links.
// The page itself stores the data, so no other file is needed.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/RonTom42/linkdash/internal/picker"
	"golang.org/x/term"
)

const (
	defaultFile = "linkdash.html"
	fileEnv     = "LINKDASH_FILE"
)

const usage = `Usage: linkdash [-f FILE] <command> [arguments]

Maintains a self-contained HTML page of links sorted by category.

Commands:
  add [-c CATEGORY] [NAME URL]  add a link, or rename it if URL is already in CATEGORY;
                                in a terminal, asks for whatever is not given
  remove [-c CATEGORY] [QUERY]  remove the link whose URL or name is QUERY; without
                                QUERY, or if it is ambiguous, pick from a searchable list
  list [-c CATEGORY]            print the links grouped by category
  title TEXT                    set the page title
  render                        rewrite the page from its embedded data
  version                       print the linkdash version (also --version)

The page is FILE (-f), else $LINKDASH_FILE, else ./linkdash.html.
`

// version is set for release builds with -ldflags "-X main.version=vX.Y.Z".
var version string

// errUsage means the problem was already reported along with usage text.
var errUsage = errors.New("usage error")

// env is the outside world as seen by run.
type env struct {
	stdin          io.Reader
	stdout, stderr io.Writer
	getenv         func(string) string
	// pick chooses from a list interactively. It is nil when stdin is not a
	// terminal, which also disables prompting on stdin.
	pick func(items []string, opt picker.Options) (int, string, error)
}

func main() {
	e := env{stdin: os.Stdin, stdout: os.Stdout, stderr: os.Stderr, getenv: os.Getenv}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		e.pick = picker.Pick
	}
	err := run(os.Args[1:], e)
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
	case errors.Is(err, errUsage):
		os.Exit(2)
	default:
		fmt.Fprintln(os.Stderr, "linkdash:", err)
		os.Exit(1)
	}
}

func run(args []string, e env) error {
	var file string
	var showVersion bool
	global := flag.NewFlagSet("linkdash", flag.ContinueOnError)
	global.SetOutput(e.stderr)
	global.Usage = func() { fmt.Fprint(e.stderr, usage) }
	fileFlag(global, &file)
	global.BoolVar(&showVersion, "version", false, "print the version")
	if err := global.Parse(args); err != nil {
		return parseError(err)
	}
	if showVersion || global.Arg(0) == "version" {
		fmt.Fprintln(e.stdout, "linkdash", buildVersion())
		return nil
	}
	if global.NArg() == 0 {
		global.Usage()
		return errUsage
	}

	c := &command{env: e, file: &file}
	name, args := global.Arg(0), global.Args()[1:]
	switch name {
	case "add":
		return c.add(args)
	case "remove", "rm":
		return c.remove(args)
	case "list", "ls":
		return c.list(args)
	case "title":
		return c.title(args)
	case "render":
		return c.render(args)
	case "help":
		fmt.Fprint(e.stdout, usage)
		return nil
	}
	fmt.Fprintf(e.stderr, "linkdash: unknown command %q\n\n%s", name, usage)
	return errUsage
}

// buildVersion returns version if set, else the module version recorded by
// "go install …@vX.Y.Z", else "(devel)".
func buildVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(devel)"
}
