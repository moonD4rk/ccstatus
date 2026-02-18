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
	if ctx.Data == nil {
		return ""
	}
	pct, ok := w.extract(ctx.Data)
	if !ok {
		return ""
	}
	if item.RawValue {
		return fmt.Sprintf("%.1f", pct)
	}
	return fmt.Sprintf("%.0f%%", pct)
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
