package widget

import "github.com/moond4rk/ccstatus/internal/config"

// CustomTextWidget displays user-defined static text.
type CustomTextWidget struct{}

// Render returns the custom text from the widget configuration.
func (w *CustomTextWidget) Render(item *config.WidgetItem, _ RenderContext, _ *config.Settings) string {
	return item.CustomText
}

// DefaultColor returns the default foreground color.
func (w *CustomTextWidget) DefaultColor() string { return "white" }

// DisplayName returns the human-readable name.
func (w *CustomTextWidget) DisplayName() string { return "Custom Text" }

// Description returns what this widget shows.
func (w *CustomTextWidget) Description() string { return "User-defined static text" }

// SupportsRawValue returns false since this widget has no compact mode.
func (w *CustomTextWidget) SupportsRawValue() bool { return false }
