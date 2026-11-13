package widget

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/moond4rk/ccstatus/internal/config"
)

// AddedDirsWidget displays directories added via /add-dir or --add-dir.
// Default: "+N" (the count). With display "list", shows the joined basenames. Renders empty when no directories have been added.
type AddedDirsWidget struct{}

// Render returns the added-directories display, or empty if none.
func (w *AddedDirsWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil || ctx.Data.Workspace == nil || len(ctx.Data.Workspace.AddedDirs) == 0 {
		return ""
	}
	dirs := ctx.Data.Workspace.AddedDirs
	if displayMode(item, "count") == "list" {
		names := make([]string, len(dirs))
		for i, d := range dirs {
			names[i] = filepath.Base(d)
		}
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("+%d", len(dirs))
}

func (w *AddedDirsWidget) displayModes() []string { return []string{"count", "list"} }

// DefaultColor returns the default foreground color.
func (w *AddedDirsWidget) DefaultColor() string { return "blue" }

// Description returns what this widget shows.
func (w *AddedDirsWidget) Description() string { return "Directories added via /add-dir" }
