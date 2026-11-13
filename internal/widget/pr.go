package widget

import (
	"strconv"

	"github.com/moond4rk/ccstatus/internal/config"
)

// PRWidget displays the open pull request for the current branch (the same data
// as Claude Code's PR badge), colored by its review state.
type PRWidget struct{}

// Render returns "#<number>" (or "!<number>" for a GitLab merge request) plus the
// review state when present (just the bare number in raw mode), and empty when
// there is no open PR. When pr.url is present and hyperlinks are on, the number
// is an OSC 8 link to it.
func (w *PRWidget) Render(item *config.WidgetItem, ctx RenderContext, settings *config.Settings) string {
	if ctx.Data == nil || ctx.Data.PR == nil || ctx.Data.PR.Number == nil {
		return ""
	}
	pr := ctx.Data.PR
	number := strconv.Itoa(*pr.Number)
	if item.RawValue {
		return number
	}
	sigil := "#"
	if pr.IsMergeRequest() {
		sigil = "!"
	}
	out := hyperlink(settings, pr.URL, sigil+number)
	if pr.ReviewState != "" {
		out += " " + pr.ReviewState
	}
	return out
}

// DefaultColorFor colors the PR by its review state: green once approved, red
// when changes are requested, yellow while pending and dim as a draft. Other
// states, and none, keep the default color.
func (w *PRWidget) DefaultColorFor(ctx RenderContext) string {
	if ctx.Data == nil || ctx.Data.PR == nil {
		return ""
	}
	switch ctx.Data.PR.ReviewState {
	case "approved":
		return defaultGreenColor
	case "changes_requested":
		return "red"
	case "pending":
		return "yellow"
	case "draft":
		return "brightBlack"
	default:
		return ""
	}
}

// DefaultColor returns the default foreground color.
func (w *PRWidget) DefaultColor() string { return "cyan" }

// Description returns what this widget shows.
func (w *PRWidget) Description() string {
	return "Open PR (#N) or GitLab merge request (!N) and review state"
}

// SupportsRawValue returns true (raw mode shows the bare number).
func (w *PRWidget) SupportsRawValue() bool { return true }
