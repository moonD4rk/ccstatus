package widget

import (
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

// ContextLengthWidget displays the context window usage as a formatted token count.
type ContextLengthWidget struct{}

// Render returns the formatted context length.
func (w *ContextLengthWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil {
		return ""
	}
	length := status.ContextLength(ctx.Data)
	if length == 0 {
		return ""
	}
	return status.FormatTokens(length)
}

// DefaultColor returns the default foreground color.
func (w *ContextLengthWidget) DefaultColor() string { return defaultDimColor }

// DisplayName returns the human-readable name.
func (w *ContextLengthWidget) DisplayName() string { return "Context Length" }

// Description returns what this widget shows.
func (w *ContextLengthWidget) Description() string { return "Context window usage in tokens" }

// SupportsRawValue returns false since formatting is always applied.
func (w *ContextLengthWidget) SupportsRawValue() bool { return false }

// DefaultPrefix returns the default prefix.
func (w *ContextLengthWidget) DefaultPrefix() string { return "CtxLen: " }

// DefaultSuffix returns the default suffix.
func (w *ContextLengthWidget) DefaultSuffix() string { return "" }
