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
func (w *LinesChangedWidget) DefaultColor() string { return defaultGreenColor }

// DisplayName returns the human-readable name.
func (w *LinesChangedWidget) DisplayName() string { return "Lines Changed" }

// Description returns what this widget shows.
func (w *LinesChangedWidget) Description() string { return "Lines added and removed in session" }

// SupportsRawValue returns false.
func (w *LinesChangedWidget) SupportsRawValue() bool { return false }

// LinesAddedWidget displays only the lines added count.
type LinesAddedWidget struct{}

// Render returns "+N" format, or empty if zero.
func (w *LinesAddedWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data.Cost == nil || ctx.Data.Cost.TotalLinesAdded == nil {
		return ""
	}
	n := *ctx.Data.Cost.TotalLinesAdded
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("+%d", n)
}

// DefaultColor returns the default foreground color.
func (w *LinesAddedWidget) DefaultColor() string { return defaultGreenColor }

// DisplayName returns the human-readable name.
func (w *LinesAddedWidget) DisplayName() string { return "Lines Added" }

// Description returns what this widget shows.
func (w *LinesAddedWidget) Description() string { return "Lines added in session" }

// SupportsRawValue returns false.
func (w *LinesAddedWidget) SupportsRawValue() bool { return false }

// LinesRemovedWidget displays only the lines removed count.
type LinesRemovedWidget struct{}

// Render returns "-N" format, or empty if zero.
func (w *LinesRemovedWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data.Cost == nil || ctx.Data.Cost.TotalLinesRemoved == nil {
		return ""
	}
	n := *ctx.Data.Cost.TotalLinesRemoved
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("-%d", n)
}

// DefaultColor returns the default foreground color.
func (w *LinesRemovedWidget) DefaultColor() string { return "red" }

// DisplayName returns the human-readable name.
func (w *LinesRemovedWidget) DisplayName() string { return "Lines Removed" }

// Description returns what this widget shows.
func (w *LinesRemovedWidget) Description() string { return "Lines removed in session" }

// SupportsRawValue returns false.
func (w *LinesRemovedWidget) SupportsRawValue() bool { return false }
