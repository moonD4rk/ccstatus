package widget

import (
	"path/filepath"

	"github.com/moond4rk/ccstatus/internal/config"
)

// ProjectDirWidget displays the project root directory.
type ProjectDirWidget struct{}

// Render returns the project directory from the workspace field.
// RawValue mode returns the full path with ~ substitution; normal mode returns the base name.
func (w *ProjectDirWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil || ctx.Data.Workspace == nil || ctx.Data.Workspace.ProjectDir == "" {
		return ""
	}
	dir := ctx.Data.Workspace.ProjectDir
	if item.RawValue {
		return shortenHome(dir)
	}
	return filepath.Base(dir)
}

// DefaultColor returns the default foreground color.
func (w *ProjectDirWidget) DefaultColor() string { return "blue" }

// DisplayName returns the human-readable name.
func (w *ProjectDirWidget) DisplayName() string { return "Project Directory" }

// Description returns what this widget shows.
func (w *ProjectDirWidget) Description() string { return "Project root directory" }

// SupportsRawValue returns true since this widget supports full path mode.
func (w *ProjectDirWidget) SupportsRawValue() bool { return true }
