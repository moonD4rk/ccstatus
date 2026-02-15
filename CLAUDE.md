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
| `internal/status/` | StatusJSON input struct parsing (official Claude Code JSON schema) |
| `internal/jsonl/` | JSONL transcript parsing (block-timer widget only) |
| `internal/color/` | ANSI color codes, color names, color levels |
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

### Available Widgets (36 registered)

Data source: (J) = from Claude Code JSON input, (G) = from git commands, (T) = from JSONL transcript, (S) = from system

- **model** (J) - Current Claude model name
- **version** (J) - Claude Code version
- **output-style** (J) - Output style (text/json/stream-json)
- **session-id** (J) - Claude Code session ID
- **git-branch** (G) - Current git branch
- **git-changes** (G) - Uncommitted changes count
- **git-worktree** (G) - Git worktree info
- **tokens-input** (J) - Input token count (from context_window)
- **tokens-output** (J) - Output token count (from context_window)
- **tokens-cached** (J) - Cached token count (from context_window.current_usage)
- **tokens-total** (J) - Total token count
- **current-usage-input** (J) - Current round input tokens (from context_window.current_usage)
- **current-usage-output** (J) - Current round output tokens (from context_window.current_usage)
- **cache-creation** (J) - Cache creation input tokens (from context_window.current_usage)
- **context-length** (J) - Context window usage (from context_window.current_usage)
- **context-percentage** (J) - Context usage as percentage (from context_window.used_percentage)
- **context-percentage-usable** (J) - Usable context percentage (80% of max)
- **remaining-percentage** (J) - Remaining context window percentage (from context_window.remaining_percentage)
- **block-timer** (J/T) - 5-hour session block timer (from cost.total_duration_ms, JSONL fallback)
- **session-clock** (J) - Session duration (from cost.total_duration_ms)
- **session-cost** (J) - Session cost in USD (from cost.total_cost_usd)
- **api-duration** (J) - API response time (from cost.total_api_duration_ms)
- **current-working-dir** (J) - Current directory (from workspace.current_dir)
- **project-dir** (J) - Project root directory (from workspace.project_dir)
- **transcript-path** (J) - Transcript file path (from transcript_path)
- **terminal-width** (S) - Terminal width in columns
- **custom-text** (-) - User-defined static text
- **custom-command** (S) - Execute shell command, display output
- **vim-mode** (J) - Vim mode indicator (from vim.mode, only when vim enabled)
- **agent-name** (J) - Agent name (from agent.name, only with --agent flag)
- **exceeds-200k** (J) - Warning when tokens exceed 200k threshold

### Configuration

Settings stored at `~/.config/ccstatus/settings.json`. Manual editing only.

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
