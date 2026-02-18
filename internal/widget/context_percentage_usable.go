package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

// ContextPercentageUsableWidget displays context usage as a percentage of the usable window (80% of max).
type ContextPercentageUsableWidget struct{}

// Render returns the usable context usage percentage string.
func (w *ContextPercentageUsableWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil {
		return ""
	}
	length := status.ContextLength(ctx.Data)
	if length == 0 {
		return ""
	}
	cfg := status.ContextConfig(ctx.Data)
	if cfg.UsableTokens == 0 {
		return ""
	}
	pct := float64(length) / float64(cfg.UsableTokens) * 100
	if pct > 100 {
		pct = 100
	}
	if item.RawValue {
		return fmt.Sprintf("%.1f", pct)
	}
	return fmt.Sprintf("%.0f%%", pct)
}

// DefaultColor returns the default foreground color.
func (w *ContextPercentageUsableWidget) DefaultColor() string { return defaultDimColor }

// DisplayName returns the human-readable name.
func (w *ContextPercentageUsableWidget) DisplayName() string { return "Context % Usable" }

// Description returns what this widget shows.
func (w *ContextPercentageUsableWidget) Description() string {
	return "Context usage as percentage of usable window (80% of max)"
}

// SupportsRawValue returns true; raw value omits the % suffix.
func (w *ContextPercentageUsableWidget) SupportsRawValue() bool { return true }

// DefaultPrefix returns the default prefix.
func (w *ContextPercentageUsableWidget) DefaultPrefix() string { return "Usable: " }

// DefaultSuffix returns the default suffix.
func (w *ContextPercentageUsableWidget) DefaultSuffix() string { return "" }
