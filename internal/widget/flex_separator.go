package widget

import "github.com/moond4rk/ccstatus/internal/config"

const flexSeparatorType = "flex-separator"

// FlexSeparatorWidget is a placeholder that expands to fill remaining terminal width.
// The actual expansion is handled by the render pipeline.
type FlexSeparatorWidget struct{}

// Render returns a sentinel value; the render pipeline replaces it with spaces.
func (w *FlexSeparatorWidget) Render(_ *config.WidgetItem, _ RenderContext, _ *config.Settings) string {
	return flexSeparatorType
}

// DefaultColor returns empty since flex separators are invisible spacing.
func (w *FlexSeparatorWidget) DefaultColor() string { return "" }

// DisplayName returns the human-readable name.
func (w *FlexSeparatorWidget) DisplayName() string { return "Flex Separator" }

// Description returns what this widget shows.
func (w *FlexSeparatorWidget) Description() string {
	return "Expands to fill remaining terminal width"
}

// SupportsRawValue returns false.
func (w *FlexSeparatorWidget) SupportsRawValue() bool { return false }
