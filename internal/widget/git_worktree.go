package widget

import (
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/git"
)

// GitWorktreeWidget displays the current git worktree name.
type GitWorktreeWidget struct{}

// Render returns the worktree name if in a linked worktree, empty otherwise.
func (w *GitWorktreeWidget) Render(_ *config.WidgetItem, _ RenderContext, _ *config.Settings) string {
	return git.GetWorktree()
}

// DefaultColor returns the default foreground color.
func (w *GitWorktreeWidget) DefaultColor() string { return "magenta" }

// DisplayName returns the human-readable name.
func (w *GitWorktreeWidget) DisplayName() string { return "Git Worktree" }

// Description returns what this widget shows.
func (w *GitWorktreeWidget) Description() string { return "Current git worktree name" }

// SupportsRawValue returns false since this widget has no compact mode.
func (w *GitWorktreeWidget) SupportsRawValue() bool { return false }
