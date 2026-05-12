package widget

import (
	"github.com/moond4rk/ccstatus/internal/config"
)

// GitWorktreeWidget displays the current git worktree name.
type GitWorktreeWidget struct{}

// Render returns the worktree name if in a linked worktree, empty otherwise.
// Source order: workspace.git_worktree (any linked worktree), worktree.name
// (--worktree sessions only), then a `git rev-parse --git-dir` heuristic.
func (w *GitWorktreeWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data != nil {
		if ctx.Data.Workspace != nil && ctx.Data.Workspace.GitWorktree != "" {
			return ctx.Data.Workspace.GitWorktree
		}
		if ctx.Data.Worktree != nil && ctx.Data.Worktree.Name != "" {
			return ctx.Data.Worktree.Name
		}
	}
	return ctx.Git.Worktree()
}

// DefaultColor returns the default foreground color.
func (w *GitWorktreeWidget) DefaultColor() string { return "magenta" }

// DisplayName returns the human-readable name.
func (w *GitWorktreeWidget) DisplayName() string { return "Git Worktree" }

// Description returns what this widget shows.
func (w *GitWorktreeWidget) Description() string { return "Current git worktree name" }

// SupportsRawValue returns false since this widget has no compact mode.
func (w *GitWorktreeWidget) SupportsRawValue() bool { return false }
