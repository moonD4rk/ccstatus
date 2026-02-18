// Package widget defines the widget interface, registry, and render context.
package widget

import (
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

// RenderContext carries runtime data available to all widgets during rendering.
// Most token/context data comes directly from Data.ContextWindow (official API).
type RenderContext struct {
	Data          *status.Session
	TerminalWidth int
}

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

	// Token metrics
	"tokens-input": &tokenWidget{
		extract: extractInputTokens, displayName: "Input Tokens", description: "Total input token count",
		defaultPrefix: "In: ",
	},
	"tokens-output": &tokenWidget{
		extract: extractOutputTokens, displayName: "Output Tokens", description: "Total output token count",
		defaultPrefix: "Out: ",
	},
	"tokens-cached": &tokenWidget{
		extract: extractCachedTokens, displayName: "Cached Tokens", description: "Cached token count",
		defaultPrefix: "Cached: ",
	},
	"tokens-total": &tokenWidget{
		extract: extractTotalTokens, displayName: "Total Tokens", description: "Total token count (input + output)",
		defaultPrefix: "Total: ",
	},

	"current-usage-input": &tokenWidget{
		extract: extractCurrentInputTokens, displayName: "Current Input Tokens", description: "Current round input token count",
		defaultPrefix: "CurIn: ",
	},
	"current-usage-output": &tokenWidget{
		extract: extractCurrentOutputTokens, displayName: "Current Output Tokens", description: "Current round output token count",
		defaultPrefix: "CurOut: ",
	},
	"cache-creation": &tokenWidget{
		extract: extractCacheCreationTokens, displayName: "Cache Creation Tokens", description: "Cache creation input token count",
		defaultPrefix: "CacheW: ",
	},

	// Context window
	"context-length": &ContextLengthWidget{},
	"context-percentage": &percentageWidget{
		extract: status.ContextPercentage, displayName: "Context %", description: "Context usage as percentage of max window",
		defaultPrefix: "Ctx: ",
	},
	"context-percentage-usable": &ContextPercentageUsableWidget{},
	"remaining-percentage": &percentageWidget{
		extract: status.RemainingPercentage, displayName: "Remaining %", description: "Remaining context window percentage",
		defaultPrefix: "Rem: ",
	},
	"cache-hit-rate": &percentageWidget{
		extract: status.CacheHitRate, displayName: "Cache Hit Rate", description: "Cache read token ratio as percentage",
		defaultPrefix: "Cache: ", defaultColor: "cyan",
	},

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

	// Session info
	"session-id": &SessionIDWidget{},
	"output-style": &stringFieldWidget{
		extract: func(data *status.Session) string {
			if data.OutputStyle == nil {
				return ""
			}
			return data.OutputStyle.Name
		},
		defaultColor:  defaultDimColor,
		displayName:   "Output Style",
		description:   "Current output style name",
		defaultPrefix: "Style: ",
	},
	"vim-mode": &stringFieldWidget{
		extract: func(data *status.Session) string {
			if data.Vim == nil {
				return ""
			}
			return data.Vim.Mode
		},
		defaultColor:  "yellow",
		displayName:   "Vim Mode",
		description:   "Current vim mode indicator",
		defaultPrefix: "Vim: ",
	},
	"agent-name": &stringFieldWidget{
		extract: func(data *status.Session) string {
			if data.Agent == nil {
				return ""
			}
			return data.Agent.Name
		},
		defaultColor:  "cyan",
		displayName:   "Agent Name",
		description:   "Agent name when using --agent flag",
		defaultPrefix: "Agent: ",
	},
	"exceeds-200k":   &Exceeds200KWidget{},
	"terminal-width": &TerminalWidthWidget{},

	// User-defined
	"custom-text":    &CustomTextWidget{},
	"custom-command": &CustomCommandWidget{},

	// Layout
	"separator":      &SeparatorWidget{},
	"flex-separator": &FlexSeparatorWidget{},
}

// Prefixer is an optional interface for widgets that provide default prefix/suffix.
// User-configured values in WidgetItem.Prefix/Suffix take precedence over defaults.
type Prefixer interface {
	DefaultPrefix() string
	DefaultSuffix() string
}

// Get returns the widget for the given type string, or nil if unknown.
func Get(widgetType string) Widget {
	return registry[widgetType]
}

// Register adds a widget to the registry. Used by other packages to register
// widgets that are implemented outside the base widget package.
func Register(widgetType string, w Widget) {
	registry[widgetType] = w
}

// Types returns all registered widget type names.
func Types() []string {
	types := make([]string, 0, len(registry))
	for k := range registry {
		types = append(types, k)
	}
	return types
}
