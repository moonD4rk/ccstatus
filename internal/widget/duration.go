package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

// durationExtractor returns a duration in milliseconds from the payload, and
// false when the payload does not carry it.
type durationExtractor func(data *status.Session) (float64, bool)

// durationWidget is a generic widget that displays a duration the payload
// reports in milliseconds, such as "2h13m" (raw: the milliseconds).
type durationWidget struct {
	extract       durationExtractor
	description   string
	defaultPrefix string
}

// Render returns the duration in human-readable format.
func (w *durationWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil {
		return ""
	}
	ms, ok := w.extract(ctx.Data)
	if !ok {
		return ""
	}
	if item.RawValue {
		return fmt.Sprintf("%.0f", ms)
	}
	return formatDuration(ms)
}

func (w *durationWidget) DefaultColor() string   { return defaultDimColor }
func (w *durationWidget) Description() string    { return w.description }
func (w *durationWidget) SupportsRawValue() bool { return true }
func (w *durationWidget) DefaultPrefix() string  { return w.defaultPrefix }

func sessionDuration(data *status.Session) (float64, bool) {
	if data.Cost == nil || data.Cost.TotalDurationMS == nil {
		return 0, false
	}
	return *data.Cost.TotalDurationMS, true
}

func apiDuration(data *status.Session) (float64, bool) {
	if data.Cost == nil || data.Cost.TotalAPIDurationMS == nil {
		return 0, false
	}
	return *data.Cost.TotalAPIDurationMS, true
}
