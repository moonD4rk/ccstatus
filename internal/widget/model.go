package widget

import "github.com/moond4rk/ccstatus/internal/config"

// ModelWidget displays the current Claude model name.
type ModelWidget struct{}

// Render returns the model display name, or model ID if rawValue is set.
func (w *ModelWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil {
		return ""
	}
	if item.RawValue {
		return ctx.Data.Model.ID
	}
	if ctx.Data.Model.DisplayName != "" {
		return ctx.Data.Model.DisplayName
	}
	return ctx.Data.Model.ID
}

// DefaultColor returns the default foreground color for the model widget.
func (w *ModelWidget) DefaultColor() string { return "cyan" }

// DisplayName returns the human-readable name.
func (w *ModelWidget) DisplayName() string { return "Model" }

// Description returns what this widget shows.
func (w *ModelWidget) Description() string { return "Current Claude model name" }

// SupportsRawValue returns true since this widget has a compact output mode.
func (w *ModelWidget) SupportsRawValue() bool { return true }
