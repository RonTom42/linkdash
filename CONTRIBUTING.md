# Contributing to linkdash

Thanks for your interest! Bug reports, ideas and pull requests are all welcome. For anything bigger than a small fix, please open an issue first so we can agree on the approach before you spend time on it.

## Development setup

The Go and golangci-lint versions are pinned in [`mise.toml`](mise.toml). With [mise](https://mise.jdx.dev) installed:

```sh
mise install      # installs the pinned Go and golangci-lint
mise run check    # lint + tests, exactly what CI runs
```

Without mise, install Go 1.27+ and golangci-lint v2 yourself and run the commands from `mise.toml` directly.

| Task | Command |
| --- | --- |
| `mise run test` | `go test -race ./...` |
| `mise run lint` | `golangci-lint run` (linters and gofmt/goimports formatting) |
| `mise run build` | `go build -o linkdash ./cmd/linkdash` |
| `mise run check` | lint and test |

Run a single test with `go test ./internal/links -run TestSaveLoadRoundTrip -v`. Fix formatting with `golangci-lint fmt`.

## Project layout

| Path | Contents |
| --- | --- |
| `cmd/linkdash/` | The CLI: `main.go` (entry point and dispatch), `command.go` (flag and path helpers), one file per command, `format.go` (terminal output). |
| `internal/links/` | Data model, sorting, URL validation, and reading/writing the HTML page. `template.html` is the page template, embedded into the binary. |
| `internal/picker/` | The interactive fzf-style list. Key decoding, state and drawing are pure functions; only `Pick` touches the terminal. |

## Design rules

These keep linkdash what it is; changes that break them are unlikely to be merged.

- **The page stays self-contained.** Hand-written HTML, CSS and vanilla JavaScript only. No frameworks, CDNs, web fonts, favicons or any other external request. Links must keep working with JavaScript disabled.
- **The HTML file is the only state.** No config or sidecar files.
- **Minimal dependencies.** The Go standard library plus `golang.org/x/term`. No CLI or TUI frameworks.
- **Existing pages must keep loading.** The data block `<script type="application/json" id="linkdash-data">` is found by that exact text, so don't change it, and don't change the JSON shape without bumping the data version and handling older pages.
- **Rendering is deterministic.** `linkdash render` on an up-to-date page must not change a byte (there is a test for it), so no timestamps or random IDs in the page.

## Testing

- Add or update tests for every change. Command-level tests drive `run()` with a fake environment (see `cmd/linkdash/main_test.go`), including a fake picker and stdin.
- The interactive list needs a real terminal. Try it with `go run ./cmd/linkdash -f /tmp/t.html remove`. To automate it, run it under `script -q /dev/null …` and pipe keystrokes in with short sleeps between them.
- For changes to `template.html`, check the page in at least one browser, in light and dark mode and at phone width.

## Pull requests

- Keep each pull request focused on one change and describe what it changes and why.
- Make sure `mise run check` passes. CI runs the same checks on Linux and macOS.
- Update `README.md` for user-visible changes, and `CLAUDE.md` if the architecture changes.

## Releasing (maintainers)

Tag a version and push the tag. The release workflow runs the tests and then GoReleaser, which builds the macOS and Linux binaries and publishes them with checksums on GitHub Releases:

```sh
git tag v0.2.0
git push origin v0.2.0
```

To try the release build locally: `mise x goreleaser@2 -- goreleaser release --snapshot --clean` (output goes to `dist/`).
