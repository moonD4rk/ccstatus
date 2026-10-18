package widget

import (
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

// ContextPercentageUsableWidget displays context usage as a percentage of the usable window (80% of max).
type ContextPercentageUsableWidget struct{}

// Render returns the usable context usage percentage string.
func (w *ContextPercentageUsableWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	pct, ok := w.Value(item, ctx)
	if !ok {
		return ""
	}
	return formatPercentage(item, pct)
}

// Value returns the usable-window percentage, capped at 100, for threshold colors.
func (w *ContextPercentageUsableWidget) Value(_ *config.WidgetItem, ctx RenderContext) (float64, bool) {
	if ctx.Data == nil {
		return 0, false
	}
	length := status.ContextLength(ctx.Data)
	if length == 0 {
		return 0, false
	}
	cfg := status.ContextConfig(ctx.Data)
	if cfg.UsableTokens == 0 {
		return 0, false
	}
	return min(float64(length)/float64(cfg.UsableTokens)*100, 100), true
}

func (w *ContextPercentageUsableWidget) displayModes() []string { return []string{"percent", "bar"} }

// DefaultColor returns the default foreground color.
func (w *ContextPercentageUsableWidget) DefaultColor() string { return defaultDimColor }

// DisplayName returns the human-readable name.
func (w *ContextPercentageUsableWidget) DisplayName() string { return "Context % Usable" }

// Description returns what this widget shows.
func (w *ContextPercentageUsableWidget) Description() string {
	return "Context usage as % of the usable window (heuristic: 80% of max, leaving auto-compact headroom)"
}

// SupportsRawValue returns true; raw value omits the % suffix.
func (w *ContextPercentageUsableWidget) SupportsRawValue() bool { return true }

// DefaultPrefix returns the default prefix.
func (w *ContextPercentageUsableWidget) DefaultPrefix() string { return "Usable: " }

// DefaultSuffix returns the default suffix.
func (w *ContextPercentageUsableWidget) DefaultSuffix() string { return "" }
