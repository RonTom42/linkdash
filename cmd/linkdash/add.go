package main

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/RonTom42/linkdash/internal/links"
	"github.com/RonTom42/linkdash/internal/picker"
)

func (c *command) add(args []string) error {
	var category string
	fs := c.flags("add", "[-c CATEGORY] [NAME URL]")
	categoryFlag(fs, &category, "category of the link (chosen from a list if omitted in a terminal)")
	pos, err := c.parse(args)
	if err != nil {
		return err
	}
	category = strings.TrimSpace(category)
	if category == "" && c.pick == nil {
		return c.usageError("a category is required (-c CATEGORY)")
	}
	if (len(pos) != 0 && len(pos) != 2) || (len(pos) == 0 && c.pick == nil) {
		return c.usageError("expected NAME and URL, got %d arguments", len(pos))
	}

	path := c.path()
	d, err := links.Load(path)
	if err != nil {
		return err
	}
	if category == "" {
		category, err = c.chooseCategory(d)
	}
	if len(pos) == 0 && err == nil {
		pos, err = c.askNameAndURL()
	}
	if errors.Is(err, picker.ErrCancelled) {
		fmt.Fprintln(c.stderr, "Nothing added.")
		return nil
	}
	if err != nil {
		return err
	}

	link, updated, err := d.Add(category, pos[0], pos[1])
	if err != nil {
		return err
	}
	if err := links.Save(path, d); err != nil {
		return err
	}
	if updated {
		fmt.Fprintf(c.stdout, "Renamed %s in %s to %q.\n", link.URL, link.Category, link.Name)
	} else {
		fmt.Fprintf(c.stdout, "Added %q (%s) to %s.\n", link.Name, link.URL, link.Category)
	}
	return nil
}

// chooseCategory lets the user pick an existing category or type a new one.
func (c *command) chooseCategory(d *links.Dashboard) (string, error) {
	var names []string
	for _, cat := range links.Group(d.Sorted("")) {
		names = append(names, cat.Name)
	}
	i, text, err := c.pick(names, picker.Options{Title: "Category: pick one or type a new name", AllowNew: true})
	if err != nil {
		return "", err
	}
	if i >= 0 {
		fmt.Fprintf(c.stderr, "Category: %s\n", names[i])
		return names[i], nil
	}
	fmt.Fprintf(c.stderr, "Category: %s (new)\n", text)
	return text, nil
}

func (c *command) askNameAndURL() ([]string, error) {
	name, err := c.ask("Name", func(s string) error {
		if s == "" {
			return errors.New("the name cannot be empty")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	url, err := c.ask("URL", func(s string) error {
		_, err := links.NormalizeURL(s)
		return err
	})
	if err != nil {
		return nil, err
	}
	return []string{name, url}, nil
}

// ask prompts on stderr and reads a line from stdin, repeating until valid
// accepts the trimmed answer. End of input returns picker.ErrCancelled.
func (c *command) ask(label string, valid func(string) error) (string, error) {
	if c.in == nil {
		c.in = bufio.NewReader(c.stdin)
	}
	for {
		fmt.Fprintf(c.stderr, "%s: ", label)
		line, readErr := c.in.ReadString('\n')
		answer := strings.TrimSpace(line)
		if readErr != nil && answer == "" {
			fmt.Fprintln(c.stderr)
			return "", picker.ErrCancelled
		}
		problem := valid(answer)
		if problem == nil {
			return answer, nil
		}
		fmt.Fprintf(c.stderr, "  %v\n", problem)
		if readErr != nil {
			return "", problem
		}
	}
}
