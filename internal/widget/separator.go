package widget

import "github.com/moond4rk/ccstatus/internal/config"

// SeparatorWidget displays a visual separator between other widgets.
type SeparatorWidget struct{}

// Render returns the separator character from the widget config or settings default.
func (w *SeparatorWidget) Render(item *config.WidgetItem, _ RenderContext, settings *config.Settings) string {
	if item.Character != "" {
		return item.Character
	}
	if settings.DefaultSeparator != "" {
		return settings.DefaultSeparator
	}
	return "|"
}

// DefaultColor returns the default foreground color.
func (w *SeparatorWidget) DefaultColor() string { return "brightBlack" }

// DisplayName returns the human-readable name.
func (w *SeparatorWidget) DisplayName() string { return "Separator" }

// Description returns what this widget shows.
func (w *SeparatorWidget) Description() string { return "Visual separator between widgets" }

// SupportsRawValue returns false since this widget has no compact mode.
func (w *SeparatorWidget) SupportsRawValue() bool { return false }
