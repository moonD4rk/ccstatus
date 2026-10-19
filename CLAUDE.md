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

# Smoke test — replays what Claude Code pipes in
echo '{"model":{"id":"claude-opus-4-7","display_name":"Opus"},"context_window":{"used_percentage":25,"context_window_size":200000}}' | ccstatus
```

## How It Works

Claude Code pipes session JSON to stdin. ccstatus parses it (`internal/status`), loads `~/.config/ccstatus/settings.json` (`internal/config`; `$XDG_CONFIG_HOME` respected), and for each configured row renders every widget via the registry (cut to its `maxWidth`), applies colors and thresholds, expands `flex-separator` to fill the flex-mode width (the terminal minus a 10- or 40-column margin), truncates to it, then post-processes (spaces → U+00A0 for VSCode; prepend `\x1b[0m` since Claude Code dims the status area) — see `internal/render` and `internal/widget`.

Render mode must never exit non-zero (Claude Code blanks the status line): settings that fail to parse render the defaults under a red error line, bad stdin renders only that line, and `status.Parse` coerces or drops mistyped fields instead of failing. Settings decode over `DefaultSettings()`, so omitted fields keep their defaults. Git lookups run concurrently up front (`widget.PrefetchGit`), always with `GIT_OPTIONAL_LOCKS=0`, and diff stats come from `git diff-index` / `git diff-files`, so no lookup rewrites the index.

Subagent rows: `ccstatus subagent` reads the `subagentStatusLine` payload (`status.ParseSubagent`: `columns` + `tasks[]`), renders the `subagentLine` setting once per task through `render.RenderRow` (exact width, no flex margin) with `RenderContext.Task` set, and prints one `{"id","content"}` JSON line per non-empty row; on any error it prints nothing so Claude Code keeps its default rows.

Subcommands: `ccstatus init` (write default settings.json), `validate`, `install` / `uninstall` (edit `~/.claude/settings.json`, or `$CLAUDE_CONFIG_DIR/`, through an order-preserving document with an atomic, symlink-following write; `install` takes `--subagent`, `--refresh N`, `--padding N`, `--hide-vim-indicator`, and a fresh install writes `refreshInterval: 30`; `uninstall` only removes blocks that run ccstatus), `preview` (render settings against the embedded samples in `internal/sample`), `subagent`, `dump` (save the raw Claude Code JSON to `/tmp/ccstatus-dump.json`, or append JSON lines with `--append`), `widgets` (list all widget types).

61 widgets implement `Widget` (`Render` / `DefaultColor` / `DisplayName` / `Description` / `SupportsRawValue` / `DefaultPrefix` / `DefaultSuffix` — embed `noAffix` for empty affixes), registered in a map keyed by type string in `internal/widget/widget.go`; generic templates (`tokenWidget`, `percentageWidget`, `stringFieldWidget`, `RateLimitWidget`, `taskFieldWidget`) back many of them. Optional interfaces: `Valuer` (numeric value for an item's `thresholds` colors), `DynamicDefaultColor` (state-driven default color), and unexported `displayModes()` (`metadata.display` values checked by `validate`), `gitFacts()` (git prefetch) and `taskScope()` (marks the `task-*` widgets that only `subagentLine` may use). Data sources: Claude Code JSON, `git` commands (run in the session dir via the injectable `git.Repo`/`GitProvider`), the JSONL transcript (`block-timer` fallback only), the system. `ccstatus widgets` lists them all; `README.md` has per-widget detail.

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
