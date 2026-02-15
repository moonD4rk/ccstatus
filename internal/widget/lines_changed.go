package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
)

// LinesChangedWidget displays the total lines added and removed in the session.
type LinesChangedWidget struct{}

// Render returns a "+N/-M" format of lines changed, or empty if no data.
func (w *LinesChangedWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data.Cost == nil {
		return ""
	}
	added := 0
	removed := 0
	if ctx.Data.Cost.TotalLinesAdded != nil {
		added = *ctx.Data.Cost.TotalLinesAdded
	}
	if ctx.Data.Cost.TotalLinesRemoved != nil {
		removed = *ctx.Data.Cost.TotalLinesRemoved
	}
	if added == 0 && removed == 0 {
		return ""
	}
	return fmt.Sprintf("+%d/-%d", added, removed)
}

// DefaultColor returns the default foreground color.
func (w *LinesChangedWidget) DefaultColor() string { return "green" }

// DisplayName returns the human-readable name.
func (w *LinesChangedWidget) DisplayName() string { return "Lines Changed" }

// Description returns what this widget shows.
func (w *LinesChangedWidget) Description() string { return "Lines added and removed in session" }

// SupportsRawValue returns false.
func (w *LinesChangedWidget) SupportsRawValue() bool { return false }
