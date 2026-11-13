package widget

import (
	"fmt"

	"github.com/moond4rk/ccstatus/internal/config"
)

// TerminalWidthWidget displays the current terminal width in columns.
type TerminalWidthWidget struct{}

// Render returns the terminal width as a string.
func (w *TerminalWidthWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.TerminalWidth <= 0 {
		return ""
	}
	return fmt.Sprintf("%d", ctx.TerminalWidth)
}

// DefaultColor returns the default foreground color.
func (w *TerminalWidthWidget) DefaultColor() string { return defaultDimColor }

// Description returns what this widget shows.
func (w *TerminalWidthWidget) Description() string {
	return "Line width in columns: the terminal width minus the flexMode margin"
}
