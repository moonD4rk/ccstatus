package widget

import "github.com/moond4rk/ccstatus/internal/config"

// FlexSeparatorWidget is a placeholder that expands to fill remaining terminal width.
// The actual expansion is handled by the render pipeline.
type FlexSeparatorWidget struct{}

// Render returns a sentinel value; the render pipeline replaces it with spaces.
func (w *FlexSeparatorWidget) Render(_ *config.WidgetItem, _ RenderContext, _ *config.Settings) string {
	return TypeFlexSeparator
}

// DefaultColor returns empty since flex separators are invisible spacing.
func (w *FlexSeparatorWidget) DefaultColor() string { return "" }

// Description returns what this widget shows.
func (w *FlexSeparatorWidget) Description() string {
	return "Expands to fill remaining terminal width"
}
