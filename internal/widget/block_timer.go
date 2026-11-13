package widget

import (
	"fmt"
	"strings"
	"time"

	"github.com/moond4rk/ccstatus/internal/config"
)

const (
	blockDuration = 5 * time.Hour
	barWidth      = 10
)

// BlockTimerWidget displays how far the rolling 5-hour rate-limit window has
// run. It is hidden without rate_limits data: on accounts with no subscription
// limits, and before the first API response.
// Supports three display modes via metadata["display"]:
//   - "time" (default): shows elapsed/total like "2h15m/5h"
//   - "progress": shows progress bar "[=====>    ] 45%"
//   - "percentage": shows just "45%"
type BlockTimerWidget struct{}

// Render returns the block timer display.
func (w *BlockTimerWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	elapsed, ok := blockElapsed(ctx)
	if !ok {
		return ""
	}
	pct := float64(elapsed) / float64(blockDuration) * 100

	switch displayMode(item, "time") {
	case "progress":
		return formatProgressBar(pct)
	case "percentage":
		return fmt.Sprintf("%.0f%%", pct)
	default:
		return formatBlockTime(elapsed)
	}
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

func formatProgressBar(pct float64) string {
	filled := min(int(pct*barWidth/100), barWidth)
	empty := barWidth - filled
	var b strings.Builder
	b.WriteByte('[')
	for range filled {
		b.WriteByte('=')
	}
	if empty > 0 {
		b.WriteByte('>')
		for range empty - 1 {
			b.WriteByte(' ')
		}
	}
	b.WriteByte(']')
	fmt.Fprintf(&b, " %.0f%%", pct)
	return b.String()
}

func (w *BlockTimerWidget) displayModes() []string { return []string{"time", "progress", "percentage"} }

// DefaultColor returns the default foreground color.
func (w *BlockTimerWidget) DefaultColor() string { return defaultDimColor }

// Description returns what this widget shows.
func (w *BlockTimerWidget) Description() string { return "Time into the 5-hour rate-limit window" }

// DefaultPrefix returns the default prefix.
func (w *BlockTimerWidget) DefaultPrefix() string { return "Block: " }
