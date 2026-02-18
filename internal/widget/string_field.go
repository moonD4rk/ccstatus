package widget

import (
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

// stringFieldExtractor extracts a string value from status Data.
type stringFieldExtractor func(data *status.Session) string

// stringFieldWidget is a generic widget that displays a single string field.
type stringFieldWidget struct {
	extract       stringFieldExtractor
	defaultColor  string
	displayName   string
	description   string
	defaultPrefix string
	defaultSuffix string
}

// Render returns the extracted string value, or empty if unavailable.
func (w *stringFieldWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil {
		return ""
	}
	return w.extract(ctx.Data)
}

// DefaultColor returns the default foreground color.
func (w *stringFieldWidget) DefaultColor() string { return w.defaultColor }

// DisplayName returns the human-readable name.
func (w *stringFieldWidget) DisplayName() string { return w.displayName }

// Description returns what this widget shows.
func (w *stringFieldWidget) Description() string { return w.description }

// SupportsRawValue returns false since string field widgets have no compact mode.
func (w *stringFieldWidget) SupportsRawValue() bool { return false }

// DefaultPrefix returns the default prefix for this widget.
func (w *stringFieldWidget) DefaultPrefix() string { return w.defaultPrefix }

// DefaultSuffix returns the default suffix for this widget.
func (w *stringFieldWidget) DefaultSuffix() string { return w.defaultSuffix }
