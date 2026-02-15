package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
)

const costPrecisionThreshold = 0.01

// SessionCostWidget displays the session cost in USD.
type SessionCostWidget struct{}

// Render returns the session cost formatted as dollars.
func (w *SessionCostWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data.Cost == nil || ctx.Data.Cost.TotalCostUSD == nil {
		return ""
	}
	cost := *ctx.Data.Cost.TotalCostUSD
	if item.RawValue {
		return fmt.Sprintf("%.4f", cost)
	}
	return "$" + formatCost(cost)
}

func formatCost(cost float64) string {
	if cost < costPrecisionThreshold {
		return fmt.Sprintf("%.4f", cost)
	}
	return fmt.Sprintf("%.2f", cost)
}

// DefaultColor returns the default foreground color.
func (w *SessionCostWidget) DefaultColor() string { return "green" }

// DisplayName returns the human-readable name.
func (w *SessionCostWidget) DisplayName() string { return "Session Cost" }

// Description returns what this widget shows.
func (w *SessionCostWidget) Description() string { return "Session cost in USD" }

// SupportsRawValue returns true since this widget supports compact output.
func (w *SessionCostWidget) SupportsRawValue() bool { return true }
