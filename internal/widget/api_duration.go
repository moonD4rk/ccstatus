package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
)

// APIDurationWidget displays the total API response time.
type APIDurationWidget struct{}

// Render returns the API duration in human-readable format.
func (w *APIDurationWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil || ctx.Data.Cost == nil || ctx.Data.Cost.TotalAPIDurationMS == nil {
		return ""
	}
	ms := *ctx.Data.Cost.TotalAPIDurationMS
	if item.RawValue {
		return fmt.Sprintf("%.0f", ms)
	}
	return formatDuration(ms)
}

// DefaultColor returns the default foreground color.
func (w *APIDurationWidget) DefaultColor() string { return defaultDimColor }

// DisplayName returns the human-readable name.
func (w *APIDurationWidget) DisplayName() string { return "API Duration" }

// Description returns what this widget shows.
func (w *APIDurationWidget) Description() string { return "Total API response time" }

// SupportsRawValue returns true since this widget supports raw ms output.
func (w *APIDurationWidget) SupportsRawValue() bool { return true }

// DefaultPrefix returns the default prefix.
func (w *APIDurationWidget) DefaultPrefix() string { return "API: " }

// DefaultSuffix returns the default suffix.
func (w *APIDurationWidget) DefaultSuffix() string { return "" }
