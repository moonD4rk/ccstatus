package widget

import (
	"fmt"
	"time"

	"github.com/moond4rk/ccstatus/internal/config"
)

const blockDuration = 5 * time.Hour

// BlockTimerWidget displays how far the rolling 5-hour rate-limit window has
// run. It is hidden without rate_limits data: on accounts with no subscription
// limits, and before the first API response.
// Display modes:
//   - "time" (default): elapsed over total, "2h15m/5h"
//   - "bar": a block progress bar, "▓▓▓▓▓░░░░░ 45%" (width via barWidth)
//   - "percent": "45%"
type BlockTimerWidget struct{}

// Render returns the block timer display.
func (w *BlockTimerWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	elapsed, ok := blockElapsed(ctx)
	if !ok {
		return ""
	}
	pct := blockPercent(elapsed)

	switch displayMode(item, "time") {
	case BarDisplay:
		return formatBlockBar(pct, barWidthFor(item)) + fmt.Sprintf(" %.0f%%", pct)
	case "percent":
		return fmt.Sprintf("%.0f%%", pct)
	default:
		return formatBlockTime(elapsed)
	}
}

// Value returns the share of the window that has run, 0-100, for threshold colors.
func (w *BlockTimerWidget) Value(_ *config.WidgetItem, ctx RenderContext) (float64, bool) {
	elapsed, ok := blockElapsed(ctx)
	if !ok {
		return 0, false
	}
	return blockPercent(elapsed), true
}

func blockPercent(elapsed time.Duration) float64 {
	return float64(elapsed) / float64(blockDuration) * 100
}

// blockElapsed returns the time since the 5-hour window opened. The window ends
// at rate_limits.five_hour.resets_at, so it opened blockDuration before that; a
// reset time already past means the window ran its full length. It reports
// false when the reset time is unknown or further away than a whole window.
func blockElapsed(ctx RenderContext) (time.Duration, bool) {
	if ctx.Data == nil || ctx.Data.RateLimits == nil || ctx.Data.RateLimits.FiveHour == nil ||
		ctx.Data.RateLimits.FiveHour.ResetsAt == nil {
		return 0, false
	}
	left := max(time.Unix(*ctx.Data.RateLimits.FiveHour.ResetsAt, 0).Sub(ctx.now()), 0)
	if left > blockDuration {
		return 0, false
	}
	return blockDuration - left, true
}

func formatBlockTime(elapsed time.Duration) string {
	hours := int(elapsed.Hours())
	mins := int(elapsed.Minutes()) % 60
	if hours == 0 && mins == 0 {
		return "<1m/5h"
	}
	if hours == 0 {
		return fmt.Sprintf("%dm/5h", mins)
	}
	if mins == 0 {
		return fmt.Sprintf("%dh/5h", hours)
	}
	return fmt.Sprintf("%dh%dm/5h", hours, mins)
}

func (w *BlockTimerWidget) displayModes() []string { return []string{"time", BarDisplay, "percent"} }

// DefaultColor returns the default foreground color.
func (w *BlockTimerWidget) DefaultColor() string { return defaultDimColor }

// Description returns what this widget shows.
func (w *BlockTimerWidget) Description() string { return "Time into the 5-hour rate-limit window" }

// DefaultPrefix returns the default prefix.
func (w *BlockTimerWidget) DefaultPrefix() string { return "Block: " }
