// Package widget defines the widget interface, registry, and render context.
package widget

import (
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/git"
	"github.com/moond4rk/ccstatus/internal/status"
)

// RenderContext carries runtime data available to all widgets during rendering.
// Most token/context data comes directly from Data.ContextWindow (official API).
type RenderContext struct {
	Data          *status.Session
	TerminalWidth int
	Git           *GitCache
	// RawInput is the exact stdin Claude Code piped in, forwarded verbatim to
	// custom commands so they see the full official schema (including fields
	// ccstatus does not model yet). Nil when unavailable.
	RawInput []byte
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
// command runs at most once. A nil cache yields zero values.
type GitCache struct {
	provider GitProvider
	branch   *string
	changes  *int
	worktree *string
	diff     *git.DiffStat
}

// NewGitCache creates a cache whose git commands run in dir (empty = the
// process working directory).
func NewGitCache(dir string) *GitCache {
	return &GitCache{provider: git.Repo{Dir: dir}}
}

// NewGitCacheWithProvider creates a cache backed by a custom provider, for tests.
func NewGitCacheWithProvider(p GitProvider) *GitCache {
	return &GitCache{provider: p}
}

// Branch returns the current git branch, caching the result.
func (c *GitCache) Branch() string {
	if c == nil || c.provider == nil {
		return ""
	}
	if c.branch == nil {
		b := c.provider.Branch()
		c.branch = &b
	}
	return *c.branch
}

// Changes returns the uncommitted change count, caching the result.
func (c *GitCache) Changes() int {
	if c == nil || c.provider == nil {
		return 0
	}
	if c.changes == nil {
		n := c.provider.Changes()
		c.changes = &n
	}
	return *c.changes
}

// Worktree returns the worktree name, caching the result.
func (c *GitCache) Worktree() string {
	if c == nil || c.provider == nil {
		return ""
	}
	if c.worktree == nil {
		w := c.provider.Worktree()
		c.worktree = &w
	}
	return *c.worktree
}

// Diff returns the line-level diff stats, caching the result.
func (c *GitCache) Diff() git.DiffStat {
	if c == nil || c.provider == nil {
		return git.DiffStat{}
	}
	if c.diff == nil {
		d := c.provider.Diff()
		c.diff = &d
	}
	return *c.diff
}

// Widget defines the contract for all status line widgets.
type Widget interface {
	// Render produces the widget text for the status line.
	// Returns empty string if the widget has nothing to display.
	Render(item *config.WidgetItem, ctx RenderContext, settings *config.Settings) string

	// DefaultColor returns the default foreground color name.
	DefaultColor() string

	// DisplayName returns the human-readable widget name.
	DisplayName() string

	// Description returns a short description of what the widget shows.
	Description() string

	// SupportsRawValue indicates if the widget has a compact output mode.
	SupportsRawValue() bool

	// DefaultPrefix returns text prepended to the output when the user set none.
	// Embed noAffix to inherit "".
	DefaultPrefix() string

	// DefaultSuffix returns text appended to the output when the user set none.
	// Embed noAffix to inherit "".
	DefaultSuffix() string
}

var registry = map[string]Widget{
	// Model and session
	"model":         &ModelWidget{},
	"version":       &VersionWidget{},
	"session-cost":  &SessionCostWidget{},
	"session-clock": &SessionClockWidget{},

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
		extract: extractInputTokens, displayName: "Input Tokens", description: "Input tokens currently in context (includes cache reads/writes)",
		defaultPrefix: "In: ",
	},
	"tokens-output": &tokenWidget{
		extract: extractOutputTokens, displayName: "Output Tokens", description: "Output tokens from the most recent API response",
		defaultPrefix: "Out: ",
	},
	"tokens-cached": &tokenWidget{
		extract: extractCachedTokens, displayName: "Cached Tokens", description: "Cached token count",
		defaultPrefix: "Cached: ",
	},
	"tokens-total": &tokenWidget{
		extract: extractTotalTokens, displayName: "Total Tokens", description: "Total tokens currently in the context window (input + output)",
		defaultPrefix: "Total: ",
	},

	"current-usage-input": &tokenWidget{
		extract: extractCurrentInputTokens, displayName: "Current Input Tokens", description: "Current round input token count",
		defaultPrefix: "CurIn: ",
	},
	"current-usage-output": &tokenWidget{
		extract: extractCurrentOutputTokens, displayName: "Current Output Tokens", description: "Current round output token count",
		defaultPrefix: "CurOut: ",
	},
	"cache-creation": &tokenWidget{
		extract: extractCacheCreationTokens, displayName: "Cache Creation Tokens", description: "Cache creation input token count",
		defaultPrefix: "CacheW: ",
	},

	// Context window
	"context-length": &ContextLengthWidget{},
	"context-percentage": &percentageWidget{
		extract: status.ContextPercentage, displayName: "Context %", description: "Context usage as percentage of max window",
		defaultPrefix: "Ctx: ",
	},
	"context-percentage-usable": &ContextPercentageUsableWidget{},
	"remaining-percentage": &percentageWidget{
		extract: status.RemainingPercentage, displayName: "Remaining %", description: "Remaining context window percentage",
		defaultPrefix: "Rem: ",
	},
	"cache-hit-rate": &percentageWidget{
		extract: status.CacheHitRate, displayName: "Cache Hit Rate", description: "Cache read token ratio as percentage",
		defaultPrefix: "Cache: ", defaultColor: "cyan",
	},

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

	// Cost and duration
	"api-duration": &APIDurationWidget{},
	"block-timer":  &BlockTimerWidget{},

	// Rate limits
	"rate-limits": &RateLimitsWidget{},
	"rate-limit-5h": &RateLimitWidget{
		extract: fiveHourWindow, displayName: "5h Rate Limit", description: "5-hour rate-limit usage",
		defaultPrefix: "5h limit: ",
	},
	"rate-limit-7d": &RateLimitWidget{
		extract: sevenDayWindow, displayName: "7d Rate Limit", description: "7-day rate-limit usage",
		defaultPrefix: "7d limit: ",
	},

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
		displayName:   "Output Style",
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
		displayName:   "Vim Mode",
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
		displayName:   "Agent Name",
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
		displayName:   "Reasoning Effort",
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
		displayName:   "Thinking",
		description:   "Extended thinking indicator",
		defaultPrefix: "Think: ",
	},
	"session-name": &stringFieldWidget{
		extract: func(data *status.Session) string {
			return data.SessionName
		},
		defaultColor: defaultDimColor,
		displayName:  "Session Name",
		description:  "Custom session name set with --name or /rename",
	},
	"exceeds-200k":   &Exceeds200KWidget{},
	"terminal-width": &TerminalWidthWidget{},

	// User-defined
	"custom-text":    &CustomTextWidget{},
	"custom-command": &CustomCommandWidget{},

	// Layout
	"separator":      &SeparatorWidget{},
	"flex-separator": &FlexSeparatorWidget{},
}

// noAffix provides empty default prefix/suffix. Embed it in widgets that add no
// affix of their own, so every Widget satisfies the interface without boilerplate.
// User-configured WidgetItem.Prefix/Suffix always take precedence over defaults.
type noAffix struct{}

func (noAffix) DefaultPrefix() string { return "" }
func (noAffix) DefaultSuffix() string { return "" }

// Get returns the widget for the given type string, or nil if unknown.
func Get(widgetType string) Widget {
	return registry[widgetType]
}

// Types returns all registered widget type names.
func Types() []string {
	types := make([]string, 0, len(registry))
	for k := range registry {
		types = append(types, k)
	}
	return types
}
