package widget

import (
	"github.com/moond4rk/ccstatus/internal/config"
)

// GitBranchWidget displays the current git branch name.
type GitBranchWidget struct{}

// Render returns the current git branch, optionally prefixed with a character.
func (w *GitBranchWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	branch := ctx.Git.Branch()
	if branch == "" {
		return ""
	}
	if item.RawValue {
		return branch
	}
	if item.Character != "" {
		return item.Character + " " + branch
	}
	return branch
}

func (w *GitBranchWidget) gitFacts(RenderContext) []gitFact { return []gitFact{factBranch} }

// DefaultColor returns the default foreground color.
func (w *GitBranchWidget) DefaultColor() string { return "magenta" }

// Description returns what this widget shows.
func (w *GitBranchWidget) Description() string { return "Current git branch name" }

// SupportsRawValue returns true since this widget has a compact output mode.
func (w *GitBranchWidget) SupportsRawValue() bool { return true }
