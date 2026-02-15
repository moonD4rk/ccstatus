package widget

import (
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/git"
)

// GitBranchWidget displays the current git branch name.
type GitBranchWidget struct{}

// Render returns the current git branch, optionally prefixed with a character.
func (w *GitBranchWidget) Render(item *config.WidgetItem, _ RenderContext, _ *config.Settings) string {
	branch := git.Branch()
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

// DefaultColor returns the default foreground color.
func (w *GitBranchWidget) DefaultColor() string { return "magenta" }

// DisplayName returns the human-readable name.
func (w *GitBranchWidget) DisplayName() string { return "Git Branch" }

// Description returns what this widget shows.
func (w *GitBranchWidget) Description() string { return "Current git branch name" }

// SupportsRawValue returns true since this widget has a compact output mode.
func (w *GitBranchWidget) SupportsRawValue() bool { return true }
