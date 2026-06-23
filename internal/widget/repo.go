package widget

import (
	"github.com/moond4rk/ccstatus/internal/config"
)

// RepoWidget displays the repository identity from workspace.repo (parsed by
// Claude Code from the origin remote).
type RepoWidget struct{ noAffix }

// Render returns "owner/name" (or just the name in raw mode, or when no owner is
// present), and empty when no repo identity is available.
func (w *RepoWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil || ctx.Data.Workspace == nil || ctx.Data.Workspace.Repo == nil {
		return ""
	}
	repo := ctx.Data.Workspace.Repo
	if repo.Name == "" {
		return ""
	}
	if item.RawValue || repo.Owner == "" {
		return repo.Name
	}
	return repo.Owner + "/" + repo.Name
}

// DefaultColor returns the default foreground color.
func (w *RepoWidget) DefaultColor() string { return "blue" }

// DisplayName returns the human-readable name.
func (w *RepoWidget) DisplayName() string { return "Repository" }

// Description returns what this widget shows.
func (w *RepoWidget) Description() string { return "Repository owner/name from the origin remote" }

// SupportsRawValue returns true (raw mode drops the owner prefix).
func (w *RepoWidget) SupportsRawValue() bool { return true }
