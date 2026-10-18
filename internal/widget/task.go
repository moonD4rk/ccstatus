package widget

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

// taskWidget marks widgets that render a subagent task; they belong in
// subagentLine and render empty on the main status line.
type taskWidget interface {
	taskScope()
}

// IsTaskWidget reports whether a widget type renders subagent tasks, which only
// subagentLine can show.
func IsTaskWidget(widgetType string) bool {
	_, ok := Get(widgetType).(taskWidget)
	return ok
}

// IsSharedWidget reports whether a widget type works on both the main status
// line and subagent rows.
func IsSharedWidget(widgetType string) bool {
	switch widgetType {
	case "separator", "flex-separator", "custom-text":
		return true
	}
	return false
}

// taskFieldWidget renders one text field of the subagent task being rendered.
type taskFieldWidget struct {
	noAffix
	extract     func(task *status.Task, item *config.WidgetItem) string
	colorFor    func(task *status.Task) string
	color       string
	displayName string
	description string
	raw         bool
}

func (w *taskFieldWidget) taskScope() {}

// Render returns the field for the current task, or empty without one.
func (w *taskFieldWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Task == nil {
		return ""
	}
	return w.extract(ctx.Task, item)
}

// DefaultColorFor returns the state color for widgets that color by task state.
func (w *taskFieldWidget) DefaultColorFor(ctx RenderContext) string {
	if w.colorFor == nil || ctx.Task == nil {
		return ""
	}
	return w.colorFor(ctx.Task)
}

// DefaultColor returns the default foreground color.
func (w *taskFieldWidget) DefaultColor() string { return w.color }

// DisplayName returns the human-readable name.
func (w *taskFieldWidget) DisplayName() string { return w.displayName }

// Description returns what this widget shows.
func (w *taskFieldWidget) Description() string { return w.description }

// SupportsRawValue reports whether the widget has a raw output mode.
func (w *taskFieldWidget) SupportsRawValue() bool { return w.raw }

func taskName(t *status.Task, _ *config.WidgetItem) string {
	switch {
	case t.Name != "":
		return t.Name
	case t.AgentType != "":
		return t.AgentType
	default:
		return t.Type
	}
}

func taskLabel(t *status.Task, _ *config.WidgetItem) string {
	if t.Label != "" {
		return t.Label
	}
	return t.Description
}

func taskStatusColor(t *status.Task) string {
	switch t.Status {
	case "running":
		return "yellow"
	case "completed":
		return defaultGreenColor
	case "failed":
		return "red"
	case "killed":
		return "brightBlack"
	default:
		return ""
	}
}

// taskElapsed is empty unless the task is running: the payload has no end time,
// so a finished task's age would keep growing.
func taskElapsed(t *status.Task, _ *config.WidgetItem) string {
	if t.Status != "running" || t.StartTime <= 0 {
		return ""
	}
	elapsed := time.Since(time.UnixMilli(t.StartTime))
	if elapsed < 0 {
		return ""
	}
	return formatElapsed(elapsed)
}

// formatElapsed renders a running time with second precision: "45s", "1m23s",
// "1h02m".
func formatElapsed(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

func taskModel(t *status.Task, item *config.WidgetItem) string {
	if item.RawValue {
		return t.Model
	}
	return shortModelName(t.Model)
}

// shortModelName turns a model ID such as "claude-sonnet-5-5",
// "claude-opus-4-7[1m]" or "claude-3-5-haiku-20241022" into "Sonnet 5.5",
// "Opus 4.7" or "Haiku 3.5". IDs in another form are returned unchanged.
func shortModelName(id string) string {
	rest, ok := strings.CutPrefix(id, "claude-")
	if !ok {
		return id
	}
	if i := strings.IndexByte(rest, '['); i >= 0 {
		rest = rest[:i]
	}
	var family string
	var version []string
	for part := range strings.SplitSeq(rest, "-") {
		switch {
		case part == "":
		case strings.IndexFunc(part, unicode.IsLetter) >= 0:
			if family == "" {
				family = strings.ToUpper(part[:1]) + part[1:]
			}
		case len(part) <= 2: // version digits; longer runs are date stamps
			version = append(version, part)
		}
	}
	switch {
	case family == "":
		return id
	case len(version) == 0:
		return family
	default:
		return family + " " + strings.Join(version, ".")
	}
}

func taskEffort(t *status.Task, _ *config.WidgetItem) string {
	switch {
	case t.Effort == nil:
		return ""
	case t.Effort.Level != "":
		return t.Effort.Level
	case t.Effort.Budget > 0:
		return compactTokens(t.Effort.Budget)
	default:
		return ""
	}
}

// compactTokens formats a token budget, dropping a ".0" for whole thousands
// ("32k" rather than "32.0k").
func compactTokens(n int) string {
	if n >= 1000 && n < 1_000_000 && n%1000 == 0 {
		return fmt.Sprintf("%dk", n/1000)
	}
	return status.FormatTokens(n)
}

func taskCwd(t *status.Task, _ *config.WidgetItem) string {
	if t.Cwd == "" {
		return ""
	}
	return filepath.Base(t.Cwd)
}

// sparkBlocks are the eight block heights a sparkline steps through.
var sparkBlocks = []rune("▁▂▃▄▅▆▇█")

// sparkline renders samples as blocks scaled from zero to their maximum, so the
// row shows how fast a subagent's token count grew. Fewer than two samples
// render nothing.
func sparkline(samples []int) string {
	if len(samples) < 2 {
		return ""
	}
	hi := slices.Max(samples)
	var b strings.Builder
	for _, s := range samples {
		idx := 0
		if hi > 0 && s > 0 {
			idx = s * (len(sparkBlocks) - 1) / hi
		}
		b.WriteRune(sparkBlocks[idx])
	}
	return b.String()
}

// TaskTokensWidget displays a subagent's running token count, e.g. "45.2k".
type TaskTokensWidget struct{ noAffix }

func (w *TaskTokensWidget) taskScope() {}

// Render returns the formatted token count, or empty when it is zero.
func (w *TaskTokensWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Task == nil || ctx.Task.TokenCount <= 0 {
		return ""
	}
	return status.FormatTokens(ctx.Task.TokenCount)
}

// Value returns the token count, for threshold colors.
func (w *TaskTokensWidget) Value(_ *config.WidgetItem, ctx RenderContext) (float64, bool) {
	if ctx.Task == nil || ctx.Task.TokenCount <= 0 {
		return 0, false
	}
	return float64(ctx.Task.TokenCount), true
}

// DefaultColor returns the default foreground color.
func (w *TaskTokensWidget) DefaultColor() string { return defaultDimColor }

// DisplayName returns the human-readable name.
func (w *TaskTokensWidget) DisplayName() string { return "Task Tokens" }

// Description returns what this widget shows.
func (w *TaskTokensWidget) Description() string { return "Subagent running token count" }

// SupportsRawValue returns false since this widget has no compact mode.
func (w *TaskTokensWidget) SupportsRawValue() bool { return false }

// TaskContextWidget displays a subagent's context usage, tokenCount over its
// model's context window, e.g. "23%" or a bar with metadata.display "bar".
type TaskContextWidget struct{ noAffix }

func (w *TaskContextWidget) taskScope() {}

// Render returns the context percentage, or empty until the model is resolved.
func (w *TaskContextWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	pct, ok := w.Value(item, ctx)
	if !ok {
		return ""
	}
	return formatPercentage(item, pct)
}

// Value returns the context percentage, for threshold colors.
func (w *TaskContextWidget) Value(_ *config.WidgetItem, ctx RenderContext) (float64, bool) {
	t := ctx.Task
	if t == nil || t.ContextWindowSize == nil || *t.ContextWindowSize <= 0 {
		return 0, false
	}
	return min(float64(t.TokenCount)/float64(*t.ContextWindowSize)*100, 100), true
}

func (w *TaskContextWidget) displayModes() []string { return []string{"percent", "bar"} }

// DefaultColor returns the default foreground color.
func (w *TaskContextWidget) DefaultColor() string { return defaultDimColor }

// DisplayName returns the human-readable name.
func (w *TaskContextWidget) DisplayName() string { return "Task Context %" }

// Description returns what this widget shows.
func (w *TaskContextWidget) Description() string {
	return "Subagent context usage as % of its model's window"
}

// SupportsRawValue returns true; raw mode drops the % sign.
func (w *TaskContextWidget) SupportsRawValue() bool { return true }
