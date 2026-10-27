# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`linkdash` is a single Go binary that maintains one self-contained static HTML page of links (bookmarks), grouped by category, with categories and links sorted alphabetically. The CLI does the following:
- Adds links. With arguments it runs non-interactively. In a terminal it asks for anything missing, choosing the category from a live-filter list that can also create one.
- Removes links, by argument or through the same live-filter list.
- Lists links, sets the page title, and re-renders the page.

## Commands

Go and golangci-lint are pinned in `mise.toml` and are not on the shell PATH. Use the mise tasks, or prefix direct tool calls with `mise exec --` (run `mise install` once on a fresh machine).

```sh
mise run check     # lint + race tests: exactly what CI runs, must pass
mise run test      # go test -race ./...
mise run lint      # golangci-lint run (standard linters + misspell, gofmt/goimports)
mise run build     # ./linkdash (gitignored)
mise exec -- go test ./internal/links -run TestSaveLoadRoundTrip -v   # single test
mise exec -- golangci-lint fmt                                      # fix formatting/imports
```

The page file is chosen by `-f FILE`, then `$LINKDASH_FILE`, then `./linkdash.html`.

## Constraints

- **The page:** hand-written HTML, CSS and vanilla JS only. No frameworks, CDNs, web fonts, favicons or any other external request. It must work offline as a single file and without JS; JS only adds search.
- **Go dependencies:** stdlib plus `golang.org/x/term`, which the picker needs for raw mode. No CLI or TUI frameworks; subcommands use stdlib `flag`.
- **No sidecar files:** the HTML page is the only state. Don't add config or data files.

## Architecture: the HTML file is the database

- **`internal/links/store.go`:**
  - Every rendered page ends with `<script type="application/json" id="linkdash-data">…</script>`.
  - `Parse` locates it by those exact strings (`dataOpen`/`dataClose`) and unmarshals it. Changing the marker or the JSON shape breaks existing pages unless you add a migration; bump `currentVersion` when the shape changes.
  - A missing file loads as an empty dashboard. An existing file without the marker returns `ErrNotDashboard`, so the tool never overwrites someone else's HTML.
- **Escaping relies on `html/template`:**
  - `html/template` treats `type="application/json"` as a JS context, so `{{.Data}}` is marshalled with `<`, `>` and `&` written as JSON Unicode escape sequences. That is why `</script>` can never appear inside the block and a plain string search finds its end.
  - Don't pre-marshal the JSON and inject it as `template.HTML`/`template.JS`.
  - `TestSaveLoadRoundTrip` guards this with hostile strings.
- **Every write re-renders the whole page** from the embedded data (`Render`). Template changes therefore reach existing pages on their next write or `linkdash render`.
  - Output must stay deterministic: `render` on an up-to-date page is byte-identical (tested). Don't add timestamps.
  - `Save` writes atomically (temp file plus rename), keeps the file mode, and follows symlinks so a symlinked page stays a symlink.
- **Data model (`internal/links/links.go`):**
  - A flat `[]Link{Category, Name, URL}`. Categories exist only through their links and vanish with the last one.
  - Category names compare case-insensitively everywhere (`Add` reuses the existing spelling).
  - Within a category, a link is identified by its URL: re-adding a URL renames the link.
  - Sorting (`compareLinks`) is case-insensitive byte order, so accented initials (Ä) sort after Z.
- **URL safety:**
  - `NormalizeURL` adds `https://` to scheme-less input (`host:port` counts as scheme-less) and rejects `javascript:`, `data:` and `vbscript:`.
  - Any other scheme is allowed, including custom ones like `obsidian://`. That is why the template's `href` func passes URLs as `template.URL` after re-checking with `safeURL`, instead of relying on html/template's http/https/mailto-only filter.
- **`cmd/linkdash/`:** `main.go` holds `env`, `run` dispatch and version; `command.go` the flag and path helpers; then one file per command (`add.go` with its prompts, `remove.go`, …) and `format.go` for terminal output. Tests mirror this layout, with the shared harness in `main_test.go`.
  - `run(args, env)` is the testable entry point. `env.pick` is nil when stdin is not a terminal; that nil is the single "interactive?" switch, gating both the picker and `ask`'s line prompts on `env.stdin`. Tests inject a fake picker and a stdin string.
  - `add` takes `[-c CATEGORY] [NAME URL]`. Without a TTY, both are required.
    - A missing category opens the picker with `AllowNew`, listing existing categories.
    - Missing NAME and URL are prompted on stderr, and the URL prompt repeats until `NormalizeURL` accepts the input.
    - Esc or EOF prints "Nothing added." and exits 0.
  - `remove QUERY` removes immediately on exactly one exact match (URL or name). On zero or several matches, it opens the picker pre-filled with QUERY on a TTY and errors otherwise.
  - Each subcommand re-declares `-f` bound to the shared variable. `fileFlag` passes the current value as the default because `flag.StringVar` resets the variable.
- **`internal/picker`:** an fzf-like list split into pure `decode`, `state.update` and `state.render` (all unit-tested) and a thin `Pick` loop. `Pick` opens `/dev/tty` in raw mode and draws on the alternate screen, which keeps stdout clean. Raw mode needs `\r\n` line endings.
  - With `Options.AllowNew`, typed text that isn't exactly an item (case-insensitive) appears as a last `+ New:` row (`newEntry`, index -1).
  - `Pick` returns that index plus the whitespace-collapsed search text.

## Repository, CI and releases

- **Module and license:** MIT licensed. The module path in `go.mod` names the upstream repository. Forks keep it unless they become a separate project. Otherwise, don't hard-code the repo's owner or name in docs: use `gh`'s `{owner}`/`{repo}` placeholders or relative wording, so the instructions also work in a fork.
- **Community files:** `README.md`, `CONTRIBUTING.md` and `SECURITY.md` are user-facing; keep them in sync with behavior changes. Issue forms live in `.github/ISSUE_TEMPLATE/`.
- **CI (`.github/workflows/ci.yml`):** runs the mise tasks on Ubuntu and macOS.
- **Action pinning:** actions are pinned to commit SHAs with a version comment, and Dependabot (`.github/dependabot.yml`) updates them. Keep that style.
- **Branch protection:** in the upstream repository, `main` is protected by the GitHub ruleset "Protect main". Rulesets live in the repository settings, so a fork starts without one.
  - a pull request is required, with 0 approvals
  - the `Lint`, `Test (ubuntu-latest)` and `Test (macos-latest)` checks must pass
  - force pushes and deletion are blocked
  - Repository admins can bypass it, so a direct push to `main` may succeed. Don't push to `main` anyway: work on a branch and open a PR with `gh pr create`.
  - The ruleset names the checks, not the workflow. If a CI job is renamed or the OS matrix changes, PRs wait forever for a check that never runs. Update the ruleset's required checks in the same change: `gh api 'repos/{owner}/{repo}/rulesets'` fills in the placeholders from the current checkout.
- **Version:** `main.version` is injected by GoReleaser's ldflags. `buildVersion()` falls back to the module version from `go install …@vX.Y.Z`, then to `(devel)`.
- **Releases:** pushing a `vX.Y.Z` tag runs `.github/workflows/release.yml`, which tests and then runs GoReleaser (`.goreleaser.yaml`, linux and darwin × amd64 and arm64).
  - Archive names deliberately contain no version, because the README's `releases/latest/download/linkdash_<os>_<arch>.tar.gz` URLs depend on it.
  - To check the release config locally, run `mise x goreleaser@2 -- goreleaser release --snapshot --clean`. This needs a git remote and a commit.

## Manual checks

The picker needs a real terminal, so test it by hand with `./linkdash -f /tmp/t.html remove` or `add`. You can also drive it from a script through `script -q /dev/null ./linkdash add`, piping in keystrokes with sleeps between them. To check the page visually, open it in a browser: try light and dark mode, the `/` shortcut to focus search, Enter to open the first match, and Esc to clear.
