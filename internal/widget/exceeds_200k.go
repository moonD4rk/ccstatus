package widget

import "github.com/moond4rk/ccstatus/internal/config"

// Exceeds200KWidget displays a warning when token count exceeds the 200k threshold.
type Exceeds200KWidget struct{}

// Render returns ">200k" if the token count exceeds 200k, empty otherwise.
func (w *Exceeds200KWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil || ctx.Data.Exceeds200K == nil || !*ctx.Data.Exceeds200K {
		return ""
	}
	return ">200k"
}

// DefaultColor returns the default foreground color.
func (w *Exceeds200KWidget) DefaultColor() string { return "red" }

// DisplayName returns the human-readable name.
func (w *Exceeds200KWidget) DisplayName() string { return "Exceeds 200k" }

// Description returns what this widget shows.
func (w *Exceeds200KWidget) Description() string {
	return "Warning when tokens exceed 200k threshold"
}

// SupportsRawValue returns false since this widget has no compact mode.
func (w *Exceeds200KWidget) SupportsRawValue() bool { return false }
