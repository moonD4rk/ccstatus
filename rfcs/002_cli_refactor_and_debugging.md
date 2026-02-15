# RFC 002: CLI Refactor and Debugging Support

Status: Draft
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
ccstatus version                   # Print version (replaces -v/--version)
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
Available widgets:

  model                      Current Claude model name (cyan)
  version                    Claude Code version (brightBlack)
  git-branch                 Current git branch name (magenta)
  tokens-input               Total input token count (brightBlack)
  tokens-output              Total output token count (brightBlack)
  tokens-cached              Cached token count (brightBlack)
  tokens-total               Total token count (input + output) (brightBlack)
  context-length             Context window usage in tokens (brightBlack)
  context-percentage         Context usage as percentage (brightBlack)
  context-percentage-usable  Usable context percentage (brightBlack)
  custom-text                User-defined static text (white)
  separator                  Visual separator character (brightBlack)
  flex-separator             Expands to fill remaining width
```

### Default Config Update

Current default config references `git-changes` which is not implemented. Update to
only include implemented and reliable widgets:

```go
func DefaultSettings() Settings {
    return Settings{
        Version:          CurrentVersion,
        ColorLevel:       2,
        FlexMode:         "full-minus-40",
        CompactThreshold: 60,
        DefaultSeparator: "|",
        DefaultPadding:   " ",
        Lines: [][]WidgetItem{
            {
                {ID: "1", Type: "model", Color: "cyan"},
                {ID: "2", Type: "separator"},
                {ID: "3", Type: "context-percentage", Color: "brightBlack"},
                {ID: "4", Type: "separator"},
                {ID: "5", Type: "git-branch", Color: "magenta"},
            },
        },
    }
}
```

Changes from current default:
- Replaced `context-length` with `context-percentage` (more reliable: reads from
  `used_percentage` which is available earlier than `current_usage`)
- Removed `git-changes` (not implemented)
- Removed separators around removed widgets (no dangling separators)

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

### Step 1: Refactor root command and extract subcommands

Split `main.go` into separate files per subcommand. Convert `--init`, `--validate`,
`--install`, `--uninstall` flags into cobra subcommands. Keep default (no subcommand)
behavior as stdin rendering.

### Step 2: Add `dump` subcommand

Implement the dump command that saves stdin JSON to a file while still rendering
the status line normally.

### Step 3: Add `widgets` subcommand

Implement the widgets listing command using the existing widget registry.

### Step 4: Update default config

Replace unimplemented `git-changes` with reliable `context-percentage`. Remove
trailing dangling separators.

### Step 5: Add `--force` flag to init

Allow `ccstatus init --force` to overwrite existing config.

### Step 6: Tests

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

## Open Questions

1. Should `dump` also render the status line, or only save the JSON?
   - Recommendation: Both (save + render) so it can be used as a drop-in replacement
2. Should `widgets` show implementation status (implemented vs planned)?
   - Recommendation: Only show implemented/registered widgets
3. Default dump file location: `/tmp/ccstatus-dump.json` or `~/.config/ccstatus/dump.json`?
   - Recommendation: `/tmp/ccstatus-dump.json` (temporary, no config pollution)
