package main

import (
	"fmt"
	"strings"

	"github.com/RonTom42/linkdash/internal/links"
)

func (c *command) title(args []string) error {
	c.flags("title", "TEXT")
	pos, err := c.parse(args)
	if err != nil {
		return err
	}
	title := strings.Join(strings.Fields(strings.Join(pos, " ")), " ")
	if title == "" {
		return c.usageError("expected the title text")
	}
	path := c.path()
	d, err := links.Load(path)
	if err != nil {
		return err
	}
	d.Title = title
	if err := links.Save(path, d); err != nil {
		return err
	}
	fmt.Fprintf(c.stdout, "Title set to %q.\n", title)
	return nil
}
