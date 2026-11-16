package widget

import (
	"strconv"

	"github.com/moond4rk/ccstatus/internal/config"
)

// GitChangesWidget displays the number of uncommitted changes.
type GitChangesWidget struct{}

// Render returns the uncommitted change count, or empty if there are none.
func (w *GitChangesWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	n := ctx.Git.Changes()
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func (w *GitChangesWidget) gitFacts(RenderContext) []gitFact { return []gitFact{factChanges} }

// DefaultColor returns the default foreground color.
func (w *GitChangesWidget) DefaultColor() string { return "yellow" }

// Description returns what this widget shows.
func (w *GitChangesWidget) Description() string { return "Uncommitted changes count" }
