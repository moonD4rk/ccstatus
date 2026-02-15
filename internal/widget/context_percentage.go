package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

// percentageExtractor extracts a percentage value from status Data.
type percentageExtractor func(data *status.Session) float64

// percentageWidget is a generic widget that displays a formatted percentage.
type percentageWidget struct {
	extract     percentageExtractor
	displayName string
	description string
}

func (w *percentageWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil {
		return ""
	}
	pct := w.extract(ctx.Data)
	if pct == 0 {
		return ""
	}
	if item.RawValue {
		return fmt.Sprintf("%.1f", pct)
	}
	return fmt.Sprintf("%.0f%%", pct)
}

func (w *percentageWidget) DefaultColor() string   { return defaultDimColor }
func (w *percentageWidget) DisplayName() string    { return w.displayName }
func (w *percentageWidget) Description() string    { return w.description }
func (w *percentageWidget) SupportsRawValue() bool { return true }
