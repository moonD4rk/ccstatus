package widget

import "github.com/moond4rk/ccstatus/internal/config"

const sessionIDShortLen = 8

// SessionIDWidget displays the Claude Code session ID.
type SessionIDWidget struct{}

// Render returns the session ID. Normal mode truncates to 8 characters;
// rawValue mode returns the full UUID.
func (w *SessionIDWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil || ctx.Data.SessionID == "" {
		return ""
	}
	if item.RawValue {
		return ctx.Data.SessionID
	}
	id := ctx.Data.SessionID
	if len(id) > sessionIDShortLen {
		return id[:sessionIDShortLen]
	}
	return id
}

// DefaultColor returns the default foreground color.
func (w *SessionIDWidget) DefaultColor() string { return defaultDimColor }

// DisplayName returns the human-readable name.
func (w *SessionIDWidget) DisplayName() string { return "Session ID" }

// Description returns what this widget shows.
func (w *SessionIDWidget) Description() string { return "Claude Code session identifier" }

// SupportsRawValue returns true since this widget supports full UUID mode.
func (w *SessionIDWidget) SupportsRawValue() bool { return true }
