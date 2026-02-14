# RFC 001: ccstatus Architecture and Design

Status: Draft
Author: @moond4rk
Date: 2026-02-14

## Summary

ccstatus is a Go reimplementation of the TypeScript ccstatusline project. It is a single-binary CLI tool that reads JSON from Claude Code via stdin, renders a customizable status line with ANSI colors, and outputs to stdout. This RFC defines the architecture, data structures, algorithms, and implementation plan.

## Motivation

- **Supply chain security**: Eliminate npm dependency tree (~23 devDeps with deep transitive dependencies)
- **Distribution**: Single static binary vs `npx`/Node.js runtime requirement
- **Startup performance**: ~5ms compiled binary vs ~200-500ms Node.js cold start
- **Simplicity**: No TUI, no Powerline -- focused on the core rendering pipeline

## Scope

### In Scope

- Piped mode: stdin JSON -> rendered ANSI status line -> stdout
- 25 widget types (22 from TypeScript version + 3 new from official API)
- Multi-line status line rendering
- Flex separator (fills remaining terminal width)
- ANSI color support (16 / 256 / truecolor)
- JSONL transcript parsing (block timer only; token/context data now provided by Claude Code JSON)
- Git integration (branch, changes, worktree)
- Claude Code settings.json integration (install/uninstall)
- Configuration via `~/.config/ccstatus/settings.json`
- CLI flags: `--init`, `--validate`, `--install`, `--uninstall`, `--version`

### Out of Scope

- TUI / interactive configuration UI
- Powerline rendering and themes
- Gradient text rendering
- Widget inline editors
- Custom keybinds

## Claude Code Status Line Protocol

Reference: https://code.claude.com/docs/en/statusline

### How It Works

Claude Code runs the configured command and pipes JSON session data to stdin. The command
reads JSON, renders output, and prints ANSI-colored text to stdout. Claude Code displays
whatever the command prints.

### Update Timing

The command runs after each new assistant message, when permission mode changes, or when
vim mode toggles. Updates are debounced at 300ms. If a new update triggers while the command
is still running, the in-flight execution is cancelled.

### Output Protocol

**Official requirements** (from Claude Code documentation):

1. Each `fmt.Println` produces a separate row in the status area
2. ANSI escape codes are supported for colors (e.g., `\033[32m` for green)
3. OSC 8 escape sequences are supported for clickable links (terminal support required)
4. Errors should go to stderr with non-zero exit code
5. Scripts that exit with non-zero codes or produce no output cause the status line to go blank
6. Keep output short -- the status bar has limited width, long output may be truncated

**Practical workarounds** (from ccstatusline, not officially documented but address real issues):

7. Replace all regular spaces (0x20) with non-breaking spaces (U+00A0) to prevent VSCode
   from trimming trailing whitespace. Without this, status lines with trailing padding may
   render incorrectly in VSCode terminals.
8. Prepend `\x1b[0m` (ANSI reset) to each output line. Claude Code applies a dim attribute
   to the status area; without an explicit reset, all text appears dimmed.
9. Skip empty lines (lines containing only ANSI codes with no visible text) to avoid blank
   rows in the status area.

Note: Items 7-9 are enabled by default in ccstatus. They solve practical rendering issues
observed in real usage but are not part of the official protocol.

### Notifications

System notifications (MCP server errors, auto-updates, token warnings) display on the right
side of the same row as the status line. On narrow terminals, these may truncate status line
output. The `padding` setting in Claude Code's settings.json controls additional horizontal
spacing.

### Official JSON Input Schema

Claude Code sends the following JSON to stdin. **All fields are optional** and may be absent
or null. The full schema from the official documentation:

```json
{
  "cwd": "/current/working/directory",
  "session_id": "abc123...",
  "transcript_path": "/path/to/transcript.jsonl",
  "model": {
    "id": "claude-opus-4-6",
    "display_name": "Opus"
  },
  "workspace": {
    "current_dir": "/current/working/directory",
    "project_dir": "/original/project/directory"
  },
  "version": "1.0.80",
  "output_style": {
    "name": "default"
  },
  "cost": {
    "total_cost_usd": 0.01234,
    "total_duration_ms": 45000,
    "total_api_duration_ms": 2300,
    "total_lines_added": 156,
    "total_lines_removed": 23
  },
  "context_window": {
    "total_input_tokens": 15234,
    "total_output_tokens": 4521,
    "context_window_size": 200000,
    "used_percentage": 8,
    "remaining_percentage": 92,
    "current_usage": {
      "input_tokens": 8500,
      "output_tokens": 1200,
      "cache_creation_input_tokens": 5000,
      "cache_read_input_tokens": 2000
    }
  },
  "exceeds_200k_tokens": false,
  "vim": {
    "mode": "NORMAL"
  },
  "agent": {
    "name": "security-reviewer"
  }
}
```

### Field Availability Notes

| Field | Notes |
|-------|-------|
| `model` | Object with `id` and `display_name`. For backward compat, may also be a plain string in older versions. |
| `cwd`, `workspace.current_dir` | Same value; prefer `workspace.current_dir` for consistency. |
| `workspace.project_dir` | Directory where Claude Code was launched; may differ from `cwd`. |
| `context_window.current_usage` | `null` before the first API call in a session. |
| `context_window.used_percentage` | May be `null` early in the session. Calculated from input tokens only (input + cache_creation + cache_read), not output tokens. |
| `context_window.context_window_size` | 200000 by default, 1000000 for extended context models. |
| `exceeds_200k_tokens` | Fixed 200k threshold regardless of actual context window size. |
| `vim` | Only present when vim mode is enabled. |
| `agent` | Only present when running with `--agent` flag or agent settings configured. |

### Data Source Strategy

The official JSON input now provides token metrics and context window data directly,
eliminating the need to parse JSONL transcripts for most widgets.

| Data | Primary Source (JSON) | Fallback Source (JSONL) |
|------|-----------------------|-------------------------|
| Input tokens | `context_window.total_input_tokens` | JSONL parsing |
| Output tokens | `context_window.total_output_tokens` | JSONL parsing |
| Cached tokens | `context_window.current_usage.cache_read_input_tokens` + `cache_creation_input_tokens` | JSONL parsing |
| Context length | `context_window.current_usage.input_tokens` + cache tokens | JSONL parsing |
| Context percentage | `context_window.used_percentage` | Manual calculation |
| Context window size | `context_window.context_window_size` | Model ID heuristic |
| Session cost | `cost.total_cost_usd` | N/A |
| Session duration | `cost.total_duration_ms` | JSONL timestamps |
| Block timer | N/A | JSONL parsing (required) |

## Architecture

### Directory Structure

```
ccstatus/
  cmd/
    ccstatus/
      main.go              # Entry point, stdin reading, CLI flags
  internal/
    config/
      config.go            # Settings struct, load/save, defaults
      config_test.go
      widget.go            # WidgetItem struct
      migration.go         # Settings version migration
    render/
      render.go            # Status line rendering pipeline
      render_test.go
      truncate.go          # Terminal width truncation
      format.go            # Token formatting, separator formatting
    widget/
      widget.go            # Widget interface and registry
      model.go             # Model widget
      version.go           # Version widget
      output_style.go      # OutputStyle widget
      session_id.go        # ClaudeSessionId widget
      git_branch.go        # GitBranch widget
      git_changes.go       # GitChanges widget
      git_worktree.go      # GitWorktree widget
      tokens.go            # TokensInput, TokensOutput, TokensCached, TokensTotal
      context.go           # ContextLength, ContextPercentage, ContextPercentageUsable
      block_timer.go       # BlockTimer widget (requires JSONL parsing)
      session_clock.go     # SessionClock widget (from cost.total_duration_ms)
      session_cost.go      # SessionCost widget (from cost.total_cost_usd)
      current_dir.go       # CurrentWorkingDir widget
      terminal_width.go    # TerminalWidth widget
      custom_text.go       # CustomText widget
      custom_command.go    # CustomCommand widget
      vim_mode.go          # VimMode widget (from vim.mode)
      agent_name.go        # AgentName widget (from agent.name)
      exceeds_200k.go      # Exceeds200K widget (from exceeds_200k_tokens)
      widget_test.go
    jsonl/
      jsonl.go             # Token metrics, session duration, block metrics
      jsonl_test.go
    color/
      color.go             # ANSI color codes, color names, color levels
      color_test.go
    git/
      git.go               # Git command wrappers
    claude/
      claude.go            # Claude Code settings.json read/write
    terminal/
      terminal.go          # Terminal width detection
    status/
      status.go            # StatusJSON input struct, parsing
  testdata/
    settings.json          # Example settings for tests
    transcript.jsonl       # Example JSONL for tests
```

### Data Flow

```
Claude Code
    |
    | (pipes JSON to stdin, see "Official JSON Input Schema" above)
    v
cmd/ccstatus/main.go
    |
    | 1. Read stdin
    | 2. Parse StatusJSON (includes context_window, cost, vim, agent)
    | 3. Load Settings from ~/.config/ccstatus/settings.json
    | 4. Parse JSONL transcript (only if block-timer widget is configured)
    | 5. Detect terminal width
    v
internal/render/render.go
    |
    | 6. For each line in settings.lines:
    |    a. Render each widget via registry
    |    b. Apply colors (fg, bg, bold) using fatih/color
    |    c. Insert separators
    |    d. Expand flex separators
    |    e. Truncate to terminal width with "..."
    |    f. Replace spaces with non-breaking spaces (U+00A0)
    |    g. Prepend ANSI reset (\x1b[0m)
    v
stdout (ANSI colored status line, one line per fmt.Println)
```

## Data Structures

### StatusJSON (Input from Claude Code)

Based on the official Claude Code status line documentation
(https://code.claude.com/docs/en/statusline).

```go
// StatusJSON represents the JSON payload piped from Claude Code.
// All fields are optional and may be absent or null.
type StatusJSON struct {
    Cwd              string         `json:"cwd,omitempty"`
    SessionID        string         `json:"session_id,omitempty"`
    TranscriptPath   string         `json:"transcript_path,omitempty"`
    Model            ModelField     `json:"model,omitempty"`
    Workspace        *Workspace     `json:"workspace,omitempty"`
    Version          string         `json:"version,omitempty"`
    OutputStyle      *OutputStyle   `json:"output_style,omitempty"`
    Cost             *CostInfo      `json:"cost,omitempty"`
    ContextWindow    *ContextWindow `json:"context_window,omitempty"`
    Exceeds200K      *bool          `json:"exceeds_200k_tokens,omitempty"`
    Vim              *VimInfo       `json:"vim,omitempty"`
    Agent            *AgentInfo     `json:"agent,omitempty"`
}

// ModelField handles the model field being either a string or an object.
// Official API always sends an object {id, display_name}, but older versions
// may send a plain string. Custom UnmarshalJSON handles both cases.
type ModelField struct {
    ID          string
    DisplayName string
}

func (m *ModelField) UnmarshalJSON(data []byte) error {
    var s string
    if json.Unmarshal(data, &s) == nil {
        m.ID = s
        return nil
    }
    var obj struct {
        ID          string `json:"id"`
        DisplayName string `json:"display_name"`
    }
    if err := json.Unmarshal(data, &obj); err != nil {
        return err
    }
    m.ID = obj.ID
    m.DisplayName = obj.DisplayName
    return nil
}

type Workspace struct {
    CurrentDir string `json:"current_dir,omitempty"`
    ProjectDir string `json:"project_dir,omitempty"`
}

type OutputStyle struct {
    Name string `json:"name,omitempty"`
}

type CostInfo struct {
    TotalCostUSD       *float64 `json:"total_cost_usd,omitempty"`
    TotalDurationMS    *float64 `json:"total_duration_ms,omitempty"`
    TotalAPIDurationMS *float64 `json:"total_api_duration_ms,omitempty"`
    TotalLinesAdded    *int     `json:"total_lines_added,omitempty"`
    TotalLinesRemoved  *int     `json:"total_lines_removed,omitempty"`
}

// ContextWindow holds context window usage data provided directly by Claude Code.
// This eliminates the need to parse JSONL transcripts for most token-related widgets.
type ContextWindow struct {
    TotalInputTokens    *int           `json:"total_input_tokens,omitempty"`
    TotalOutputTokens   *int           `json:"total_output_tokens,omitempty"`
    ContextWindowSize   *int           `json:"context_window_size,omitempty"`   // 200000 or 1000000
    UsedPercentage      *float64       `json:"used_percentage,omitempty"`       // Pre-calculated by Claude Code
    RemainingPercentage *float64       `json:"remaining_percentage,omitempty"`
    CurrentUsage        *CurrentUsage  `json:"current_usage,omitempty"`         // null before first API call
}

// CurrentUsage holds token counts from the most recent API call.
// used_percentage is calculated from input tokens only:
// input_tokens + cache_creation_input_tokens + cache_read_input_tokens.
type CurrentUsage struct {
    InputTokens              int `json:"input_tokens"`
    OutputTokens             int `json:"output_tokens"`
    CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
    CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// VimInfo is only present when vim mode is enabled in Claude Code.
type VimInfo struct {
    Mode string `json:"mode,omitempty"` // "NORMAL" or "INSERT"
}

// AgentInfo is only present when running with --agent flag or agent settings.
type AgentInfo struct {
    Name string `json:"name,omitempty"`
}
```

### Settings

```go
const CurrentVersion = 3

// Settings represents the ccstatus configuration.
type Settings struct {
    Version                int          `json:"version"`
    Lines                  [][]WidgetItem `json:"lines"`
    FlexMode               string       `json:"flexMode"`
    CompactThreshold       int          `json:"compactThreshold"`
    ColorLevel             int          `json:"colorLevel"`
    DefaultSeparator       string       `json:"defaultSeparator,omitempty"`
    DefaultPadding         string       `json:"defaultPadding,omitempty"`
    InheritSeparatorColors bool         `json:"inheritSeparatorColors"`
    OverrideBackgroundColor string      `json:"overrideBackgroundColor,omitempty"`
    OverrideForegroundColor string      `json:"overrideForegroundColor,omitempty"`
    GlobalBold             bool         `json:"globalBold"`
}

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
                {ID: "3", Type: "context-length", Color: "brightBlack"},
                {ID: "4", Type: "separator"},
                {ID: "5", Type: "git-branch", Color: "magenta"},
                {ID: "6", Type: "separator"},
                {ID: "7", Type: "git-changes", Color: "yellow"},
            },
        },
    }
}
```

### WidgetItem

```go
// WidgetItem represents a single widget in the status line configuration.
type WidgetItem struct {
    ID              string            `json:"id"`
    Type            string            `json:"type"`
    Color           string            `json:"color,omitempty"`
    BackgroundColor string            `json:"backgroundColor,omitempty"`
    Bold            bool              `json:"bold,omitempty"`
    Character       string            `json:"character,omitempty"`
    RawValue        bool              `json:"rawValue,omitempty"`
    CustomText      string            `json:"customText,omitempty"`
    CommandPath     string            `json:"commandPath,omitempty"`
    MaxWidth        int               `json:"maxWidth,omitempty"`
    PreserveColors  bool              `json:"preserveColors,omitempty"`
    Timeout         int               `json:"timeout,omitempty"`
    Merge           any               `json:"merge,omitempty"` // bool or "no-padding"
    Metadata        map[string]string `json:"metadata,omitempty"`
}
```

### RenderContext

```go
// RenderContext carries runtime data available to all widgets during rendering.
// Most token/context data comes directly from StatusJSON.ContextWindow (official API).
// BlockMetrics is the only field that still requires JSONL transcript parsing.
type RenderContext struct {
    Data          *StatusJSON
    BlockMetrics  *BlockMetrics   // From JSONL parsing (block-timer widget only)
    TerminalWidth int
}
```

Note: `TokenMetrics` and `SessionDuration` have been removed from RenderContext because
Claude Code now provides this data directly in the JSON input:
- Token data: `StatusJSON.ContextWindow.TotalInputTokens`, `TotalOutputTokens`, `CurrentUsage`
- Session duration: `StatusJSON.Cost.TotalDurationMS`
- Context percentage: `StatusJSON.ContextWindow.UsedPercentage`

### BlockMetrics

```go
// BlockMetrics tracks 5-hour session block timing.
type BlockMetrics struct {
    StartTime    time.Time
    LastActivity time.Time
}
```

## Widget Interface

```go
// Widget defines the contract for all status line widgets.
type Widget interface {
    // Render produces the widget text for the status line.
    // Returns empty string if the widget has nothing to display.
    Render(item config.WidgetItem, ctx RenderContext, settings config.Settings) string

    // DefaultColor returns the default foreground color name.
    DefaultColor() string

    // DisplayName returns the human-readable widget name.
    DisplayName() string

    // Description returns a short description of what the widget shows.
    Description() string

    // SupportsRawValue indicates if the widget has a compact output mode.
    SupportsRawValue() bool
}
```

### Widget Registry

```go
var registry = map[string]Widget{
    // Model & session
    "model":                       &ModelWidget{},
    "version":                     &VersionWidget{},
    "output-style":                &OutputStyleWidget{},
    "session-id":                  &SessionIDWidget{},

    // Git
    "git-branch":                  &GitBranchWidget{},
    "git-changes":                 &GitChangesWidget{},
    "git-worktree":                &GitWorktreeWidget{},

    // Token metrics (data from context_window JSON)
    "tokens-input":                &TokensInputWidget{},
    "tokens-output":               &TokensOutputWidget{},
    "tokens-cached":               &TokensCachedWidget{},
    "tokens-total":                &TokensTotalWidget{},

    // Context window (data from context_window JSON)
    "context-length":              &ContextLengthWidget{},
    "context-percentage":          &ContextPercentageWidget{},
    "context-percentage-usable":   &ContextPercentageUsableWidget{},

    // Session metrics (data from cost JSON)
    "block-timer":                 &BlockTimerWidget{},     // Requires JSONL parsing
    "session-clock":               &SessionClockWidget{},
    "session-cost":                &SessionCostWidget{},

    // Environment
    "current-working-dir":         &CurrentDirWidget{},
    "terminal-width":              &TerminalWidthWidget{},

    // User-defined
    "custom-text":                 &CustomTextWidget{},
    "custom-command":              &CustomCommandWidget{},

    // New widgets from official Claude Code API
    "vim-mode":                    &VimModeWidget{},
    "agent-name":                  &AgentNameWidget{},
    "exceeds-200k":                &Exceeds200KWidget{},
}

// Get returns the widget for the given type string, or nil if unknown.
func Get(widgetType string) Widget {
    return registry[widgetType]
}
```

## Key Algorithms

### Token Formatting

```go
func FormatTokens(count int) string {
    if count >= 1_000_000 {
        return fmt.Sprintf("%.1fM", float64(count)/1_000_000)
    }
    if count >= 1000 {
        return fmt.Sprintf("%.1fk", float64(count)/1000)
    }
    return strconv.Itoa(count)
}
```

### Context Window Size Resolution

Claude Code now provides `context_window.context_window_size` directly in the JSON input.
The model ID heuristic is kept only as a fallback for older Claude Code versions.

```go
const (
    defaultMaxTokens    = 200_000
    defaultUsableTokens = 160_000  // 80% of 200k
    longMaxTokens       = 1_000_000
    longUsableTokens    = 800_000  // 80% of 1M
)

type ContextConfig struct {
    MaxTokens    int
    UsableTokens int
}

// GetContextConfig resolves context window size.
// Primary: use context_window.context_window_size from JSON input.
// Fallback: heuristic based on model ID (for older Claude Code versions).
func GetContextConfig(data *StatusJSON) ContextConfig {
    // Primary: use official context_window_size if available
    if data.ContextWindow != nil && data.ContextWindow.ContextWindowSize != nil {
        size := *data.ContextWindow.ContextWindowSize
        return ContextConfig{
            MaxTokens:    size,
            UsableTokens: size * 80 / 100,
        }
    }

    // Fallback: model ID heuristic
    lower := strings.ToLower(data.Model.ID)
    if strings.Contains(lower, "claude-sonnet-4-5") && strings.Contains(lower, "[1m]") {
        return ContextConfig{MaxTokens: longMaxTokens, UsableTokens: longUsableTokens}
    }
    return ContextConfig{MaxTokens: defaultMaxTokens, UsableTokens: defaultUsableTokens}
}
```

### Context Percentage

Claude Code now provides `context_window.used_percentage` directly. The manual calculation
is kept as a fallback. Note: `used_percentage` is calculated from input tokens only
(input_tokens + cache_creation_input_tokens + cache_read_input_tokens), not output tokens.

```go
// GetContextPercentage returns the context usage percentage.
// Primary: use pre-calculated used_percentage from JSON input.
// Fallback: calculate from current_usage tokens and context_window_size.
func GetContextPercentage(data *StatusJSON) float64 {
    if data.ContextWindow != nil && data.ContextWindow.UsedPercentage != nil {
        return *data.ContextWindow.UsedPercentage
    }
    // Fallback: manual calculation if used_percentage is null (early in session)
    if data.ContextWindow != nil && data.ContextWindow.CurrentUsage != nil {
        cu := data.ContextWindow.CurrentUsage
        contextLength := cu.InputTokens + cu.CacheCreationInputTokens + cu.CacheReadInputTokens
        cfg := GetContextConfig(data)
        if cfg.MaxTokens == 0 {
            return 0
        }
        pct := float64(contextLength) / float64(cfg.MaxTokens) * 100
        if pct > 100 {
            return 100
        }
        return pct
    }
    return 0
}
```

### JSONL Transcript Parsing

With the official Claude Code API now providing token metrics and session duration directly
in the JSON input, JSONL transcript parsing is only required for the **block-timer** widget
(5-hour session block tracking). This data is not available in the official API.

The `internal/jsonl` package uses `gjson` for efficient field extraction from JSONL lines.

### Session Duration Formatting

Session duration now comes from `cost.total_duration_ms` in the JSON input. The formatting
function converts milliseconds to human-readable format:

```go
func FormatDuration(ms float64) string {
    totalSec := int(ms / 1000)
    hours := totalSec / 3600
    mins := (totalSec % 3600) / 60
    if hours == 0 && mins == 0 {
        return "<1m"
    }
    if hours == 0 {
        return fmt.Sprintf("%dm", mins)
    }
    if mins == 0 {
        return fmt.Sprintf("%dhr", hours)
    }
    return fmt.Sprintf("%dhr %dm", hours, mins)
}
```

### Block Metrics (5-Hour Session Blocks)

```go
func GetBlockMetrics() *BlockMetrics {
    // 1. Glob ~/.claude/projects/**/*.jsonl
    // 2. Sort by mtime (most recent first)
    // 3. Progressive lookback: 10h -> 20h -> 48h
    // 4. Extract timestamps from each file
    // 5. Find gaps >= 5 hours (session boundary)
    // 6. Return start of current block and last activity
    // 7. Floor startTime to hour boundary
}
```

### Rendering Pipeline

```go
func RenderStatusLine(items []config.WidgetItem, settings config.Settings, ctx RenderContext) string {
    // 1. Iterate items, render each widget via registry
    // 2. Skip items that produce empty output
    // 3. Apply colors (foreground, background, bold) with override support
    // 4. Insert separators between items (skip before first, after last)
    // 5. Apply padding around widget content
    // 6. Handle flex-separator: split into parts, calculate remaining space, distribute
    // 7. Handle merge mode (combine adjacent widgets with/without padding)
    // 8. Truncate to terminal width with "..." if needed
    //
    // Post-processing (practical workarounds, see Output Protocol in this RFC):
    // 9. Replace spaces with non-breaking spaces U+00A0 (workaround: VSCode trims trailing spaces)
    // 10. Prepend ANSI reset \x1b[0m (workaround: Claude Code applies dim to status area)
    // 11. Skip line if no visible text remains after stripping ANSI codes
}
```

### Terminal Width Detection

```go
func GetWidth() int {
    // 1. Try golang.org/x/term.GetSize(fd) on stdout
    // 2. Fallback: parse `stty size` output
    // 3. Fallback: parse `tput cols` output
    // 4. Return 0 if all fail
}
```

### FlexMode Width Calculation

```go
func CalculateWidth(detected int, flexMode string, compactThreshold int, contextPct float64) int {
    switch flexMode {
    case "full":
        return detected - 6
    case "full-minus-40":
        return detected - 40
    case "full-until-compact":
        if contextPct >= float64(compactThreshold) {
            return detected - 40
        }
        return detected - 6
    }
    return detected
}
```

## Color System

### Supported Colors (32 total)

8 basic: black, red, green, yellow, blue, magenta, cyan, white
8 bright: brightBlack, brightRed, ..., brightWhite
8 bg: bgBlack, bgRed, ..., bgWhite
8 bgBright: bgBrightBlack, ..., bgBrightWhite

### Custom Color Formats

- `ansi256:N` - Direct ANSI 256 color code (0-255)
- `hex:RRGGBB` - RGB hex color (truecolor only)

### Color Levels

- 0: No colors (strip all ANSI codes)
- 1: ANSI 16 colors
- 2: ANSI 256 colors (default)
- 3: Truecolor (24-bit RGB)

### ANSI Code Generation

```go
func ApplyColor(text, fg, bg string, bold bool, level int) string {
    // Build ANSI escape sequence based on color level
    // fg: \x1b[3Xm (16) or \x1b[38;5;Nm (256) or \x1b[38;2;R;G;Bm (true)
    // bg: \x1b[4Xm (16) or \x1b[48;5;Nm (256) or \x1b[48;2;R;G;Bm (true)
    // bold: \x1b[1m ... \x1b[22m
    // reset: \x1b[0m
}
```

## Claude Code Integration

### Config Directory Resolution

```go
func GetClaudeConfigDir() string {
    if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
        return dir
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".claude")
}
```

### Install/Uninstall

```go
func Install() error {
    // Read existing ~/.claude/settings.json
    // Set statusLine.type = "command"
    // Set statusLine.command = "ccstatus"
    // Set statusLine.padding = 0
    // Write back (preserve other fields)
}

func Uninstall() error {
    // Read existing settings
    // Remove statusLine field
    // Write back
}
```

## Configuration Compatibility

ccstatus settings.json is a **subset** of the TypeScript ccstatusline format. The Go version ignores unknown fields (forward-compatible). Users migrating from ccstatusline can use the same settings file -- Powerline and TUI-only fields are silently ignored.

## Implementation Plan

Each phase includes corresponding testify unit tests. Tests are written alongside
implementation, not deferred to the end.

### Phase 1: Core Engine (Priority: P0)

Goal: `echo '{"model":{"id":"claude-sonnet-4-5","display_name":"Sonnet"}}' | ccstatus` produces colored output.

1. `go mod tidy` - Add all dependencies (cobra, fatih/color, gjson, x/term, testify)
2. `cmd/ccstatus/main.go` - cobra root command, stdin reading, flag definitions
3. `internal/status/status.go` - StatusJSON parsing with ModelField, ContextWindow, VimInfo, AgentInfo
4. `internal/config/config.go` - Settings load/save/defaults
5. `internal/color/color.go` - ANSI color system based on fatih/color
6. `internal/widget/widget.go` - Widget interface and registry
7. `internal/render/render.go` - Rendering pipeline (colors, separators, truncation, U+00A0, ANSI reset)
8. Simple widgets: model, version, git-branch, custom-text, separator

### Phase 2: Token & Context Widgets (Priority: P0)

Goal: Display token usage and context percentage from Claude Code JSON input (no JSONL needed).

9. Token widgets: tokens-input, tokens-output, tokens-cached, tokens-total
   (read from `context_window.total_input_tokens`, `total_output_tokens`, `current_usage`)
10. Context widgets: context-length, context-percentage, context-percentage-usable
    (read from `context_window.used_percentage`, `context_window_size`)
11. `internal/terminal/terminal.go` - Width detection
12. Flex separator expansion

### Phase 3: Remaining Widgets (Priority: P1)

Goal: Complete all widget implementations.

13. git-changes, git-worktree (from git commands)
14. session-clock (from `cost.total_duration_ms`), session-cost (from `cost.total_cost_usd`)
15. block-timer (requires `internal/jsonl/` JSONL parsing with gjson)
16. output-style, session-id, terminal-width, current-working-dir
17. custom-command (with timeout and preserveColors)
18. New widgets: vim-mode (from `vim.mode`), agent-name (from `agent.name`), exceeds-200k (from `exceeds_200k_tokens`)

### Phase 4: Integration & Polish (Priority: P1)

19. `internal/claude/claude.go` - Install/uninstall
20. `--init`, `--validate` flags
21. Settings migration
22. Multi-line rendering
23. Merge mode (widget combining)

### Phase 5: Release (Priority: P2)

24. Integration test with official JSON schema sample
25. GoReleaser configuration
26. README documentation

## Dependencies

```
require (
    github.com/fatih/color      // ANSI color output
    github.com/spf13/cobra      // CLI framework
    github.com/tidwall/gjson    // JSONL field extraction
    golang.org/x/term           // Terminal width detection
)

require (
    github.com/stretchr/testify // Unit testing (test only)
)
```

## Open Questions

1. Should `--init` generate a minimal or full default config?
   - Recommendation: Minimal (only line 1 with basic widgets)
2. Should we support `go install` as the primary distribution method, or also provide Homebrew tap?
   - Recommendation: Both -- `go install` for Go users, GitHub Releases for binary downloads
3. Should custom-command widget use `sh -c` or direct exec?
   - Recommendation: `sh -c` for shell feature compatibility (pipes, env vars)
