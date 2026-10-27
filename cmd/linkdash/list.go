package main

import (
	"fmt"

	"github.com/RonTom42/linkdash/internal/links"
)

func (c *command) list(args []string) error {
	var category string
	fs := c.flags("list", "[-c CATEGORY]")
	categoryFlag(fs, &category, "only list this category")
	pos, err := c.parse(args)
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return c.usageError("unexpected argument %q", pos[0])
	}
	path := c.path()
	d, err := links.Load(path)
	if err != nil {
		return err
	}
	sorted := d.Sorted(category)
	if len(sorted) == 0 {
		fmt.Fprintf(c.stderr, "No links in %s%s.\n", path, inCategory(category))
		return nil
	}
	return writeLinks(c.stdout, sorted)
}
