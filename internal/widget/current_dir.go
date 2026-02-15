package widget

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/moond4rk/ccstatus/internal/config"
)

// CurrentDirWidget displays the current working directory.
type CurrentDirWidget struct{}

// Render returns the current directory from the workspace or cwd field.
// RawValue mode returns the full path with ~ substitution; normal mode returns the base name.
func (w *CurrentDirWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	dir := ""
	if ctx.Data.Workspace != nil && ctx.Data.Workspace.CurrentDir != "" {
		dir = ctx.Data.Workspace.CurrentDir
	} else if ctx.Data.Cwd != "" {
		dir = ctx.Data.Cwd
	}
	if dir == "" {
		return ""
	}
	if item.RawValue {
		return shortenHome(dir)
	}
	return filepath.Base(dir)
}

// shortenHome replaces the home directory prefix with ~.
func shortenHome(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+"/") {
		return "~" + path[len(home):]
	}
	return path
}

// DefaultColor returns the default foreground color.
func (w *CurrentDirWidget) DefaultColor() string { return "blue" }

// DisplayName returns the human-readable name.
func (w *CurrentDirWidget) DisplayName() string { return "Current Directory" }

// Description returns what this widget shows.
func (w *CurrentDirWidget) Description() string { return "Current working directory" }

// SupportsRawValue returns true since this widget supports full path mode.
func (w *CurrentDirWidget) SupportsRawValue() bool { return true }
