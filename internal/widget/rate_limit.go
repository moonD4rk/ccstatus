package widget

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

const defaultBarWidth = 10

// rateLimitExtractor returns the rate-limit window a widget displays, or nil.
type rateLimitExtractor func(data *status.Session) *status.RateLimitWindow

// RateLimitWidget displays a Claude.ai subscription rate-limit window.
// Display modes via metadata["display"]:
//   - "percent" (default): "45%" from used_percentage
//   - "bar": a block progress bar, e.g. "▓▓▓▓░░░░░░ 45%" (width via metadata["barWidth"])
//   - "reset": time until the window resets, e.g. "2h13m"
//   - "full": "45% / 2h13m"
//
// Renders empty when the relevant data is absent (e.g. non-Pro/Max accounts,
// or before the first API response).
type RateLimitWidget struct {
	extract       rateLimitExtractor
	displayName   string
	description   string
	defaultPrefix string
}

// Render returns the rate-limit display for the configured window and mode.
func (w *RateLimitWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil {
		return ""
	}
	window := w.extract(ctx.Data)
	if window == nil {
		return ""
	}

	switch displayMode(item, "percent") {
	case "bar":
		return rateLimitBar(window, barWidthFor(item))
	case "reset":
		return rateLimitTimeToReset(window)
	case "full":
		pct := rateLimitPercent(window, item.RawValue)
		rst := rateLimitTimeToReset(window)
		switch {
		case pct == "":
			return rst
		case rst == "":
			return pct
		default:
			return pct + " / " + rst
		}
	default:
		return rateLimitPercent(window, item.RawValue)
	}
}

func rateLimitPercent(window *status.RateLimitWindow, raw bool) string {
	if window.UsedPercentage == nil {
		return ""
	}
	pct := clampPercent(*window.UsedPercentage)
	if raw {
		return fmt.Sprintf("%.1f", pct)
	}
	return fmt.Sprintf("%.0f%%", pct)
}

func rateLimitBar(window *status.RateLimitWindow, width int) string {
	if window.UsedPercentage == nil {
		return ""
	}
	pct := clampPercent(*window.UsedPercentage)
	return formatBlockBar(pct, width) + fmt.Sprintf(" %.0f%%", pct)
}

// clampPercent bounds a subscription window's usage to 0-100; Claude Code has
// briefly reported values above 100.
func clampPercent(pct float64) float64 {
	return min(max(pct, 0), 100)
}

// formatBlockBar renders a fixed-width bar like "▓▓▓░░░░░░░" for the given percentage.
func formatBlockBar(pct float64, width int) string {
	if width <= 0 {
		width = defaultBarWidth
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

func barWidthFor(item *config.WidgetItem) int {
	if item.Metadata == nil {
		return defaultBarWidth
	}
	if v, ok := item.Metadata["barWidth"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultBarWidth
}

func rateLimitTimeToReset(window *status.RateLimitWindow) string {
	if window.ResetsAt == nil {
		return ""
	}
	remaining := time.Until(time.Unix(*window.ResetsAt, 0))
	if remaining < 0 {
		remaining = 0
	}
	return formatDuration(float64(remaining.Milliseconds()))
}

func fiveHourWindow(data *status.Session) *status.RateLimitWindow {
	if data.RateLimits == nil {
		return nil
	}
	return data.RateLimits.FiveHour
}

func sevenDayWindow(data *status.Session) *status.RateLimitWindow {
	if data.RateLimits == nil {
		return nil
	}
	return data.RateLimits.SevenDay
}

func (w *RateLimitWidget) displayModes() []string { return []string{"percent", "bar", "reset", "full"} }

// DefaultColor returns the default foreground color.
func (w *RateLimitWidget) DefaultColor() string { return "yellow" }

// DisplayName returns the human-readable name.
func (w *RateLimitWidget) DisplayName() string { return w.displayName }

// Description returns what this widget shows.
func (w *RateLimitWidget) Description() string { return w.description }

// SupportsRawValue returns true since the percent value can be shown without "%".
func (w *RateLimitWidget) SupportsRawValue() bool { return true }

// DefaultPrefix returns the default prefix for this widget.
func (w *RateLimitWidget) DefaultPrefix() string { return w.defaultPrefix }

// DefaultSuffix returns the default suffix for this widget.
func (w *RateLimitWidget) DefaultSuffix() string { return "" }

// RateLimitsWidget displays the Claude.ai rate-limit windows and the gateway
// spend limit compactly, e.g. "5h: 3% / 7d: 12%" or "spend: $314.12/$500.00".
// Each limit is shown only when it has data; the widget renders empty (and so is
// omitted, prefix included) when none does — before the first API response, or
// for accounts with no limits.
type RateLimitsWidget struct{}

// Render returns "5h: x% / 7d: y% / spend: $u/$l" for whichever limits have data.
func (w *RateLimitsWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil || ctx.Data.RateLimits == nil {
		return ""
	}
	rl := ctx.Data.RateLimits
	var parts []string
	if win := rl.FiveHour; win != nil && win.UsedPercentage != nil {
		parts = append(parts, fmt.Sprintf("5h: %.0f%%", clampPercent(*win.UsedPercentage)))
	}
	if win := rl.SevenDay; win != nil && win.UsedPercentage != nil {
		parts = append(parts, fmt.Sprintf("7d: %.0f%%", clampPercent(*win.UsedPercentage)))
	}
	if rl.SpendLimit != nil {
		if spend := spendAmount(rl.SpendLimit, "/"); spend != "" {
			parts = append(parts, "spend: "+spend)
		}
	}
	return strings.Join(parts, " / ")
}

// Value returns the highest usage across the limits shown, for threshold colors.
func (w *RateLimitsWidget) Value(_ *config.WidgetItem, ctx RenderContext) (float64, bool) {
	if ctx.Data == nil || ctx.Data.RateLimits == nil {
		return 0, false
	}
	rl := ctx.Data.RateLimits
	best, found := 0.0, false
	consider := func(pct *float64, clamp bool) {
		if pct == nil {
			return
		}
		v := *pct
		if clamp {
			v = clampPercent(v)
		}
		if !found || v > best {
			best, found = v, true
		}
	}
	if rl.FiveHour != nil {
		consider(rl.FiveHour.UsedPercentage, true)
	}
	if rl.SevenDay != nil {
		consider(rl.SevenDay.UsedPercentage, true)
	}
	if rl.SpendLimit != nil {
		consider(rl.SpendLimit.UsedPercentage, false)
	}
	return best, found
}

// Value returns the window's used percentage, clamped to 0-100.
func (w *RateLimitWidget) Value(_ *config.WidgetItem, ctx RenderContext) (float64, bool) {
	if ctx.Data == nil {
		return 0, false
	}
	window := w.extract(ctx.Data)
	if window == nil || window.UsedPercentage == nil {
		return 0, false
	}
	return clampPercent(*window.UsedPercentage), true
}

// DefaultColor returns the default foreground color.
func (w *RateLimitsWidget) DefaultColor() string { return "yellow" }

// DisplayName returns the human-readable name.
func (w *RateLimitsWidget) DisplayName() string { return "Rate Limits" }

// Description returns what this widget shows.
func (w *RateLimitsWidget) Description() string {
	return "Combined 5h/7d rate-limit usage and gateway spend limit"
}

// SupportsRawValue returns false since this widget has no compact mode.
func (w *RateLimitsWidget) SupportsRawValue() bool { return false }

// DefaultPrefix returns the default prefix for this widget.
func (w *RateLimitsWidget) DefaultPrefix() string { return "Limit " }

// DefaultSuffix returns the default suffix for this widget.
func (w *RateLimitsWidget) DefaultSuffix() string { return "" }
