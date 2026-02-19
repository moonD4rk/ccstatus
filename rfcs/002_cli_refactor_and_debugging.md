# RFC 002: CLI Refactor and Debugging Support

Status: Completed
Author: @moond4rk
Date: 2026-02-15

## Summary

Refactor the ccstatus CLI from flag-based operations (`--init`, `--install`) to standard
cobra subcommands (`ccstatus init`, `ccstatus install`), add a `dump` subcommand for
debugging real Claude Code JSON input, and fix the default configuration to match
implemented widgets.

## Motivation

### 1. CLI structure violates cobra conventions

The current CLI uses boolean flags for distinct operations:

```
ccstatus --init        # flag-based
ccstatus --validate    # flag-based
ccstatus --install     # flag-based
ccstatus --uninstall   # flag-based
```

This is non-idiomatic for cobra. These are independent actions, not modifiers to the
default behavior. The standard cobra pattern uses subcommands:

```
ccstatus init          # subcommand
ccstatus validate      # subcommand
ccstatus install       # subcommand
ccstatus uninstall     # subcommand
```

Problems with the current flag approach:
- `ccstatus --init --validate` is ambiguous (which runs first?)
- `ccstatus init` currently fails with `unknown command "init"`
- Help output mixes operational flags with modifier flags
- Cannot add per-subcommand flags (e.g., `ccstatus init --force`)

### 2. No way to debug real Claude Code input

When developing and testing widgets, there is no way to capture the actual JSON
that Claude Code pipes to ccstatus. Developers must guess at field availability.
A `dump` subcommand solves this by saving stdin to a file for inspection.

### 3. Default config references unimplemented widget

The default configuration includes `git-changes` (id=7), but this widget has no
implementation. It is silently skipped, causing a trailing separator to appear and
then be cleaned up. This confuses users who expect to see git change counts.

### 4. Status line missing token/context data

The current default config only includes `context-length`, which requires
`current_usage` data. In practice, this is often empty early in a session. Token
widgets (`tokens-input`, `context-percentage`) that read from `total_input_tokens`
and `used_percentage` are more reliable and already implemented but not in the
default config.

## Scope

### In Scope

1. Refactor CLI from flags to cobra subcommands
2. Add `dump` subcommand for capturing Claude Code JSON input
3. Add `widgets` subcommand to list all available widget types
4. Fix default config to only reference implemented widgets
5. Improve default config with better widget selection

### Out of Scope

- Implementing missing widgets (git-changes, session-cost, etc.) -- separate phase
- Settings migration logic changes
- New widget types

## Design

### CLI Structure (After Refactor)

```
ccstatus                           # Default: read stdin JSON, render status line
ccstatus init [--force]            # Generate default settings.json
ccstatus validate                  # Validate settings.json
ccstatus install                   # Register in Claude Code settings.json
ccstatus uninstall                 # Remove from Claude Code settings.json
ccstatus dump [--output FILE]      # Dump stdin JSON to file (for debugging)
ccstatus widgets                   # List all registered widget types
ccstatus --version                 # Print version (cobra built-in)
```

Help output:

```
A customizable status line formatter for Claude Code CLI.

Usage:
  ccstatus [flags]
  ccstatus [command]

Available Commands:
  dump        Dump raw JSON input from Claude Code for debugging
  init        Generate default settings.json
  install     Register ccstatus in Claude Code settings
  uninstall   Remove ccstatus from Claude Code settings
  validate    Validate settings.json
  widgets     List all available widget types

Flags:
  -h, --help      help for ccstatus
  -v, --version   version for ccstatus

Use "ccstatus [command] --help" for more information about a command.
```

### Subcommand Details

#### `ccstatus` (root, no subcommand)

Default behavior unchanged: read JSON from stdin, render status line to stdout.

```go
rootCmd := &cobra.Command{
    Use:   "ccstatus",
    Short: "Customizable status line for Claude Code",
    RunE:  runStatusLine,
}
```

#### `ccstatus init`

Generate default `settings.json` at the config path.

```go
initCmd := &cobra.Command{
    Use:   "init",
    Short: "Generate default settings.json",
    RunE:  runInit,
}
initCmd.Flags().Bool("force", false, "Overwrite existing settings.json")
```

Behavior:
- If file exists and `--force` not set: print path and exit with error
- If file exists and `--force` set: overwrite
- If file does not exist: create with default settings
- Print the file path to stderr on success

#### `ccstatus validate`

Validate the settings file.

```go
validateCmd := &cobra.Command{
    Use:   "validate",
    Short: "Validate settings.json",
    RunE:  runValidate,
}
```

Behavior:
- Load and parse settings.json
- Check for unknown widget types (warn, not error)
- Print "Settings are valid" to stderr on success
- Exit with error and message on failure

#### `ccstatus install` / `ccstatus uninstall`

Register/remove ccstatus in Claude Code's `~/.claude/settings.json`.

```go
installCmd := &cobra.Command{
    Use:   "install",
    Short: "Register ccstatus in Claude Code settings",
    RunE:  runInstall,
}
```

#### `ccstatus dump`

Read stdin JSON from Claude Code and save it to a file for debugging.

```go
dumpCmd := &cobra.Command{
    Use:   "dump",
    Short: "Dump raw JSON input from Claude Code for debugging",
    Long:  "Read JSON from stdin and save to a file. Useful for inspecting what Claude Code sends.",
    RunE:  runDump,
}
dumpCmd.Flags().StringP("output", "o", "", "Output file path (default: /tmp/ccstatus-dump.json)")
```

Behavior:
1. Read all of stdin
2. Pretty-print the JSON (indent with 2 spaces) for readability
3. Write to the output file (default: `/tmp/ccstatus-dump.json`)
4. Also pass through to normal rendering (so the status line still works)
5. Print the saved file path to stderr

Usage: Temporarily change Claude Code's command to `ccstatus dump` to capture input,
then inspect `/tmp/ccstatus-dump.json` to see exactly what fields are available.

Advanced usage with `--output`:
```bash
ccstatus dump --output ~/debug/claude-input.json
```

#### `ccstatus widgets`

List all registered widget types with their descriptions.

```go
widgetsCmd := &cobra.Command{
    Use:   "widgets",
    Short: "List all available widget types",
    RunE:  runWidgets,
}
```

Output format:
```
Available widgets (37):

  model                        Current Claude model name (cyan)
  version                      Claude Code version (white)
  session-id                   Claude Code session ID (white)
  session-cost                 Session cost in USD (green)
  session-clock                Session duration (white)
  git-branch                   Current git branch name (magenta)
  git-changes                  Uncommitted changes count (yellow)
  git-worktree                 Git worktree info (magenta)
  tokens-input                 Total input token count (white)
  tokens-output                Total output token count (white)
  tokens-cached                Cached token count (white)
  tokens-total                 Total token count (input + output) (white)
  current-usage-input          Current round input token count (white)
  current-usage-output         Current round output token count (white)
  cache-creation               Cache creation input token count (white)
  context-length               Context window usage in tokens (white)
  context-percentage           Context usage as percentage of max window (white)
  context-percentage-usable    Usable context percentage (80% of max) (white)
  remaining-percentage         Remaining context window percentage (white)
  cache-hit-rate               Cache read token ratio as percentage (cyan)
  api-duration                 API response time (white)
  block-timer                  5-hour session block timer (white)
  current-working-dir          Current working directory (blue)
  project-dir                  Project root directory (blue)
  transcript-path              Transcript file path (white)
  lines-changed                Git diff lines changed (+N/-M) (green)
  lines-added                  Git diff lines added (green)
  lines-removed                Git diff lines removed (red)
  output-style                 Current output style name (white)
  vim-mode                     Current vim mode indicator (yellow)
  agent-name                   Agent name when using --agent flag (cyan)
  exceeds-200k                 Warning when tokens exceed 200k (red)
  terminal-width               Terminal width in columns (white)
  custom-text                  User-defined static text (white)
  custom-command               Execute shell command, display output (white)
  separator                    Visual separator character (white)
  flex-separator               Expands to fill remaining width
```

### Default Config Update

The default configuration now features a rich 2-line layout with comprehensive
session information:

```go
func DefaultSettings() Settings {
    return Settings{
        Version:          CurrentVersion,  // 4
        ColorLevel:       2,
        FlexMode:         "full-minus-40",
        CompactThreshold: 60,
        DefaultSeparator: "|",
        DefaultPadding:   " ",
        Lines: [][]WidgetItem{
            {
                // Line 1: model, context, tokens, cache, git, lines, cost
                {ID: "1", Type: "model", Color: "cyan"},
                {ID: "2", Type: "separator"},
                {ID: "3", Type: "context-percentage", Color: "brightBlack"},
                {ID: "4", Type: "separator"},
                {ID: "5", Type: "tokens-input", Color: "white"},
                {ID: "6", Type: "separator"},
                {ID: "7", Type: "tokens-output", Color: "white"},
                {ID: "8", Type: "separator"},
                {ID: "9", Type: "cache-hit-rate", Color: "cyan"},
                {ID: "10", Type: "separator"},
                {ID: "11", Type: "git-branch", Color: "magenta"},
                {ID: "12", Type: "separator"},
                {ID: "13", Type: "lines-added", Color: "green"},
                {ID: "14", Type: "lines-removed", Color: "red"},
                {ID: "15", Type: "separator"},
                {ID: "16", Type: "session-cost", Color: "green"},
            },
            {
                // Line 2: working directory (left) + session clock (right)
                {ID: "17", Type: "current-working-dir", Color: "blue", RawValue: true},
                {ID: "18", Type: "flex-separator"},
                {ID: "19", Type: "session-clock", Color: "white"},
            },
        },
    }
}
```

### File Structure Changes

Before:
```
cmd/ccstatus/
  main.go              # Everything in one file
```

After:
```
cmd/ccstatus/
  main.go              # Entry point, root command setup
  cmd_dump.go          # dump subcommand
  cmd_init.go          # init subcommand
  cmd_install.go       # install/uninstall subcommands
  cmd_validate.go      # validate subcommand
  cmd_widgets.go       # widgets subcommand
```

## Implementation Plan

### Step 1: Refactor root command and extract subcommands (Completed)

Split `main.go` into separate files per subcommand. Convert `--init`, `--validate`,
`--install`, `--uninstall` flags into cobra subcommands. Keep default (no subcommand)
behavior as stdin rendering.

### Step 2: Add `dump` subcommand (Completed)

Implement the dump command that saves stdin JSON to a file while still rendering
the status line normally.

### Step 3: Add `widgets` subcommand (Completed)

Implement the widgets listing command using the existing widget registry.

### Step 4: Update default config (Completed)

Replaced unimplemented widgets with full 2-line layout featuring model, context,
tokens, cache, git, lines, cost, and session clock.

### Step 5: Add `--force` flag to init (Completed)

Allow `ccstatus init --force` to overwrite existing config.

### Step 6: Tests (Completed)

- Test each subcommand's behavior
- Test default config only references registered widgets
- Verify `ccstatus -h` shows subcommands correctly
- Test dump writes valid JSON

## Testing Strategy

```bash
# Manual integration test: capture real Claude Code input
# 1. Install dump mode
ccstatus install   # or manually edit ~/.claude/settings.json command to "ccstatus dump"

# 2. Use Claude Code normally, then inspect
cat /tmp/ccstatus-dump.json | jq .

# 3. Replay captured input to test rendering
cat /tmp/ccstatus-dump.json | ccstatus

# 4. List available widgets
ccstatus widgets
```

## Backward Compatibility

- `ccstatus` (no args) behavior is unchanged
- `ccstatus --version` / `-v` still works (cobra built-in)
- `ccstatus --init` will break (becomes `ccstatus init`)
- This is acceptable per CLAUDE.md: "Breaking changes are acceptable"

## Resolved Questions

1. **`dump` rendering**: Both save + render, so it can be used as a drop-in replacement.
2. **`widgets` listing**: Only shows implemented/registered widgets.
3. **Default dump location**: `/tmp/ccstatus-dump.json` (temporary, no config pollution).
