# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Mandatory Rules

- **English Only**: All code, comments, documentation, and commit messages MUST be in English
- **No Emoji**: Never use emoji in any file (code, docs, comments, commits)
- **No Local Paths**: Never expose local machine paths in code, tests, or documentation
- **No Backward Compatibility**: Breaking changes are acceptable. Prioritize optimal design and elegant code over backward compatibility
- **No Powerline Support**: This project intentionally excludes Powerline rendering
- **No TUI**: This project uses manual JSON config editing, no interactive TUI

## Project Overview

**ccstatus** is a Go implementation of a customizable status line formatter for Claude Code CLI. It reads JSON from stdin, renders a formatted status line with model info, git status, token usage, and other metrics, then outputs ANSI-colored text to stdout.

## Build and Development Commands

```bash
go test ./...                                          # Run all tests
go test -cover ./...                                   # Run tests with coverage
go test -run TestTokenMetrics ./...                     # Run single test
golangci-lint run                                      # Run linter
gofumpt -l -w .                                        # Format (stricter than gofmt)
goimports -w -local github.com/moond4rk/ccstatus .     # Format imports
go build -o ccstatus ./cmd/ccstatus                    # Build binary
go install ./cmd/ccstatus                              # Install to $GOBIN
```

## Architecture

### Runtime Mode

Single mode: piped JSON processor (no TUI). Claude Code pipes JSON session data to stdin,
ccstatus renders ANSI-colored status line to stdout.

```
echo '{"model":{"id":"claude-sonnet-4-5","display_name":"Sonnet"},"context_window":{"used_percentage":25}}' | ccstatus
```

### CLI Flags

- `ccstatus --init` - Generate default settings.json
- `ccstatus --validate` - Validate settings.json
- `ccstatus --install` - Register in Claude Code settings.json
- `ccstatus --uninstall` - Remove from Claude Code settings.json
- `ccstatus --version` - Print version

### Core Components

| Package | Purpose |
|---------|---------|
| `cmd/ccstatus/` | Main entry point, stdin reading, CLI flags |
| `internal/config/` | Settings loading, defaults, validation, migration |
| `internal/render/` | Status line rendering, color application, truncation |
| `internal/widget/` | Widget interface, registry, all widget implementations |
| `internal/status/` | Session input struct parsing (official Claude Code JSON schema) |
| `internal/jsonl/` | JSONL transcript parsing (block-timer widget only) |
| `internal/color/` | ANSI color output via fatih/color, standard 16 named colors |
| `internal/jsonl/` | JSONL transcript parsing (block-timer widget) |
| `internal/git/` | Git branch, changes, worktree detection |
| `internal/claude/` | Claude Code settings.json integration |
| `internal/terminal/` | Terminal width detection |

### Widget System

All widgets implement a common interface:

```go
type Widget interface {
    Render(item config.WidgetItem, ctx RenderContext, settings config.Settings) string
    DefaultColor() string
    DisplayName() string
    Description() string
    SupportsRawValue() bool
}
```

Widgets are registered in a map-based registry keyed by type string.

### Available Widgets (44 registered)

Data source: (J) = from Claude Code JSON input, (G) = from git commands, (T) = from JSONL transcript, (S) = from system

- **model** (J) - Current Claude model name
- **version** (J) - Claude Code version
- **output-style** (J) - Output style name (from output_style.name)
- **session-id** (J) - Claude Code session ID
- **session-name** (J) - Custom session name (from session_name, only with --name / /rename)
- **session-clock** (J) - Session duration (from cost.total_duration_ms)
- **session-cost** (J) - Session cost in USD (from cost.total_cost_usd)
- **git-branch** (G) - Current git branch
- **git-changes** (G) - Uncommitted changes count
- **git-worktree** (G/J) - Git worktree name (workspace.git_worktree, then worktree.name, then git command)
- **tokens-input** (J) - Input tokens in current context incl. cache (from context_window.total_input_tokens; ~= context-length since CC v2.1.132)
- **tokens-output** (J) - Output tokens from the most recent API response (from context_window.total_output_tokens)
- **tokens-cached** (J) - Cached token count (from context_window.current_usage.cache_read_input_tokens)
- **tokens-total** (J) - Tokens currently in the context window (input + output)
- **current-usage-input** (J) - Current round input tokens (from context_window.current_usage)
- **current-usage-output** (J) - Current round output tokens (from context_window.current_usage)
- **cache-creation** (J) - Cache creation input tokens (from context_window.current_usage)
- **context-length** (J) - Context window usage in tokens (from context_window.current_usage)
- **context-percentage** (J) - Context usage as percentage (from context_window.used_percentage)
- **context-percentage-usable** (J) - Usable context percentage (80% of max)
- **remaining-percentage** (J) - Remaining context window percentage (from context_window.remaining_percentage)
- **cache-hit-rate** (J) - Cache read token ratio as percentage (from context_window.current_usage)
- **exceeds-200k** (J) - Warning when tokens exceed 200k threshold (from exceeds_200k_tokens)
- **api-duration** (J) - API response time (from cost.total_api_duration_ms)
- **block-timer** (J/T) - 5-hour block timer (from rate_limits.five_hour.resets_at, then cost.total_duration_ms, then JSONL)
- **rate-limits** (J) - Combined 5h/7d rate-limit usage, e.g. "5h: 3% / 7d: 12%" (from rate_limits; Pro/Max only; each window shown only when present; default prefix "Limit ")
- **rate-limit-5h** (J) - 5-hour rate-limit usage (from rate_limits.five_hour; Pro/Max only; metadata.display: percent/bar/reset/full, metadata.barWidth)
- **rate-limit-7d** (J) - 7-day rate-limit usage (from rate_limits.seven_day; Pro/Max only; metadata.display: percent/bar/reset/full, metadata.barWidth)
- **current-working-dir** (J) - Current directory (from workspace.current_dir)
- **project-dir** (J) - Project root directory (from workspace.project_dir)
- **transcript-path** (J) - Transcript file path (from transcript_path)
- **added-dirs** (J) - Directories added via /add-dir (from workspace.added_dirs; metadata.display: list for names)
- **lines-changed** (G) - Git diff lines changed (+N/-M)
- **lines-added** (G) - Git diff lines added
- **lines-removed** (G) - Git diff lines removed
- **vim-mode** (J) - Vim mode indicator (from vim.mode, only when vim enabled)
- **agent-name** (J) - Agent name (from agent.name, only with --agent flag)
- **effort** (J) - Reasoning effort level (from effort.level, only when the model supports it)
- **thinking** (J) - Extended thinking indicator (from thinking.enabled)
- **terminal-width** (S) - Terminal width in columns
- **custom-text** (-) - User-defined static text
- **custom-command** (S) - Execute shell command, display output
- **separator** (-) - Visual separator character
- **flex-separator** (-) - Expands to fill remaining width

### Configuration

Settings stored at `~/.config/ccstatus/settings.json`. Manual editing only.

`terminalWidth` (int, optional): force the status line width when auto-detection is wrong. Claude Code runs the status line command with stdio piped (not a TTY), so `terminal.Width()` often falls back to 80; setting `terminalWidth` overrides that.

### Claude Code Integration

Reads/writes `~/.claude/settings.json` (or `$CLAUDE_CONFIG_DIR/settings.json`):

```json
{
  "statusLine": {
    "type": "command",
    "command": "ccstatus",
    "padding": 0
  }
}
```

`ccstatus install` accepts optional flags: `--refresh N` writes `refreshInterval` (re-run every N seconds), `--hide-vim-indicator` writes `hideVimModeIndicator` (suppress Claude Code's built-in `-- INSERT --`). Both keys are omitted when the flag is unset.

## Code Quality Standards

### Naming Conventions

- Exported types: PascalCase (Settings, Widget, RenderContext)
- Internal types: camelCase (registry, colorMap)
- Interfaces: Descriptive names (Widget, Renderer)
- Constants: PascalCase for exported, camelCase for internal

### Documentation Requirements

Every exported type and function MUST have Go doc comments starting with the identifier name.

### Testing Requirements

- Table-driven tests for widget rendering and config parsing
- Never hardcode local paths in tests
- Test with both valid and malformed JSON inputs

## Dependencies

### Runtime

- `github.com/fatih/color` - ANSI color output
- `github.com/spf13/cobra` - CLI framework
- `github.com/tidwall/gjson` - Efficient JSONL field extraction
- `golang.org/x/term` - Terminal width detection

### Test

- `github.com/stretchr/testify` - Unit testing (required)

## Reference

- Official Claude Code status line documentation: https://code.claude.com/docs/en/statusline
- Original TypeScript implementation: https://github.com/sirmalloc/ccstatusline
