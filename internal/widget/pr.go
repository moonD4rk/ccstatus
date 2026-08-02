package widget

import (
	"strconv"

	"github.com/moond4rk/ccstatus/internal/config"
)

// PRWidget displays the open pull request for the current branch (the same data
// as Claude Code's PR badge).
type PRWidget struct{ noAffix }

// Render returns "#<number>" plus the review state when present (just the bare
// number in raw mode), and empty when there is no open PR. When pr.url is
// present the number is wrapped in an OSC 8 hyperlink; terminals without
// hyperlink support ignore the sequence and show plain text.
func (w *PRWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil || ctx.Data.PR == nil || ctx.Data.PR.Number == nil {
		return ""
	}
	pr := ctx.Data.PR
	number := strconv.Itoa(*pr.Number)
	if item.RawValue {
		return number
	}
	out := "#" + number
	if pr.URL != "" {
		out = "\x1b]8;;" + pr.URL + "\x07" + out + "\x1b]8;;\x07"
	}
	if pr.ReviewState != "" {
		out += " " + pr.ReviewState
	}
	return out
}

// DefaultColor returns the default foreground color.
func (w *PRWidget) DefaultColor() string { return "cyan" }

// DisplayName returns the human-readable name.
func (w *PRWidget) DisplayName() string { return "Pull Request" }

// Description returns what this widget shows.
func (w *PRWidget) Description() string { return "Open PR number and review state" }

// SupportsRawValue returns true (raw mode shows the bare number).
func (w *PRWidget) SupportsRawValue() bool { return true }
