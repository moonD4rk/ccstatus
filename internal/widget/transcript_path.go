package widget

import (
	"path/filepath"

	"github.com/moond4rk/ccstatus/internal/config"
)

// TranscriptPathWidget displays the transcript file path.
type TranscriptPathWidget struct{}

// Render returns the transcript path.
// RawValue mode returns the full path with ~ substitution; normal mode returns the base name.
func (w *TranscriptPathWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil || ctx.Data.TranscriptPath == "" {
		return ""
	}
	if item.RawValue {
		return shortenHome(ctx.Data.TranscriptPath)
	}
	return filepath.Base(ctx.Data.TranscriptPath)
}

// DefaultColor returns the default foreground color.
func (w *TranscriptPathWidget) DefaultColor() string { return defaultDimColor }

// DisplayName returns the human-readable name.
func (w *TranscriptPathWidget) DisplayName() string { return "Transcript Path" }

// Description returns what this widget shows.
func (w *TranscriptPathWidget) Description() string { return "Transcript file path" }

// SupportsRawValue returns true since this widget supports full path mode.
func (w *TranscriptPathWidget) SupportsRawValue() bool { return true }
