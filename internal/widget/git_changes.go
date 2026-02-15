package widget

import (
	"strconv"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/git"
)

// GitChangesWidget displays the number of uncommitted changes.
type GitChangesWidget struct{}

// Render returns the uncommitted change count, or empty if there are none.
func (w *GitChangesWidget) Render(_ *config.WidgetItem, _ RenderContext, _ *config.Settings) string {
	n := git.Changes()
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// DefaultColor returns the default foreground color.
func (w *GitChangesWidget) DefaultColor() string { return "yellow" }

// DisplayName returns the human-readable name.
func (w *GitChangesWidget) DisplayName() string { return "Git Changes" }

// Description returns what this widget shows.
func (w *GitChangesWidget) Description() string { return "Uncommitted changes count" }

// SupportsRawValue returns false.
func (w *GitChangesWidget) SupportsRawValue() bool { return false }
