// Package widget defines the widget interface, registry, and render context.
package widget

import (
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/git"
	"github.com/moond4rk/ccstatus/internal/status"
)

// The widget types the render pipeline and the validator treat specially.
const (
	TypeSeparator     = "separator"
	TypeFlexSeparator = "flex-separator"
	TypeCustomText    = "custom-text"
	TypeCustomCommand = "custom-command"
)

// RenderContext carries runtime data available to all widgets during rendering.
// Most token/context data comes directly from Data.ContextWindow (official API).
type RenderContext struct {
	Data          *status.Session
	TerminalWidth int
	Git           *GitCache
	// Now is the time the render started. Widgets measure countdowns and elapsed
	// times against it instead of reading the clock, so one render is consistent
	// and reproducible. The zero value means the current time.
	Now time.Time
	// RawInput is the exact stdin Claude Code piped in, forwarded verbatim to
	// custom commands so they see the full official schema (including fields
	// ccstatus does not model yet). Nil when unavailable.
	RawInput []byte
	// Task is the subagent row `ccstatus subagent` is rendering. Nil on the main
	// status line, where task widgets render empty.
	Task *status.Task
}

// now returns the render time, falling back to the clock for a context that
// sets none.
func (ctx RenderContext) now() time.Time {
	if ctx.Now.IsZero() {
		return time.Now()
	}
	return ctx.Now
}

// GitProvider supplies git repository facts for one render cycle.
// git.Repo satisfies it for production; tests inject a deterministic fake.
type GitProvider interface {
	Branch() string
	Changes() int
	Diff() git.DiffStat
	Worktree() string
}

// GitCache memoizes GitProvider results for a single render cycle so each git
// command runs at most once, even when prefetched concurrently. A nil cache
// yields zero values.
type GitCache struct {
	provider     GitProvider
	branchOnce   sync.Once
	changesOnce  sync.Once
	worktreeOnce sync.Once
	diffOnce     sync.Once
	branch       string
	changes      int
	worktree     string
	diff         git.DiffStat
}

// NewGitCache creates a cache whose git commands run in dir (empty = the
// process working directory).
func NewGitCache(dir string) *GitCache {
	return &GitCache{provider: git.Repo{Dir: dir}}
}

// NewGitCacheWithProvider creates a cache backed by a custom provider: a fake
// in tests, sample data in `ccstatus preview`.
func NewGitCacheWithProvider(p GitProvider) *GitCache {
	return &GitCache{provider: p}
}

// Branch returns the current git branch, caching the result.
func (c *GitCache) Branch() string {
	if c == nil || c.provider == nil {
		return ""
	}
	c.branchOnce.Do(func() { c.branch = c.provider.Branch() })
	return c.branch
}

// Changes returns the uncommitted change count, caching the result.
func (c *GitCache) Changes() int {
	if c == nil || c.provider == nil {
		return 0
	}
	c.changesOnce.Do(func() { c.changes = c.provider.Changes() })
	return c.changes
}

// Worktree returns the worktree name, caching the result.
func (c *GitCache) Worktree() string {
	if c == nil || c.provider == nil {
		return ""
	}
	c.worktreeOnce.Do(func() { c.worktree = c.provider.Worktree() })
	return c.worktree
}

// Diff returns the line-level diff stats, caching the result.
func (c *GitCache) Diff() git.DiffStat {
	if c == nil || c.provider == nil {
		return git.DiffStat{}
	}
	c.diffOnce.Do(func() { c.diff = c.provider.Diff() })
	return c.diff
}

// gitFact names one GitCache lookup.
type gitFact int

const (
	factBranch gitFact = iota
	factChanges
	factWorktree
	factDiff
)

// load runs the lookup for f through the cache.
func (c *GitCache) load(f gitFact) {
	switch f {
	case factBranch:
		c.Branch()
	case factChanges:
		c.Changes()
	case factWorktree:
		c.Worktree()
	case factDiff:
		c.Diff()
	}
}

// gitWidget is implemented by widgets that read git state, so their lookups can
// be prefetched before rendering.
type gitWidget interface {
	gitFacts(ctx RenderContext) []gitFact
}

// PrefetchGit runs the git lookups needed by the widgets in lines concurrently,
// so a render waits for the slowest lookup rather than the sum of them all.
func PrefetchGit(lines [][]config.WidgetItem, ctx RenderContext) {
	if ctx.Git == nil || ctx.Git.provider == nil {
		return
	}
	needed := make(map[gitFact]bool)
	for _, line := range lines {
		for i := range line {
			if w, ok := Get(line[i].Type).(gitWidget); ok {
				for _, f := range w.gitFacts(ctx) {
					needed[f] = true
				}
			}
		}
	}
	var wg sync.WaitGroup
	for f := range needed {
		wg.Go(func() { ctx.Git.load(f) })
	}
	wg.Wait()
}

// Widget defines the contract for all status line widgets. What only some
// widgets have (a label, a raw mode, a numeric value, display modes) is an
// optional interface below.
type Widget interface {
	// Render produces the widget text for the status line.
	// Returns empty string if the widget has nothing to display.
	Render(item *config.WidgetItem, ctx RenderContext, settings *config.Settings) string

	// DefaultColor returns the default foreground color name.
	DefaultColor() string

	// Description returns a short description of what the widget shows.
	Description() string
}

// defaultPrefixer is implemented by widgets that label their output, such as
// "Ctx: "; an item's own prefix replaces the label.
type defaultPrefixer interface {
	DefaultPrefix() string
}

// DefaultPrefix returns the label w puts before its output, or "" when it has
// none.
func DefaultPrefix(w Widget) string {
	if p, ok := w.(defaultPrefixer); ok {
		return p.DefaultPrefix()
	}
	return ""
}

// rawValuer is implemented by widgets whose output changes with an item's
// rawValue.
type rawValuer interface {
	SupportsRawValue() bool
}

// SupportsRawValue reports whether rawValue changes what the widget type
// renders.
func SupportsRawValue(widgetType string) bool {
	w, ok := Get(widgetType).(rawValuer)
	return ok && w.SupportsRawValue()
}

// Valuer is implemented by numeric widgets: Value exposes the number the widget
// renders, which selects the item's threshold colors.
type Valuer interface {
	Value(item *config.WidgetItem, ctx RenderContext) (float64, bool)
}

// DynamicDefaultColor is implemented by widgets whose default color depends on
// the data; the renderer consults it when the item sets no color.
type DynamicDefaultColor interface {
	DefaultColorFor(ctx RenderContext) string
}

// displayMode returns the display mode an item selects, or def when it sets none.
func displayMode(item *config.WidgetItem, def string) string {
	if item.Display != "" {
		return item.Display
	}
	return def
}

var registry = map[string]Widget{
	// Model and session
	"model":         &ModelWidget{},
	"version":       &VersionWidget{},
	"session-cost":  &SessionCostWidget{},
	"session-clock": &durationWidget{extract: sessionDuration, description: "Session duration", defaultPrefix: "Session: "},

	// Git
	"git-branch":   &GitBranchWidget{},
	"git-changes":  &GitChangesWidget{},
	"git-worktree": &GitWorktreeWidget{},

	// Token metrics. Since Claude Code v2.1.132 the total_input_tokens /
	// total_output_tokens fields reflect the current context window, not
	// cumulative session totals. tokens-input and context-length track the same
	// quantity; context-length also falls back to total_input_tokens when
	// current_usage is null (e.g. just after /compact), so it does not blank out.
	"tokens-input": &tokenWidget{
		extract: extractInputTokens, description: "Input tokens currently in context (includes cache reads/writes)",
		defaultPrefix: "In: ",
	},
	"tokens-output": &tokenWidget{
		extract: extractOutputTokens, description: "Output tokens from the most recent API response",
		defaultPrefix: "Out: ",
	},
	"tokens-cached": &tokenWidget{
		extract: extractCachedTokens, description: "Cached token count",
		defaultPrefix: "Cached: ",
	},
	"tokens-total": &tokenWidget{
		extract: extractTotalTokens, description: "Total tokens currently in the context window (input + output)",
		defaultPrefix: "Total: ",
	},

	"current-usage-input": &tokenWidget{
		extract: extractCurrentInputTokens, description: "Current round input token count",
		defaultPrefix: "CurIn: ",
	},
	"current-usage-output": &tokenWidget{
		extract: extractCurrentOutputTokens, description: "Current round output token count",
		defaultPrefix: "CurOut: ",
	},
	"cache-creation": &tokenWidget{
		extract: extractCacheCreationTokens, description: "Cache creation input token count",
		defaultPrefix: "CacheW: ",
	},

	// Context window
	"context-length": &ContextLengthWidget{},
	"context-percentage": &percentageWidget{
		extract: status.ContextPercentage, description: "Context usage as percentage of max window",
		defaultPrefix: "Ctx: ",
	},
	"context-percentage-usable": &ContextPercentageUsableWidget{},
	"remaining-percentage": &percentageWidget{
		extract: status.RemainingPercentage, description: "Remaining context window percentage",
		defaultPrefix: "Rem: ",
	},
	"cache-hit-rate": &percentageWidget{
		extract:       status.CacheHitRate,
		description:   "Session prompt cache hit ratio (falls back to the last request's cache reads)",
		defaultPrefix: "Cache: ", defaultColor: "cyan",
	},
	"prompt-cache": &PromptCacheWidget{},

	// Environment
	"current-working-dir": &CurrentDirWidget{},
	"project-dir":         &ProjectDirWidget{},
	"transcript-path":     &TranscriptPathWidget{},
	"added-dirs":          &AddedDirsWidget{},
	"repo":                &RepoWidget{},
	"pr":                  &PRWidget{},
	"lines-changed":       &LinesChangedWidget{},
	"lines-added":         &LinesAddedWidget{},
	"lines-removed":       &LinesRemovedWidget{},
	"session-lines":       &SessionLinesWidget{},

	// Cost and duration
	"api-duration": &durationWidget{extract: apiDuration, description: "Total API response time", defaultPrefix: "API: "},
	"block-timer":  &BlockTimerWidget{},

	// Rate limits
	"rate-limits": &RateLimitsWidget{},
	"rate-limit-5h": &RateLimitWidget{
		extract: fiveHourWindow, description: "5-hour rate-limit usage",
		defaultPrefix: "5h limit: ",
	},
	"rate-limit-7d": &RateLimitWidget{
		extract: sevenDayWindow, description: "7-day rate-limit usage",
		defaultPrefix: "7d limit: ",
	},
	"spend-limit": &SpendLimitWidget{},

	// Session info
	"session-id": &SessionIDWidget{},
	"output-style": &stringFieldWidget{
		extract: func(data *status.Session) string {
			if data.OutputStyle == nil {
				return ""
			}
			return data.OutputStyle.Name
		},
		defaultColor:  defaultDimColor,
		description:   "Current output style name",
		defaultPrefix: "Style: ",
	},
	"vim-mode": &stringFieldWidget{
		extract: func(data *status.Session) string {
			if data.Vim == nil {
				return ""
			}
			return data.Vim.Mode
		},
		defaultColor:  "yellow",
		description:   "Current vim mode indicator",
		defaultPrefix: "Vim: ",
	},
	"agent-name": &stringFieldWidget{
		extract: func(data *status.Session) string {
			if data.Agent == nil {
				return ""
			}
			return data.Agent.Name
		},
		defaultColor:  "cyan",
		description:   "Agent name when using --agent flag",
		defaultPrefix: "Agent: ",
	},
	"effort": &stringFieldWidget{
		extract: func(data *status.Session) string {
			if data.Effort == nil {
				return ""
			}
			return data.Effort.Level
		},
		defaultColor:  "cyan",
		description:   "Current reasoning effort level",
		defaultPrefix: "Effort: ",
	},
	"thinking": &stringFieldWidget{
		extract: func(data *status.Session) string {
			if data.Thinking == nil || !data.Thinking.Enabled {
				return ""
			}
			return "on"
		},
		defaultColor:  "magenta",
		description:   "Extended thinking indicator",
		defaultPrefix: "Think: ",
	},
	"fast-mode": &stringFieldWidget{
		extract: func(data *status.Session) string {
			if data.FastMode == nil || !*data.FastMode {
				return ""
			}
			return "Fast"
		},
		defaultColor: "yellow",
		description:  "Shows Fast while fast mode is on",
	},
	"session-name": &stringFieldWidget{
		extract: func(data *status.Session) string {
			return data.SessionName
		},
		defaultColor: defaultDimColor,
		description:  "Session name from --name or /rename, or the AI-generated title",
	},
	"exceeds-200k":   &Exceeds200KWidget{},
	"terminal-width": &TerminalWidthWidget{},

	// Subagent rows (subagentLine only)
	"task-name": &taskFieldWidget{
		extract: taskName, color: "cyan",
		description: "Subagent name, else its agent type",
	},
	"task-type": &taskFieldWidget{
		extract: func(t *status.Task, _ *config.WidgetItem) string { return t.AgentType }, color: "magenta",
		description: "Subagent type, such as Explore or a custom agent",
	},
	"task-status": &taskFieldWidget{
		extract: func(t *status.Task, _ *config.WidgetItem) string { return t.Status }, colorFor: taskStatusColor,
		color: defaultDimColor, description: "Subagent state, colored: running, completed, failed, killed",
	},
	"task-label": &taskFieldWidget{
		extract: taskLabel, color: defaultDimColor,
		description: "Subagent progress summary, else its description",
	},
	"task-elapsed": &TaskElapsedWidget{},
	"task-model": &taskFieldWidget{
		extract: taskModel, color: "cyan", raw: true,
		description: "Subagent model, e.g. Sonnet 5.5 (raw: model ID)",
	},
	"task-effort": &taskFieldWidget{
		extract: taskEffort, color: "cyan",
		description: "Subagent effort level or token budget",
	},
	"task-tokens":  &TaskTokensWidget{},
	"task-context": &TaskContextWidget{},
	"task-sparkline": &taskFieldWidget{
		extract: taskSparkline, color: "cyan",
		description: "Tokens a running subagent added per refresh tick",
	},
	"task-cwd": &taskFieldWidget{
		extract: taskCwd, color: "blue",
		description: "Base name of the subagent's working directory",
	},

	// User-defined
	TypeCustomText:    &CustomTextWidget{},
	TypeCustomCommand: &CustomCommandWidget{},

	// Layout
	TypeSeparator:     &SeparatorWidget{},
	TypeFlexSeparator: &FlexSeparatorWidget{},
}

// Get returns the widget for the given type string, or nil if unknown.
func Get(widgetType string) Widget {
	return registry[widgetType]
}

// displayModer is implemented by widgets that read an item's display option.
// The first mode is the default.
type displayModer interface {
	displayModes() []string
}

// DisplayModes returns the display values the widget type accepts, the default
// first, or false when the type has no display modes.
func DisplayModes(widgetType string) ([]string, bool) {
	if w, ok := Get(widgetType).(displayModer); ok {
		return w.displayModes(), true
	}
	return nil, false
}

// BarDisplay is the display mode that draws a bar and reads an item's barWidth.
const BarDisplay = "bar"

// HasBarDisplay reports whether the widget type has a bar display, the only one
// an item's barWidth applies to.
func HasBarDisplay(widgetType string) bool {
	modes, _ := DisplayModes(widgetType)
	return slices.Contains(modes, BarDisplay)
}

// optionUsers lists, for each item option only some widgets read, the widget
// types that read it.
var optionUsers = map[string][]string{
	"character":      {TypeSeparator, "git-branch"},
	"customText":     {TypeCustomText},
	"commandPath":    {TypeCustomCommand},
	"timeout":        {TypeCustomCommand},
	"preserveColors": {TypeCustomCommand},
}

// OptionUsers returns the widget types that read the named item option, or nil
// for an option every widget takes.
func OptionUsers(option string) []string {
	return optionUsers[option]
}

// SpecificOptions returns, sorted, the item options that only some widget
// types read.
func SpecificOptions() []string {
	return slices.Sorted(maps.Keys(optionUsers))
}

// RequiredOption returns the item option a widget type renders nothing
// without, or "".
func RequiredOption(widgetType string) string {
	switch widgetType {
	case TypeCustomText:
		return "customText"
	case TypeCustomCommand:
		return "commandPath"
	default:
		return ""
	}
}

// TakesThresholds reports whether the widget type renders a number that an
// item's thresholds can restyle.
func TakesThresholds(widgetType string) bool {
	_, ok := Get(widgetType).(Valuer)
	return ok
}

// Types returns all registered widget type names.
func Types() []string {
	types := make([]string, 0, len(registry))
	for k := range registry {
		types = append(types, k)
	}
	return types
}
