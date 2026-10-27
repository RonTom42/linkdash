package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/RonTom42/linkdash/internal/links"
	"github.com/RonTom42/linkdash/internal/picker"
)

func (c *command) remove(args []string) error {
	var category string
	fs := c.flags("remove", "[-c CATEGORY] [URL or NAME]")
	categoryFlag(fs, &category, "only consider links in this category")
	pos, err := c.parse(args)
	if err != nil {
		return err
	}
	query := strings.Join(pos, " ")

	path := c.path()
	d, err := links.Load(path)
	if err != nil {
		return err
	}
	candidates := d.Sorted(category)
	if len(candidates) == 0 {
		return fmt.Errorf("no links to remove in %s%s", path, inCategory(category))
	}

	var target links.Link
	matches := d.Match(query, category)
	switch {
	case query != "" && len(matches) == 1:
		target = matches[0]
	case c.pick == nil && query == "":
		return errors.New("no link given and stdin is not a terminal; pass the URL or name of the link to remove")
	case c.pick == nil && len(matches) == 0:
		return fmt.Errorf("no link has the URL or name %q%s", query, inCategory(category))
	case c.pick == nil:
		var b strings.Builder
		fmt.Fprintf(&b, "%q matches %d links; use -c CATEGORY or the exact URL:\n", query, len(matches))
		_ = writeLinks(&b, matches) // writing to a strings.Builder cannot fail
		return errors.New(strings.TrimSuffix(b.String(), "\n"))
	default:
		i, _, err := c.pick(labels(candidates), picker.Options{Title: "Remove a link", Query: query})
		if errors.Is(err, picker.ErrCancelled) {
			fmt.Fprintln(c.stderr, "Nothing removed.")
			return nil
		}
		if err != nil {
			return err
		}
		target = candidates[i]
	}

	d.Remove(target)
	if err := links.Save(path, d); err != nil {
		return err
	}
	undo := []string{"linkdash"}
	if *c.file != "" {
		undo = append(undo, "-f", shellQuote(*c.file))
	}
	undo = append(undo, "add", "-c", shellQuote(target.Category), shellQuote(target.Name), shellQuote(target.URL))
	fmt.Fprintf(c.stdout, "Removed %q (%s) from %s.\nUndo: %s\n", target.Name, target.URL, target.Category, strings.Join(undo, " "))
	return nil
}
