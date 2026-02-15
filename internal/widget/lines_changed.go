package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/git"
)

// LinesChangedWidget displays git diff line additions and deletions.
type LinesChangedWidget struct{}

// Render returns a "+N/-M" format from git diff --shortstat, or empty if clean.
func (w *LinesChangedWidget) Render(_ *config.WidgetItem, _ RenderContext, _ *config.Settings) string {
	stat := git.GetDiffStat()
	if stat.Added == 0 && stat.Removed == 0 {
		return ""
	}
	return fmt.Sprintf("+%d/-%d", stat.Added, stat.Removed)
}

// DefaultColor returns the default foreground color.
func (w *LinesChangedWidget) DefaultColor() string { return defaultGreenColor }

// DisplayName returns the human-readable name.
func (w *LinesChangedWidget) DisplayName() string { return "Lines Changed" }

// Description returns what this widget shows.
func (w *LinesChangedWidget) Description() string {
	return "Uncommitted lines added and removed (git diff)"
}

// SupportsRawValue returns false.
func (w *LinesChangedWidget) SupportsRawValue() bool { return false }

// LinesAddedWidget displays only the git diff lines added count.
type LinesAddedWidget struct{}

// Render returns "+N" from git diff, or empty if zero.
func (w *LinesAddedWidget) Render(_ *config.WidgetItem, _ RenderContext, _ *config.Settings) string {
	stat := git.GetDiffStat()
	if stat.Added == 0 {
		return ""
	}
	return fmt.Sprintf("+%d", stat.Added)
}

// DefaultColor returns the default foreground color.
func (w *LinesAddedWidget) DefaultColor() string { return defaultGreenColor }

// DisplayName returns the human-readable name.
func (w *LinesAddedWidget) DisplayName() string { return "Lines Added" }

// Description returns what this widget shows.
func (w *LinesAddedWidget) Description() string { return "Uncommitted lines added (git diff)" }

// SupportsRawValue returns false.
func (w *LinesAddedWidget) SupportsRawValue() bool { return false }

// LinesRemovedWidget displays only the git diff lines removed count.
type LinesRemovedWidget struct{}

// Render returns "-N" from git diff, or empty if zero.
func (w *LinesRemovedWidget) Render(_ *config.WidgetItem, _ RenderContext, _ *config.Settings) string {
	stat := git.GetDiffStat()
	if stat.Removed == 0 {
		return ""
	}
	return fmt.Sprintf("-%d", stat.Removed)
}

// DefaultColor returns the default foreground color.
func (w *LinesRemovedWidget) DefaultColor() string { return "red" }

// DisplayName returns the human-readable name.
func (w *LinesRemovedWidget) DisplayName() string { return "Lines Removed" }

// Description returns what this widget shows.
func (w *LinesRemovedWidget) Description() string { return "Uncommitted lines removed (git diff)" }

// SupportsRawValue returns false.
func (w *LinesRemovedWidget) SupportsRawValue() bool { return false }
