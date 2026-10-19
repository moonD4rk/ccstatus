# RFC 001: ccstatus Architecture and Design

Status: Completed
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
- 37 widget types (23 shared with TypeScript version + 14 ccstatus-exclusive)
- Multi-line status line rendering
- Flex separator (fills remaining terminal width)
- ANSI color support (16 named colors via fatih/color)
- JSONL transcript parsing (block timer only; token/context data now provided by Claude Code JSON)
- Git integration (branch, changes, worktree, diff stats)
- Claude Code settings.json integration (install/uninstall)
- Configuration via `~/.config/ccstatus/settings.json`
- CLI subcommands: `init`, `validate`, `install`, `uninstall`, `dump`, `widgets`

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
| Block timer | `cost.total_duration_ms` | JSONL parsing (fallback) |

## Architecture

### Directory Structure

```
ccstatus/
  cmd/
    ccstatus/
      main.go              # Entry point, root command, stdin rendering
      cmd_dump.go          # dump subcommand (debug JSON capture)
      cmd_init.go          # init subcommand (generate settings.json)
      cmd_install.go       # install/uninstall subcommands
      cmd_validate.go      # validate subcommand
      cmd_widgets.go       # widgets subcommand (list all widgets)
  internal/
    config/
      config.go            # Settings struct, load/save, defaults
      config_test.go
      widget.go            # WidgetItem struct, IsMerged, MergeNoPadding
    render/
      render.go            # Status line rendering pipeline
      render_test.go
      truncate.go          # Terminal width truncation
    widget/
      widget.go            # Widget interface, Prefixer interface, and registry
      widget_test.go
      model.go             # Model widget
      version.go           # Version widget
      session_id.go        # SessionID widget
      session_cost.go      # SessionCost widget
      session_clock.go     # SessionClock widget
      git_branch.go        # GitBranch widget
      git_changes.go       # GitChanges widget
      git_worktree.go      # GitWorktree widget
      tokens.go            # Generic tokenWidget (7 instances: input/output/cached/total/current-usage-input/current-usage-output/cache-creation)
      context_length.go    # ContextLength widget
      context_percentage.go        # Generic percentageWidget (context-percentage, remaining-percentage, cache-hit-rate)
      context_percentage_usable.go # ContextPercentageUsable widget
      block_timer.go       # BlockTimer widget (cost.total_duration_ms + JSONL fallback)
      api_duration.go      # APIDuration widget
      current_dir.go       # CurrentWorkingDir widget
      project_dir.go       # ProjectDir widget
      transcript_path.go   # TranscriptPath widget
      lines_changed.go     # LinesChanged, LinesAdded, LinesRemoved widgets
      terminal_width.go    # TerminalWidth widget
      custom_text.go       # CustomText widget
      custom_command.go    # CustomCommand widget
      string_field.go      # Generic stringFieldWidget (output-style, vim-mode, agent-name)
      exceeds_200k.go      # Exceeds200K widget
      separator.go         # Separator widget
      flex_separator.go    # FlexSeparator widget
    jsonl/
      reader.go            # JSONL transcript parsing (block-timer fallback)
      reader_test.go
    color/
      color.go             # ANSI color output via fatih/color (16 named colors)
      color_test.go
    git/
      git.go               # Git branch, changes, worktree
      changes.go           # Git changes count
      diff.go              # Git diff stats (lines added/removed)
      diff_test.go
      worktree.go          # Git worktree detection
    terminal/
      terminal.go          # Terminal width detection via golang.org/x/term
    status/
      status.go            # Session input struct, parsing
      status_test.go
      format.go            # Token formatting, context config, percentage functions
      format_test.go
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
    | 2. Parse Session (includes context_window, cost, vim, agent)
    | 3. Load Settings from ~/.config/ccstatus/settings.json
    | 4. Detect terminal width
    v
internal/render/render.go
    |
    | 5. For each line in settings.Lines:
    |    a. Render each widget via registry
    |    b. Apply prefix/suffix (including Prefixer defaults)
    |    c. Clean separators (remove empty widgets, trim edges, deduplicate)
    |    d. Apply colors (fg, bg, bold) using fatih/color
    |    e. Join with padding / expand flex separators
    |    f. Truncate to terminal width with "..."
    |    g. Replace spaces with non-breaking spaces (U+00A0)
    |    h. Prepend ANSI reset (\x1b[0m)
    v
stdout (ANSI colored status line, one line per fmt.Println)
```

## Data Structures

### Session (Input from Claude Code)

Based on the official Claude Code status line documentation
(https://code.claude.com/docs/en/statusline).

```go
// Session represents the JSON payload piped from Claude Code.
// All fields are optional and may be absent or null.
type Session struct {
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
        m.DisplayName = inferDisplayName(s)
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
const CurrentVersion = 4

// Settings represents the ccstatus configuration.
type Settings struct {
    Version                 int            `json:"version"`
    Lines                   [][]WidgetItem `json:"lines"`
    FlexMode                string         `json:"flexMode"`
    CompactThreshold        int            `json:"compactThreshold"`
    ColorLevel              int            `json:"colorLevel"`
    DefaultSeparator        string         `json:"defaultSeparator,omitempty"`
    DefaultPadding          string         `json:"defaultPadding,omitempty"`
    InheritSeparatorColors  bool           `json:"inheritSeparatorColors"`
    OverrideBackgroundColor string         `json:"overrideBackgroundColor,omitempty"`
    OverrideForegroundColor string         `json:"overrideForegroundColor,omitempty"`
    GlobalBold              bool           `json:"globalBold"`
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
                {ID: "17", Type: "current-working-dir", Color: "blue", RawValue: true},
                {ID: "18", Type: "flex-separator"},
                {ID: "19", Type: "session-clock", Color: "white"},
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
    Prefix          string            `json:"prefix,omitempty"`
    Suffix          string            `json:"suffix,omitempty"`
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

// IsMerged returns true if this widget should merge with the adjacent widget.
func (w *WidgetItem) IsMerged() bool { ... }

// MergeNoPadding returns true if merge mode is "no-padding".
func (w *WidgetItem) MergeNoPadding() bool { ... }
```

### RenderContext

```go
// RenderContext carries runtime data available to all widgets during rendering.
// All token/context data comes directly from Data.ContextWindow (official API).
type RenderContext struct {
    Data          *status.Session
    TerminalWidth int
}
```

## Widget Interface

```go
// Widget defines the contract for all status line widgets.
type Widget interface {
    // Render produces the widget text for the status line.
    // Returns empty string if the widget has nothing to display.
    Render(item *config.WidgetItem, ctx RenderContext, settings *config.Settings) string

    // DefaultColor returns the default foreground color name.
    DefaultColor() string

    // DisplayName returns the human-readable widget name.
    DisplayName() string

    // Description returns a short description of what the widget shows.
    Description() string

    // SupportsRawValue indicates if the widget has a compact output mode.
    SupportsRawValue() bool
}

// Prefixer is an optional interface for widgets that provide default prefix/suffix.
// User-configured values in WidgetItem.Prefix/Suffix take precedence over defaults.
type Prefixer interface {
    DefaultPrefix() string
    DefaultSuffix() string
}
```

### Widget Registry (37 total)

```go
var registry = map[string]Widget{
    // Model and session
    "model":         &ModelWidget{},
    "version":       &VersionWidget{},
    "session-cost":  &SessionCostWidget{},
    "session-clock": &SessionClockWidget{},

    // Git
    "git-branch":   &GitBranchWidget{},
    "git-changes":  &GitChangesWidget{},
    "git-worktree": &GitWorktreeWidget{},

    // Token metrics (generic tokenWidget with extractor functions)
    "tokens-input":          &tokenWidget{...},  // Total input tokens
    "tokens-output":         &tokenWidget{...},  // Total output tokens
    "tokens-cached":         &tokenWidget{...},  // Cached tokens
    "tokens-total":          &tokenWidget{...},  // Total tokens (input + output)
    "current-usage-input":   &tokenWidget{...},  // Current round input tokens
    "current-usage-output":  &tokenWidget{...},  // Current round output tokens
    "cache-creation":        &tokenWidget{...},  // Cache creation input tokens

    // Context window (generic percentageWidget with extractor functions)
    "context-length":              &ContextLengthWidget{},
    "context-percentage":          &percentageWidget{...},  // Context usage %
    "context-percentage-usable":   &ContextPercentageUsableWidget{},
    "remaining-percentage":        &percentageWidget{...},  // Remaining context %
    "cache-hit-rate":              &percentageWidget{...},  // Cache read ratio %

    // Environment
    "current-working-dir": &CurrentDirWidget{},
    "project-dir":         &ProjectDirWidget{},
    "transcript-path":     &TranscriptPathWidget{},
    "lines-changed":       &LinesChangedWidget{},
    "lines-added":         &LinesAddedWidget{},
    "lines-removed":       &LinesRemovedWidget{},

    // Cost and duration
    "api-duration": &APIDurationWidget{},
    "block-timer":  &BlockTimerWidget{},

    // Session info (generic stringFieldWidget with extractor functions)
    "session-id":    &SessionIDWidget{},
    "output-style":  &stringFieldWidget{...},  // Output style name
    "vim-mode":      &stringFieldWidget{...},  // Vim mode indicator
    "agent-name":    &stringFieldWidget{...},  // Agent name

    "exceeds-200k":   &Exceeds200KWidget{},
    "terminal-width": &TerminalWidthWidget{},

    // User-defined
    "custom-text":    &CustomTextWidget{},
    "custom-command": &CustomCommandWidget{},

    // Layout
    "separator":      &SeparatorWidget{},
    "flex-separator": &FlexSeparatorWidget{},
}

// Get returns the widget for the given type string, or nil if unknown.
func Get(widgetType string) Widget {
    return registry[widgetType]
}

// Register adds a widget to the registry.
func Register(widgetType string, w Widget) { ... }

// Types returns all registered widget type names.
func Types() []string { ... }
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
    defaultMaxTokens   = 200_000
    defaultUsableRatio = 80 // 80% of max
    longMaxTokens      = 1_000_000
)

// WindowLimits holds resolved context window size information.
type WindowLimits struct {
    MaxTokens    int
    UsableTokens int
}

// ContextConfig resolves context window size.
// Primary: use context_window.context_window_size from JSON input.
// Fallback: heuristic based on model ID (for older Claude Code versions).
func ContextConfig(data *Session) WindowLimits {
    // Primary: use official context_window_size if available
    if data.ContextWindow != nil && data.ContextWindow.ContextWindowSize != nil {
        size := *data.ContextWindow.ContextWindowSize
        return WindowLimits{
            MaxTokens:    size,
            UsableTokens: size * defaultUsableRatio / 100,
        }
    }

    // Fallback: model ID heuristic
    lower := strings.ToLower(data.Model.ID)
    if strings.Contains(lower, "claude-sonnet-4-5") && strings.Contains(lower, "[1m]") {
        return WindowLimits{
            MaxTokens:    longMaxTokens,
            UsableTokens: longMaxTokens * defaultUsableRatio / 100,
        }
    }
    return WindowLimits{
        MaxTokens:    defaultMaxTokens,
        UsableTokens: defaultMaxTokens * defaultUsableRatio / 100,
    }
}
```

### Context Percentage

Claude Code now provides `context_window.used_percentage` directly. The manual calculation
is kept as a fallback. Note: `used_percentage` is calculated from input tokens only
(input_tokens + cache_creation_input_tokens + cache_read_input_tokens), not output tokens.

```go
// ContextPercentage returns the context usage percentage.
// Primary: use pre-calculated used_percentage from JSON input.
// Fallback: calculate from current_usage tokens and context_window_size.
// Returns (value, ok) where ok=false means no data available.
func ContextPercentage(data *Session) (float64, bool) {
    if data.ContextWindow != nil && data.ContextWindow.UsedPercentage != nil {
        return *data.ContextWindow.UsedPercentage, true
    }
    // Fallback: manual calculation if used_percentage is null (early in session)
    if data.ContextWindow != nil && data.ContextWindow.CurrentUsage != nil {
        cu := data.ContextWindow.CurrentUsage
        contextLength := cu.InputTokens + cu.CacheCreationInputTokens + cu.CacheReadInputTokens
        cfg := ContextConfig(data)
        if cfg.MaxTokens == 0 {
            return 0, false
        }
        pct := float64(contextLength) / float64(cfg.MaxTokens) * 100
        if pct > 100 {
            return 100, true
        }
        return pct, true
    }
    return 0, false
}
```

### Cache Hit Rate

```go
// CacheHitRate returns the cache read ratio as a percentage.
// Formula: cache_read_input_tokens / (input_tokens + cache_creation_input_tokens + cache_read_input_tokens) * 100
// Returns (value, ok) where ok=false means no data available.
func CacheHitRate(data *Session) (float64, bool) { ... }
```

### JSONL Transcript Parsing

With the official Claude Code API now providing token metrics and session duration directly
in the JSON input, JSONL transcript parsing is only required for the **block-timer** widget
as a fallback when `cost.total_duration_ms` is unavailable.

The `internal/jsonl` package reads the first JSONL entry timestamp using `encoding/json`.

```go
// SessionStart reads the first entry from a JSONL transcript file and returns
// its timestamp. Returns zero time if the file cannot be read or parsed.
func SessionStart(path string) time.Time { ... }
```

### Rendering Pipeline

```go
// RenderLine renders a single line of widgets into an ANSI-colored string.
func RenderLine(items []config.WidgetItem, settings *config.Settings, ctx widget.RenderContext) string {
    // 1. Iterate items, render each widget via registry
    // 2. Apply prefix/suffix (including Prefixer interface defaults)
    // 3. Skip items that produce empty output
    // 4. Clean separators (remove empty, trim edges, deduplicate)
    // 5. Apply colors (foreground, background, bold) with override support
    // 6. Join with padding / expand flex separators
    // 7. Truncate to terminal width with "..." if needed
}

// PostProcess applies practical workarounds to a rendered line.
func PostProcess(line string) string {
    // 1. Skip line if no visible text after stripping ANSI codes
    // 2. Replace spaces with non-breaking spaces U+00A0 (VSCode workaround)
    // 3. Prepend ANSI reset \x1b[0m (Claude Code dim workaround)
}
```

### Terminal Width Detection

```go
// Width returns the terminal width in columns, or 0 if detection fails.
func Width() int {
    // Uses golang.org/x/term.GetSize(fd) on stdout
    // Returns 0 if detection fails
}
```

### FlexMode Width Calculation

```go
func CalculateFlexWidth(detected int, flexMode string, compactThreshold int, contextPct float64) int {
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

### Supported Colors (16 named)

8 basic: black, red, green, yellow, blue, magenta, cyan, white
8 bright: brightBlack, brightRed, brightGreen, brightYellow, brightBlue, brightMagenta, brightCyan, brightWhite

Colors are mapped to `fatih/color` attributes. Background colors are derived by adding
the standard ANSI foreground-to-background offset (+10).

### Color Levels

- 0: No colors (returns text unmodified)
- 1+: Apply colors via fatih/color (16 named colors)

### ANSI Code Generation

```go
// Apply wraps text with ANSI color codes based on the given color level.
// Returns unmodified text when color level is 0 or text is empty.
func Apply(text, fg, bg string, bold bool, level int) string {
    // Uses fatih/color to build ANSI escape sequences
    // fg: mapped from named color to fatih/color attribute
    // bg: fg attribute + fgToBGOffset (10)
    // bold: fatih/color Bold attribute
}

// StripANSI removes all ANSI escape sequences from a string.
func StripANSI(s string) string { ... }

// VisibleWidth returns the number of visible runes, ignoring ANSI codes.
func VisibleWidth(s string) int { ... }
```

## Claude Code Integration

### Config Directory Resolution

ccstatus reads/writes `~/.claude/settings.json` (or `$CLAUDE_CONFIG_DIR/settings.json`).

### Install/Uninstall

The `install` and `uninstall` subcommands manage the Claude Code integration:

```json
{
  "statusLine": {
    "type": "command",
    "command": "ccstatus",
    "padding": 0
  }
}
```

## Configuration Compatibility

ccstatus settings.json is a **subset** of the TypeScript ccstatusline format. The Go version ignores unknown fields (forward-compatible). Users migrating from ccstatusline can use the same settings file -- Powerline and TUI-only fields are silently ignored.

## Implementation Plan

All phases are completed. Each phase included corresponding testify unit tests.

### Phase 1: Core Engine (Completed)

1. `go mod tidy` - Added dependencies (cobra, fatih/color, x/term, testify)
2. `cmd/ccstatus/main.go` - cobra root command with subcommands
3. `internal/status/status.go` - Session parsing with ModelField, ContextWindow, VimInfo, AgentInfo
4. `internal/config/config.go` - Settings load/save/defaults
5. `internal/color/color.go` - ANSI color system based on fatih/color
6. `internal/widget/widget.go` - Widget interface, Prefixer interface, and registry
7. `internal/render/render.go` - Rendering pipeline (colors, separators, truncation, U+00A0, ANSI reset)
8. Simple widgets: model, version, git-branch, custom-text, separator

### Phase 2: Token & Context Widgets (Completed)

9. Token widgets: tokens-input, tokens-output, tokens-cached, tokens-total (via generic tokenWidget)
10. Context widgets: context-length, context-percentage, context-percentage-usable (via generic percentageWidget)
11. `internal/terminal/terminal.go` - Width detection
12. Flex separator expansion

### Phase 3: Remaining Widgets (Completed)

13. git-changes, git-worktree (from git commands)
14. session-clock (from `cost.total_duration_ms`), session-cost (from `cost.total_cost_usd`)
15. block-timer (primary: `cost.total_duration_ms`, fallback: JSONL transcript parsing)
16. output-style, session-id, terminal-width, current-working-dir (via generic stringFieldWidget)
17. custom-command (with timeout and preserveColors)
18. vim-mode, agent-name, exceeds-200k (from official Claude Code JSON fields)
19. current-usage-input, current-usage-output, cache-creation (per-round token widgets)
20. remaining-percentage, cache-hit-rate (via generic percentageWidget)
21. api-duration, project-dir, transcript-path, lines-changed/added/removed

### Phase 4: Integration & Polish (Completed)

22. CLI subcommands: init, validate, install, uninstall, dump, widgets
23. Multi-line rendering (2-line default layout)
24. Merge mode (widget combining with/without padding)

### Phase 5: Release (Completed)

25. Integration test with official JSON schema sample
26. README documentation

## Dependencies

```
require (
    github.com/fatih/color      // ANSI color output via named color attributes
    github.com/spf13/cobra      // CLI framework with subcommands
    golang.org/x/term           // Terminal width detection
)

require (
    github.com/stretchr/testify // Unit testing (test only)
)
```

## Resolved Questions

1. **`init` config**: Generates a full 2-line default config with model, context, tokens, cache, git, lines, cost, and session clock.
2. **Distribution**: `go install` as primary, GitHub Releases for binary downloads.
3. **custom-command**: Uses `sh -c` for shell feature compatibility.
