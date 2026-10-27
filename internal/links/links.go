// Package links holds the bookmark data model and stores it inside the
// self-contained HTML page it renders, so the page is the only state.
package links

import (
	"cmp"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
)

const (
	currentVersion = 1
	defaultTitle   = "Links"
)

// Link is a single bookmark. Within a category, links are identified by URL.
type Link struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	URL      string `json:"url"`
}

// Dashboard is the complete state embedded in the HTML page.
type Dashboard struct {
	Version int    `json:"version"`
	Title   string `json:"title"`
	Links   []Link `json:"links"`
}

// Category is a named group of links, as shown on the page.
type Category struct {
	Name  string
	Links []Link
}

// New returns an empty dashboard with the default title.
func New() *Dashboard {
	return &Dashboard{Version: currentVersion, Title: defaultTitle, Links: []Link{}}
}

// Add stores a link in category. Category names are matched case-insensitively
// and keep the spelling they already have. If the URL already exists in that
// category, its name is replaced and updated is true.
func (d *Dashboard) Add(category, name, rawURL string) (link Link, updated bool, err error) {
	category = collapseSpace(category)
	name = collapseSpace(name)
	if category == "" {
		return Link{}, false, errors.New("category is empty")
	}
	if name == "" {
		return Link{}, false, errors.New("name is empty")
	}
	u, err := NormalizeURL(rawURL)
	if err != nil {
		return Link{}, false, err
	}
	for i, l := range d.Links {
		if strings.EqualFold(l.Category, category) && l.URL == u {
			d.Links[i].Name = name
			return d.Links[i], true, nil
		}
	}
	link = Link{Category: d.canonicalCategory(category), Name: name, URL: u}
	d.Links = append(d.Links, link)
	return link, false, nil
}

// Match returns the links whose URL is query (scheme-less queries get
// https:// as in Add) or whose name equals query case-insensitively. A
// non-empty category restricts the search to that category.
func (d *Dashboard) Match(query, category string) []Link {
	query = strings.TrimSpace(query)
	normalized, _ := NormalizeURL(query)
	var out []Link
	for _, l := range d.Sorted(category) {
		if l.URL == query || l.URL == normalized || strings.EqualFold(l.Name, collapseSpace(query)) {
			out = append(out, l)
		}
	}
	return out
}

// Remove deletes the link with l's category and URL and reports whether it
// was present.
func (d *Dashboard) Remove(l Link) bool {
	for i, x := range d.Links {
		if strings.EqualFold(x.Category, l.Category) && x.URL == l.URL {
			d.Links = slices.Delete(d.Links, i, i+1)
			return true
		}
	}
	return false
}

// Sorted returns a copy of the links in display order: by category, then by
// name, both case-insensitive. A non-empty category keeps only that category.
func (d *Dashboard) Sorted(category string) []Link {
	out := make([]Link, 0, len(d.Links))
	for _, l := range d.Links {
		if category == "" || strings.EqualFold(l.Category, strings.TrimSpace(category)) {
			out = append(out, l)
		}
	}
	slices.SortFunc(out, compareLinks)
	return out
}

// Group splits links, which must be in Sorted order, into categories.
func Group(links []Link) []Category {
	var out []Category
	for _, l := range links {
		if n := len(out); n > 0 && strings.EqualFold(out[n-1].Name, l.Category) {
			out[n-1].Links = append(out[n-1].Links, l)
			continue
		}
		out = append(out, Category{Name: l.Category, Links: []Link{l}})
	}
	return out
}

func (d *Dashboard) canonicalCategory(category string) string {
	for _, l := range d.Links {
		if strings.EqualFold(l.Category, category) {
			return l.Category
		}
	}
	return category
}

func compareLinks(a, b Link) int {
	return cmp.Or(
		strings.Compare(strings.ToLower(a.Category), strings.ToLower(b.Category)),
		strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)),
		strings.Compare(a.Name, b.Name),
		strings.Compare(a.URL, b.URL),
		strings.Compare(a.Category, b.Category),
	)
}

func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// Schemes that would run code when the link is clicked.
var blockedSchemes = map[string]bool{"javascript": true, "vbscript": true, "data": true}

// NormalizeURL trims raw, prefixes https:// when it has no scheme
// ("go.dev/doc", "localhost:8080") and rejects schemes that execute code.
func NormalizeURL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.New("URL is empty")
	}
	if !hasScheme(s) {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	if blockedSchemes[strings.ToLower(u.Scheme)] {
		return "", fmt.Errorf("URL scheme %q is not allowed", u.Scheme)
	}
	if (u.Scheme == "http" || u.Scheme == "https") && u.Host == "" {
		return "", fmt.Errorf("invalid URL %q: missing host", raw)
	}
	return s, nil
}

// safeURL reports whether a stored URL may be rendered as a link. It repeats
// NormalizeURL's scheme check so hand-edited data cannot inject scripts.
func safeURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && !blockedSchemes[strings.ToLower(u.Scheme)]
}

// hasScheme reports whether s starts with "scheme:", treating "host:port" as
// schemeless.
func hasScheme(s string) bool {
	scheme, rest, ok := strings.Cut(s, ":")
	if !ok || scheme == "" || rest != "" && isDigit(rest[0]) {
		return false
	}
	for i, c := range []byte(scheme) {
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z':
		case i > 0 && (isDigit(c) || c == '+' || c == '-' || c == '.'):
		default:
			return false
		}
	}
	return true
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }
