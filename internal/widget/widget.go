// Package widget defines the widget interface, registry, and render context.
package widget

import (
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

// RenderContext carries runtime data available to all widgets during rendering.
// Most token/context data comes directly from StatusJSON.ContextWindow (official API).
type RenderContext struct {
	Data          *status.StatusJSON
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
	"model":       &ModelWidget{},
	"version":     &VersionWidget{},
	"git-branch":  &GitBranchWidget{},
	"custom-text": &CustomTextWidget{},
	"separator":   &SeparatorWidget{},

	// Token metrics
	"tokens-input": &tokenWidget{
		extract: extractInputTokens, displayName: "Input Tokens", description: "Total input token count",
	},
	"tokens-output": &tokenWidget{
		extract: extractOutputTokens, displayName: "Output Tokens", description: "Total output token count",
	},
	"tokens-cached": &tokenWidget{
		extract: extractCachedTokens, displayName: "Cached Tokens", description: "Cached token count",
	},
	"tokens-total": &tokenWidget{
		extract: extractTotalTokens, displayName: "Total Tokens", description: "Total token count (input + output)",
	},

	// Context window
	"context-length":            &ContextLengthWidget{},
	"context-percentage":        &ContextPercentageWidget{},
	"context-percentage-usable": &ContextPercentageUsableWidget{},

	// Layout
	"flex-separator": &FlexSeparatorWidget{},
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
