package widget

import (
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
	case TypeSeparator, TypeFlexSeparator, TypeCustomText:
		return true
	}
	return false
}

// taskFieldWidget renders one text field of the subagent task being rendered.
type taskFieldWidget struct {
	extract     func(task *status.Task, item *config.WidgetItem) string
	colorFor    func(task *status.Task) string
	color       string
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
	case status.TaskRunning:
		return "yellow"
	case status.TaskCompleted:
		return defaultGreenColor
	case status.TaskFailed:
		return "red"
	case status.TaskKilled:
		return "brightBlack"
	default:
		return ""
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
// "Opus 4.7" or "Haiku 3.5". It reads through what a provider wraps around the
// ID: a Bedrock "us.anthropic." prefix and "-v1:0" suffix, a Vertex "@date"
// suffix, a gateway "anthropic/" prefix. IDs in another form are returned
// unchanged.
func shortModelName(id string) string {
	const marker = "claude-"
	start := strings.LastIndex(id, marker)
	if start < 0 {
		return id
	}
	rest := id[start+len(marker):]
	if i := strings.IndexAny(rest, "[@"); i >= 0 {
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

func taskCwd(t *status.Task, _ *config.WidgetItem) string {
	if t.Cwd == "" {
		return ""
	}
	return filepath.Base(t.Cwd)
}

// sparkBlocks are the eight block heights a sparkline steps through.
var sparkBlocks = []rune("▁▂▃▄▅▆▇█")

// taskSparkline is empty unless the task is running: a finished task's samples stop changing.
func taskSparkline(t *status.Task, _ *config.WidgetItem) string {
	if t.Status != status.TaskRunning {
		return ""
	}
	return sparkline(t.TokenSamples)
}

// sparkline draws the tokens added between samples, so a fork's inherited context does not
// fill the bar. Fewer than two samples render nothing.
func sparkline(samples []int) string {
	if len(samples) < 2 {
		return ""
	}
	steps := make([]int, len(samples)-1)
	for i := range steps {
		steps[i] = max(samples[i+1]-samples[i], 0)
	}
	hi := slices.Max(steps)
	top := len(sparkBlocks) - 1
	var b strings.Builder
	for _, s := range steps {
		idx := 0
		if s > 0 {
			idx = (s*top + hi - 1) / hi // round up so any growth sits above an idle tick
		}
		b.WriteRune(sparkBlocks[idx])
	}
	return b.String()
}

// TaskElapsedWidget displays how long a running subagent has been running, e.g.
// "1m23s".
type TaskElapsedWidget struct{}

func (w *TaskElapsedWidget) taskScope() {}

// Render is empty unless the task is running: the payload has no end time, so a
// finished task's age would keep growing.
func (w *TaskElapsedWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	t := ctx.Task
	if t == nil || t.Status != status.TaskRunning || t.StartTime <= 0 {
		return ""
	}
	elapsed := ctx.now().Sub(time.UnixMilli(t.StartTime))
	if elapsed < 0 {
		return ""
	}
	return formatElapsed(elapsed)
}

// DefaultColor returns the default foreground color.
func (w *TaskElapsedWidget) DefaultColor() string { return defaultDimColor }

// Description returns what this widget shows.
func (w *TaskElapsedWidget) Description() string { return "Running time of a running subagent" }

// TaskTokensWidget displays a subagent's running token count, e.g. "45.2k".
type TaskTokensWidget struct{}

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

// Description returns what this widget shows.
func (w *TaskTokensWidget) Description() string { return "Subagent running token count" }

// TaskContextWidget displays a subagent's context usage, tokenCount over its
// model's context window, e.g. "23%" or a bar with display "bar".
type TaskContextWidget struct{}

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

func (w *TaskContextWidget) displayModes() []string { return []string{"percent", BarDisplay} }

// DefaultColor returns the default foreground color.
func (w *TaskContextWidget) DefaultColor() string { return defaultDimColor }

// Description returns what this widget shows.
func (w *TaskContextWidget) Description() string {
	return "Subagent context usage as % of its model's window"
}

// SupportsRawValue returns true; raw mode drops the % sign.
func (w *TaskContextWidget) SupportsRawValue() bool { return true }
