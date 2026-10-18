package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

// SpendLimitWidget displays the Claude apps gateway spend limit.
// Display modes via metadata["display"]:
//   - "amount" (default): "$314.12 / $500.00", or "63%" until the USD fields arrive
//   - "percent": "63%" (exceeds 100% once over the limit)
//   - "bar": "▓▓▓▓▓▓░░░░ 63%" (width via metadata["barWidth"])
//   - "reset": time until the limit's period resets, e.g. "12d4h"
//   - "full": "$314.12 / $500.00 monthly, resets 12d4h"
//
// Hidden when no spend limit applies.
type SpendLimitWidget struct{}

// Render returns the spend limit display for the configured mode.
func (w *SpendLimitWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	sl := spendLimit(ctx)
	if sl == nil {
		return ""
	}
	switch displayMode(item, "amount") {
	case "percent":
		return spendPercent(sl, item.RawValue)
	case "bar":
		if sl.UsedPercentage == nil {
			return ""
		}
		pct := *sl.UsedPercentage
		return formatBlockBar(pct, barWidthFor(item)) + fmt.Sprintf(" %.0f%%", pct)
	case "reset":
		return rateLimitTimeToReset(&sl.RateLimitWindow)
	case "full":
		out := spendAmount(sl, " / ")
		if out == "" {
			return ""
		}
		if sl.Period != "" {
			out += " " + sl.Period
		}
		if reset := rateLimitTimeToReset(&sl.RateLimitWindow); reset != "" {
			out += ", resets " + reset
		}
		return out
	default:
		return spendAmount(sl, " / ")
	}
}

// Value returns the used percentage, unclamped, for threshold colors.
func (w *SpendLimitWidget) Value(_ *config.WidgetItem, ctx RenderContext) (float64, bool) {
	sl := spendLimit(ctx)
	if sl == nil || sl.UsedPercentage == nil {
		return 0, false
	}
	return *sl.UsedPercentage, true
}

func spendLimit(ctx RenderContext) *status.SpendLimit {
	if ctx.Data == nil || ctx.Data.RateLimits == nil {
		return nil
	}
	return ctx.Data.RateLimits.SpendLimit
}

// spendAmount returns "$used<sep>$limit" when both dollar amounts are known and
// falls back to the percentage, which Claude Code sends first.
func spendAmount(sl *status.SpendLimit, sep string) string {
	if sl.UsedUSD != nil && sl.LimitUSD != nil {
		return fmt.Sprintf("$%.2f%s$%.2f", *sl.UsedUSD, sep, *sl.LimitUSD)
	}
	return spendPercent(sl, false)
}

func spendPercent(sl *status.SpendLimit, raw bool) string {
	if sl.UsedPercentage == nil {
		return ""
	}
	if raw {
		return fmt.Sprintf("%.1f", *sl.UsedPercentage)
	}
	return fmt.Sprintf("%.0f%%", *sl.UsedPercentage)
}

func (w *SpendLimitWidget) displayModes() []string {
	return []string{"amount", "percent", "bar", "reset", "full"}
}

// DefaultColor returns the default foreground color.
func (w *SpendLimitWidget) DefaultColor() string { return "yellow" }

// DisplayName returns the human-readable name.
func (w *SpendLimitWidget) DisplayName() string { return "Spend Limit" }

// Description returns what this widget shows.
func (w *SpendLimitWidget) Description() string {
	return "Claude apps gateway spend against its limit"
}

// SupportsRawValue returns true; raw mode drops the % in "percent" mode.
func (w *SpendLimitWidget) SupportsRawValue() bool { return true }

// DefaultPrefix returns the default prefix.
func (w *SpendLimitWidget) DefaultPrefix() string { return "Spend: " }

// DefaultSuffix returns the default suffix.
func (w *SpendLimitWidget) DefaultSuffix() string { return "" }
