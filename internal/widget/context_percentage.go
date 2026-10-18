package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

// percentageExtractor extracts a percentage value from status Data.
// Returns (value, ok) where ok=false means no data available (widget hidden),
// and ok=true means the value is valid even if 0 (widget shown as "0%").
type percentageExtractor func(data *status.Session) (float64, bool)

// percentageWidget is a generic widget that displays a formatted percentage.
type percentageWidget struct {
	extract       percentageExtractor
	displayName   string
	description   string
	defaultPrefix string
	defaultSuffix string
	defaultColor  string
}

func (w *percentageWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	pct, ok := w.Value(item, ctx)
	if !ok {
		return ""
	}
	return formatPercentage(item, pct)
}

// Value returns the rendered percentage, for threshold colors.
func (w *percentageWidget) Value(_ *config.WidgetItem, ctx RenderContext) (float64, bool) {
	if ctx.Data == nil {
		return 0, false
	}
	return w.extract(ctx.Data)
}

func (w *percentageWidget) displayModes() []string { return []string{"percent", "bar"} }

// formatPercentage renders pct as "45%", "45.0" in raw mode, or a block bar
// such as "▓▓▓▓░░░░░░ 45%" with metadata.display "bar".
func formatPercentage(item *config.WidgetItem, pct float64) string {
	switch {
	case displayMode(item, "percent") == "bar":
		return formatBlockBar(pct, barWidthFor(item)) + fmt.Sprintf(" %.0f%%", pct)
	case item.RawValue:
		return fmt.Sprintf("%.1f", pct)
	default:
		return fmt.Sprintf("%.0f%%", pct)
	}
}

func (w *percentageWidget) DefaultColor() string {
	if w.defaultColor != "" {
		return w.defaultColor
	}
	return defaultDimColor
}
func (w *percentageWidget) DisplayName() string    { return w.displayName }
func (w *percentageWidget) Description() string    { return w.description }
func (w *percentageWidget) SupportsRawValue() bool { return true }
func (w *percentageWidget) DefaultPrefix() string  { return w.defaultPrefix }
func (w *percentageWidget) DefaultSuffix() string  { return w.defaultSuffix }
