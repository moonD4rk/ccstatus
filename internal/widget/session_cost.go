package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
)

const (
	costPrecisionThreshold = 0.01
	// costRoundsToZero is the cost below which four decimals read as zero.
	costRoundsToZero = 0.00005
)

// SessionCostWidget displays the session cost in USD.
type SessionCostWidget struct{}

// Render returns the session cost formatted as dollars.
func (w *SessionCostWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil || ctx.Data.Cost == nil || ctx.Data.Cost.TotalCostUSD == nil {
		return ""
	}
	cost := *ctx.Data.Cost.TotalCostUSD
	if item.RawValue {
		return fmt.Sprintf("%.4f", cost)
	}
	return "$" + formatCost(cost)
}

// Value returns the session cost in USD, for threshold colors.
func (w *SessionCostWidget) Value(_ *config.WidgetItem, ctx RenderContext) (float64, bool) {
	if ctx.Data == nil || ctx.Data.Cost == nil || ctx.Data.Cost.TotalCostUSD == nil {
		return 0, false
	}
	return *ctx.Data.Cost.TotalCostUSD, true
}

// formatCost renders dollars with two decimals, and with four below one cent so
// a session that has barely started still shows movement.
func formatCost(cost float64) string {
	if cost >= costRoundsToZero && cost < costPrecisionThreshold {
		return fmt.Sprintf("%.4f", cost)
	}
	return fmt.Sprintf("%.2f", cost)
}

// DefaultColor returns the default foreground color.
func (w *SessionCostWidget) DefaultColor() string { return defaultGreenColor }

// Description returns what this widget shows.
func (w *SessionCostWidget) Description() string { return "Session cost in USD" }

// SupportsRawValue returns true since this widget supports compact output.
func (w *SessionCostWidget) SupportsRawValue() bool { return true }
