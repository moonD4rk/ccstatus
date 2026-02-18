package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
)

const (
	msPerSecond = 1000
	msPerMinute = 60 * msPerSecond
	msPerHour   = 60 * msPerMinute
)

// SessionClockWidget displays the session duration.
type SessionClockWidget struct{}

// Render returns the session duration in human-readable format.
func (w *SessionClockWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data.Cost == nil || ctx.Data.Cost.TotalDurationMS == nil {
		return ""
	}
	ms := *ctx.Data.Cost.TotalDurationMS
	if item.RawValue {
		return fmt.Sprintf("%.0f", ms)
	}
	return formatDuration(ms)
}

func formatDuration(ms float64) string {
	totalMs := int(ms)
	hours := totalMs / msPerHour
	mins := (totalMs % msPerHour) / msPerMinute
	if hours == 0 && mins == 0 {
		return "<1m"
	}
	if hours == 0 {
		return fmt.Sprintf("%dm", mins)
	}
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}

// DefaultColor returns the default foreground color.
func (w *SessionClockWidget) DefaultColor() string { return defaultDimColor }

// DisplayName returns the human-readable name.
func (w *SessionClockWidget) DisplayName() string { return "Session Clock" }

// Description returns what this widget shows.
func (w *SessionClockWidget) Description() string { return "Session duration" }

// SupportsRawValue returns true since this widget supports raw ms output.
func (w *SessionClockWidget) SupportsRawValue() bool { return true }

// DefaultPrefix returns the default prefix.
func (w *SessionClockWidget) DefaultPrefix() string { return "Session: " }

// DefaultSuffix returns the default suffix.
func (w *SessionClockWidget) DefaultSuffix() string { return "" }
