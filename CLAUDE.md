# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ccstatus is a Go reimplementation of the TypeScript [ccstatusline](https://github.com/sirmalloc/ccstatusline): a single static binary that formats the Claude Code CLI status line. Claude Code pipes session JSON to stdin; ccstatus reads `~/.config/ccstatus/settings.json`, renders one ANSI-colored line per configured row, and prints them to stdout. Config is hand-edited JSON — no interactive TUI, no Powerline.

## Development Commands

```bash
# Build / install
go build -o ccstatus ./cmd/ccstatus
go install ./cmd/ccstatus

# Test
go test ./...
go test -cover ./...

# Lint (golangci-lint v2)
golangci-lint run

# Format (gofumpt is stricter than gofmt)
gofumpt -l -w .
goimports -w -local github.com/moond4rk/ccstatus .

# Dependencies
go mod tidy

# Regenerate the settings JSON Schema after changing a widget or a setting (a test fails when it is stale)
go run ./cmd/ccstatus schema > schema.json

# Smoke test — replays what Claude Code pipes in
echo '{"model":{"id":"claude-opus-4-7","display_name":"Opus"},"context_window":{"used_percentage":25,"context_window_size":200000}}' | ccstatus
```

## How It Works

Claude Code pipes session JSON to stdin. ccstatus parses it (`internal/status`), loads `~/.config/ccstatus/settings.json` (`internal/config`; `$XDG_CONFIG_HOME` respected), and for each configured row renders every widget via the registry (cut to its `maxWidth`), applies colors and thresholds (`internal/color`: names, `#rrggbb` and palette indexes, mapped down to what `colorLevel` allows), expands `flex-separator` to fill the flex-mode width (the terminal minus a 10- or 40-column margin), truncates to it, then post-processes (spaces → U+00A0 for VSCode; prepend `\x1b[0m` since Claude Code dims the status area) — see `internal/render` and `internal/widget`.

Render mode must never exit non-zero (Claude Code blanks the status line): settings that fail to parse render the defaults under a red error line, bad stdin renders only that line, a panic is recovered into one, and `status.Parse` coerces or drops mistyped fields instead of failing. Settings decode over `DefaultSettings()`, so omitted fields keep their defaults. Git lookups run concurrently up front (`widget.PrefetchGit`), always with `GIT_OPTIONAL_LOCKS=0`, and diff stats come from `git diff-index` / `git diff-files`, so no lookup rewrites the index. Every external command (`git`, `custom-command`) is built by `proc.Command` (`internal/proc`), which kills the command's whole process group at its deadline, so a timeout also covers what an `sh -c` started.

Subagent rows: `ccstatus subagent` reads the `subagentStatusLine` payload (`status.ParseSubagent`: `columns` + `tasks[]`), renders the `subagentLine` setting once per task through `render.RenderRow` (exact width, no flex margin) with `RenderContext.Task` set, and prints one `{"id","content"}` JSON line per non-empty row, with an empty `content` (which hides the row) for tasks whose status is in `subagentHideStatuses`; on any error it prints nothing so Claude Code keeps its default rows.

Subcommands: `ccstatus init` (write default settings.json), `validate` (strict where rendering is lenient: `validate_document.go` walks the raw JSON for unknown, removed and repeated keys, `cmd_validate.go` checks the decoded settings against the widget registry, including options that have no effect on their widget), `install` / `uninstall` (edit `~/.claude/settings.json`, or `$CLAUDE_CONFIG_DIR/`, through an order-preserving document with an atomic, symlink-following write; `install` takes `--subagent`, `--refresh N`, `--padding N`, `--hide-vim-indicator`, a fresh install writes `refreshInterval: 30`, a command that names the binary by absolute path keeps that path, and a bare `ccstatus` that is not on `PATH` gets a warning; `uninstall` only removes blocks that run ccstatus), `preview` (render settings against the embedded samples in `internal/sample`, which also supplies a sample `GitProvider`), `subagent`, `dump` (save the raw Claude Code JSON to `/tmp/ccstatus-dump.json`, or append JSON lines with `--append`; `--subagent` captures the subagent payload and prints its rows), `widgets` (list all widget types), `schema` (print the settings JSON Schema that `internal/schema` generates from the settings types and the widget registry; `schema.json` is the committed copy, and `init` writes a `$schema` reference to it).

61 widgets implement `Widget` (`Render` / `DefaultColor` / `Description`), registered in a map keyed by type string in `internal/widget/widget.go`; generic templates (`tokenWidget`, `percentageWidget`, `stringFieldWidget`, `RateLimitWidget`, `taskFieldWidget`) back many of them, and the formatters they share live in `internal/widget/format.go`. Optional interfaces: `Valuer` (numeric value for an item's `thresholds` colors), `DynamicDefaultColor` (state-driven default color), `DefaultPrefix()` (a label such as `Ctx: `), `SupportsRawValue()` (the widget has a `rawValue` mode), and unexported `displayModes()` (the `display` values an item may select, the default first), `gitFacts()` (git prefetch) and `taskScope()` (marks the `task-*` widgets that only `subagentLine` may use). Widgets never read the clock: countdowns and elapsed times are measured against `RenderContext.Now`, which `preview` shares with its sample so its output is reproducible. Data sources: Claude Code JSON, `git` commands (run in the session dir via the injectable `git.Repo`/`GitProvider`), the system. `ccstatus widgets` lists them all; `README.md` has per-widget detail.

## Conventions

- **English only, no emoji** — code, comments, docs, commit messages. Never put local machine paths in code, tests, or docs.
- **Breaking changes are fine** — prefer the best design over backward compatibility. No Powerline rendering. No interactive TUI.
- **Doc comments** — every exported type and function has a Go doc comment that starts with the identifier name.
- **Comments** — only when they earn their keep; one short line, the _why_ not the _what_.
- **Naming** — exported `PascalCase` (`Settings`, `Widget`); internal `camelCase` (`registry`).
- **Tests** — table-driven with `testify`; cover valid _and_ malformed JSON; never hardcode local paths.

## Reference

- Official Claude Code status line docs: https://code.claude.com/docs/en/statusline
- Original TypeScript implementation: https://github.com/sirmalloc/ccstatusline
- Design RFCs: `docs/rfcs/`, numbered; the highest is the latest plan
