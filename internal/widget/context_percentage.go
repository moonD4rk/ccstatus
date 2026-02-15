package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

// ContextPercentageWidget displays the context usage as a percentage of the max context window.
type ContextPercentageWidget struct{}

// Render returns the context usage percentage string.
func (w *ContextPercentageWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil {
		return ""
	}
	pct := status.GetContextPercentage(ctx.Data)
	if pct == 0 {
		return ""
	}
	if item.RawValue {
		return fmt.Sprintf("%.1f", pct)
	}
	return fmt.Sprintf("%.0f%%", pct)
}

// DefaultColor returns the default foreground color.
func (w *ContextPercentageWidget) DefaultColor() string { return defaultDimColor }

// DisplayName returns the human-readable name.
func (w *ContextPercentageWidget) DisplayName() string { return "Context %" }

// Description returns what this widget shows.
func (w *ContextPercentageWidget) Description() string {
	return "Context usage as percentage of max window"
}

// SupportsRawValue returns true; raw value omits the % suffix.
func (w *ContextPercentageWidget) SupportsRawValue() bool { return true }
