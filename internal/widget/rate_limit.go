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

	mode := "percent"
	if item.Metadata != nil {
		if m, ok := item.Metadata["display"]; ok {
			mode = m
		}
	}

	switch mode {
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
	if raw {
		return fmt.Sprintf("%.1f", *window.UsedPercentage)
	}
	return fmt.Sprintf("%.0f%%", *window.UsedPercentage)
}

func rateLimitBar(window *status.RateLimitWindow, width int) string {
	if window.UsedPercentage == nil {
		return ""
	}
	pct := *window.UsedPercentage
	return formatBlockBar(pct, width) + fmt.Sprintf(" %.0f%%", pct)
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

// RateLimitsWidget displays both Claude.ai rate-limit windows compactly, e.g.
// "5h: 3% / 7d: 12%". Each window is shown only when it has a used_percentage;
// the widget renders empty (and so is omitted, prefix included) when neither
// window has data — non-Pro/Max accounts, or before the first API response.
type RateLimitsWidget struct{}

// Render returns "5h: x% / 7d: y%" for whichever windows have data.
func (w *RateLimitsWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil || ctx.Data.RateLimits == nil {
		return ""
	}
	var parts []string
	if win := ctx.Data.RateLimits.FiveHour; win != nil && win.UsedPercentage != nil {
		parts = append(parts, fmt.Sprintf("5h: %.0f%%", *win.UsedPercentage))
	}
	if win := ctx.Data.RateLimits.SevenDay; win != nil && win.UsedPercentage != nil {
		parts = append(parts, fmt.Sprintf("7d: %.0f%%", *win.UsedPercentage))
	}
	return strings.Join(parts, " / ")
}

// DefaultColor returns the default foreground color.
func (w *RateLimitsWidget) DefaultColor() string { return "yellow" }

// DisplayName returns the human-readable name.
func (w *RateLimitsWidget) DisplayName() string { return "Rate Limits" }

// Description returns what this widget shows.
func (w *RateLimitsWidget) Description() string { return "Combined 5h/7d rate-limit usage" }

// SupportsRawValue returns false since this widget has no compact mode.
func (w *RateLimitsWidget) SupportsRawValue() bool { return false }

// DefaultPrefix returns the default prefix for this widget.
func (w *RateLimitsWidget) DefaultPrefix() string { return "Limit " }

// DefaultSuffix returns the default suffix for this widget.
func (w *RateLimitsWidget) DefaultSuffix() string { return "" }
