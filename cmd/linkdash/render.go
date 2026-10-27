package main

import (
	"fmt"

	"github.com/RonTom42/linkdash/internal/links"
)

func (c *command) render(args []string) error {
	c.flags("render", "")
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
	if err := links.Save(path, d); err != nil {
		return err
	}
	fmt.Fprintf(c.stdout, "Wrote %s.\n", path)
	return nil
}
