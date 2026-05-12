package widget

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/moond4rk/ccstatus/internal/config"
)

// AddedDirsWidget displays directories added via /add-dir or --add-dir.
// Default: "+N" (the count). With metadata["display"]="list", shows the joined
// basenames. Renders empty when no directories have been added.
type AddedDirsWidget struct{}

// Render returns the added-directories display, or empty if none.
func (w *AddedDirsWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil || ctx.Data.Workspace == nil || len(ctx.Data.Workspace.AddedDirs) == 0 {
		return ""
	}
	dirs := ctx.Data.Workspace.AddedDirs
	if item.Metadata != nil && item.Metadata["display"] == "list" {
		names := make([]string, len(dirs))
		for i, d := range dirs {
			names[i] = filepath.Base(d)
		}
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("+%d", len(dirs))
}

// DefaultColor returns the default foreground color.
func (w *AddedDirsWidget) DefaultColor() string { return "blue" }

// DisplayName returns the human-readable name.
func (w *AddedDirsWidget) DisplayName() string { return "Added Dirs" }

// Description returns what this widget shows.
func (w *AddedDirsWidget) Description() string { return "Directories added via /add-dir" }

// SupportsRawValue returns false since this widget has no compact mode.
func (w *AddedDirsWidget) SupportsRawValue() bool { return false }
