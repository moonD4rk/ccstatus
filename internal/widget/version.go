package widget

import "github.com/moond4rk/ccstatus/internal/config"

// VersionWidget displays the Claude Code version.
type VersionWidget struct{}

// Render returns the Claude Code version string.
func (w *VersionWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil {
		return ""
	}
	return ctx.Data.Version
}

// DefaultColor returns the default foreground color.
func (w *VersionWidget) DefaultColor() string { return "brightBlack" }

// DisplayName returns the human-readable name.
func (w *VersionWidget) DisplayName() string { return "Version" }

// Description returns what this widget shows.
func (w *VersionWidget) Description() string { return "Claude Code version" }

// SupportsRawValue returns false since this widget has no compact mode.
func (w *VersionWidget) SupportsRawValue() bool { return false }
