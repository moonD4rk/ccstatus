package widget

import (
	"fmt"
	"strings"
	"time"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

const (
	msPerSecond = 1000
	msPerMinute = 60 * msPerSecond
	msPerHour   = 60 * msPerMinute
	msPerDay    = 24 * msPerHour
)

const (
	// DefaultBarWidth is the width of a bar display when an item sets no barWidth.
	DefaultBarWidth = 10
	// MaxBarWidth caps an item's barWidth: a bar wider than any terminal only
	// costs memory.
	MaxBarWidth = 100
)

// formatDuration renders a duration compactly: "<1m", "42m", "2h13m", and with
// days from 24h up ("4d15h"), where minutes are dropped.
func formatDuration(ms float64) string {
	totalMs := int(ms)
	days := totalMs / msPerDay
	hours := (totalMs % msPerDay) / msPerHour
	mins := (totalMs % msPerHour) / msPerMinute
	switch {
	case days > 0 && hours > 0:
		return fmt.Sprintf("%dd%dh", days, hours)
	case days > 0:
		return fmt.Sprintf("%dd", days)
	case hours == 0 && mins == 0:
		return "<1m"
	case hours == 0:
		return fmt.Sprintf("%dm", mins)
	case mins == 0:
		return fmt.Sprintf("%dh", hours)
	default:
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
}

// formatElapsed renders a running time with second precision below an hour,
// "45s" or "1m23s", and like every other duration from there on: "1h2m".
func formatElapsed(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return formatDuration(float64(d.Milliseconds()))
	}
}

// compactTokens formats a token budget, dropping a ".0" for whole thousands
// ("32k" rather than "32.0k").
func compactTokens(n int) string {
	if n >= 1000 && n < 1_000_000 && n%1000 == 0 {
		return fmt.Sprintf("%dk", n/1000)
	}
	return status.FormatTokens(n)
}

// formatPercentage renders pct as "45%", "45.0" in raw mode, or a block bar
// such as "▓▓▓▓░░░░░░ 45%" with display "bar".
func formatPercentage(item *config.WidgetItem, pct float64) string {
	switch {
	case displayMode(item, "percent") == BarDisplay:
		return formatBlockBar(pct, barWidthFor(item)) + fmt.Sprintf(" %.0f%%", pct)
	case item.RawValue:
		return fmt.Sprintf("%.1f", pct)
	default:
		return fmt.Sprintf("%.0f%%", pct)
	}
}

// clampPercent bounds a subscription window's usage to 0-100; Claude Code has
// briefly reported values above 100.
func clampPercent(pct float64) float64 {
	return min(max(pct, 0), 100)
}

// formatBlockBar renders a fixed-width bar like "▓▓▓░░░░░░░" for the given percentage.
func formatBlockBar(pct float64, width int) string {
	if width <= 0 {
		width = DefaultBarWidth
	}
	filled := int(pct*float64(width)/100 + 0.5)
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("▓", filled) + strings.Repeat("░", width-filled)
}

// barWidthFor returns the width of an item's bar: its barWidth, capped, or the
// default when it sets none.
func barWidthFor(item *config.WidgetItem) int {
	if item.BarWidth > 0 {
		return min(item.BarWidth, MaxBarWidth)
	}
	return DefaultBarWidth
}
