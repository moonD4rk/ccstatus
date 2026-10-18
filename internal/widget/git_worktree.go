package widget

import (
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

// GitWorktreeWidget displays the current git worktree name.
type GitWorktreeWidget struct{ noAffix }

// Render returns the worktree name if in a linked worktree, empty otherwise.
// Source order: workspace.git_worktree (any linked worktree), worktree.name
// (--worktree sessions only), then `git rev-parse --absolute-git-dir` for
// Claude Code versions whose payload cannot be trusted to name the worktree.
func (w *GitWorktreeWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if name, ok := worktreeFromJSON(ctx.Data); ok {
		return name
	}
	return ctx.Git.Worktree()
}

func (w *GitWorktreeWidget) gitFacts(ctx RenderContext) []gitFact {
	if _, ok := worktreeFromJSON(ctx.Data); ok {
		return nil
	}
	return []gitFact{factWorktree}
}

// worktreeFromJSON returns the worktree name from the payload and whether the
// payload settles the question. Claude Code v2.1.98+ sets workspace.git_worktree
// whenever the directory is inside a linked worktree, so its absence means none.
func worktreeFromJSON(data *status.Session) (string, bool) {
	if data == nil {
		return "", false
	}
	if data.Workspace != nil && data.Workspace.GitWorktree != "" {
		return data.Workspace.GitWorktree, true
	}
	if data.Worktree != nil && data.Worktree.Name != "" {
		return data.Worktree.Name, true
	}
	return "", data.VersionAtLeast(2, 1, 98)
}

// DefaultColor returns the default foreground color.
func (w *GitWorktreeWidget) DefaultColor() string { return "magenta" }

// DisplayName returns the human-readable name.
func (w *GitWorktreeWidget) DisplayName() string { return "Git Worktree" }

// Description returns what this widget shows.
func (w *GitWorktreeWidget) Description() string { return "Current git worktree name" }

// SupportsRawValue returns false since this widget has no compact mode.
func (w *GitWorktreeWidget) SupportsRawValue() bool { return false }
