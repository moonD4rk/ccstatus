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

# Spelling
typos

# Dependencies
go mod tidy

# Smoke test — replays what Claude Code pipes in
echo '{"model":{"id":"claude-opus-4-7","display_name":"Opus"},"context_window":{"used_percentage":25,"context_window_size":200000}}' | ccstatus
```

## How It Works

Claude Code pipes session JSON to stdin. ccstatus parses it (`internal/status`), loads `~/.config/ccstatus/settings.json` (`internal/config`; `$XDG_CONFIG_HOME` respected), and for each configured row renders every widget via the registry, applies colors, expands `flex-separator` to fill the terminal width, truncates, then post-processes (spaces → U+00A0 for VSCode; prepend `\x1b[0m` since Claude Code dims the status area) — see `internal/render` and `internal/widget`.

Subcommands: `ccstatus init` (write default settings.json), `validate`, `install` / `uninstall` (manage the `statusLine` block in `~/.claude/settings.json`, or `$CLAUDE_CONFIG_DIR/`; `install` takes `--refresh N` and `--hide-vim-indicator`), `dump` (save the raw Claude Code JSON to `/tmp/ccstatus-dump.json`), `widgets` (list all widget types).

~47 widgets implement `Widget` (`Render` / `DefaultColor` / `DisplayName` / `Description` / `SupportsRawValue` / `DefaultPrefix` / `DefaultSuffix` — embed `noAffix` for empty affixes), registered in a map keyed by type string in `internal/widget/widget.go`; generic templates (`tokenWidget`, `percentageWidget`, `stringFieldWidget`, `RateLimitWidget`) back most of them. Data sources: Claude Code JSON, `git` commands (run in the session dir via the injectable `git.Repo`/`GitProvider`), the JSONL transcript (`block-timer` fallback only), the system. `ccstatus widgets` lists them all; `README.md` has per-widget detail.

## Conventions

- **English only, no emoji** — code, comments, docs, commit messages. Never put local machine paths in code, tests, or docs.
- **Breaking changes are fine** — prefer the best design over backward compatibility. No Powerline rendering. No interactive TUI.
- **Doc comments** — every exported type and function has a Go doc comment that starts with the identifier name.
- **Comments** — only when they earn their keep; one short line, the *why* not the *what*.
- **Naming** — exported `PascalCase` (`Settings`, `Widget`); internal `camelCase` (`registry`).
- **Tests** — table-driven with `testify`; cover valid *and* malformed JSON; never hardcode local paths.

## Reference

- Official Claude Code status line docs: https://code.claude.com/docs/en/statusline
- Original TypeScript implementation: https://github.com/sirmalloc/ccstatusline
