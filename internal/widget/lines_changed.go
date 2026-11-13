package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
)

// LinesChangedWidget displays git diff line additions and deletions.
type LinesChangedWidget struct{}

// Render returns a "+N/-M" format from git diff --shortstat, or empty if clean.
func (w *LinesChangedWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	stat := ctx.Git.Diff()
	if stat.Added == 0 && stat.Removed == 0 {
		return ""
	}
	return fmt.Sprintf("+%d/-%d", stat.Added, stat.Removed)
}

func (w *LinesChangedWidget) gitFacts(RenderContext) []gitFact { return []gitFact{factDiff} }

// DefaultColor returns the default foreground color.
func (w *LinesChangedWidget) DefaultColor() string { return defaultGreenColor }

// Description returns what this widget shows.
func (w *LinesChangedWidget) Description() string {
	return "Uncommitted lines added and removed (git diff)"
}

// LinesAddedWidget displays only the git diff lines added count.
type LinesAddedWidget struct{}

// Render returns "+N" from git diff, or empty if zero.
func (w *LinesAddedWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	stat := ctx.Git.Diff()
	if stat.Added == 0 {
		return ""
	}
	return fmt.Sprintf("+%d", stat.Added)
}

func (w *LinesAddedWidget) gitFacts(RenderContext) []gitFact { return []gitFact{factDiff} }

// DefaultColor returns the default foreground color.
func (w *LinesAddedWidget) DefaultColor() string { return defaultGreenColor }

// Description returns what this widget shows.
func (w *LinesAddedWidget) Description() string { return "Uncommitted lines added (git diff)" }

// LinesRemovedWidget displays only the git diff lines removed count.
type LinesRemovedWidget struct{}

// Render returns "-N" from git diff, or empty if zero.
func (w *LinesRemovedWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	stat := ctx.Git.Diff()
	if stat.Removed == 0 {
		return ""
	}
	return fmt.Sprintf("-%d", stat.Removed)
}

func (w *LinesRemovedWidget) gitFacts(RenderContext) []gitFact { return []gitFact{factDiff} }

// DefaultColor returns the default foreground color.
func (w *LinesRemovedWidget) DefaultColor() string { return "red" }

// Description returns what this widget shows.
func (w *LinesRemovedWidget) Description() string { return "Uncommitted lines removed (git diff)" }
