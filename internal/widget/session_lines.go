package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
)

// SessionLinesWidget displays the lines Claude added and removed this session
// (cost.total_lines_added / total_lines_removed). The lines-* widgets show the
// uncommitted git diff instead.
type SessionLinesWidget struct{ noAffix }

// Render returns "+N/-M", or empty when Claude has changed no lines.
func (w *SessionLinesWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil || ctx.Data.Cost == nil {
		return ""
	}
	var added, removed int
	if p := ctx.Data.Cost.TotalLinesAdded; p != nil {
		added = *p
	}
	if p := ctx.Data.Cost.TotalLinesRemoved; p != nil {
		removed = *p
	}
	if added == 0 && removed == 0 {
		return ""
	}
	return fmt.Sprintf("+%d/-%d", added, removed)
}

// DefaultColor returns the default foreground color.
func (w *SessionLinesWidget) DefaultColor() string { return defaultGreenColor }

// DisplayName returns the human-readable name.
func (w *SessionLinesWidget) DisplayName() string { return "Session Lines" }

// Description returns what this widget shows.
func (w *SessionLinesWidget) Description() string {
	return "Lines Claude added and removed this session"
}

// SupportsRawValue returns false since this widget has no compact mode.
func (w *SessionLinesWidget) SupportsRawValue() bool { return false }
