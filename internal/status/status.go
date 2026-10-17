// Package status defines the JSON input schema from Claude Code.
package status

import (
	"encoding/json"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// Session represents the JSON payload piped from Claude Code.
// All fields are optional and may be absent or null.
type Session struct {
	Cwd            string         `json:"cwd,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	SessionName    string         `json:"session_name,omitempty"`
	PromptID       string         `json:"prompt_id,omitempty"`
	TranscriptPath string         `json:"transcript_path,omitempty"`
	Model          ModelField     `json:"model,omitempty"`
	Workspace      *Workspace     `json:"workspace,omitempty"`
	Version        string         `json:"version,omitempty"`
	OutputStyle    *OutputStyle   `json:"output_style,omitempty"`
	Cost           *CostInfo      `json:"cost,omitempty"`
	ContextWindow  *ContextWindow `json:"context_window,omitempty"`
	Exceeds200K    *bool          `json:"exceeds_200k_tokens,omitempty"`
	FastMode       *bool          `json:"fast_mode,omitempty"`
	Effort         *EffortInfo    `json:"effort,omitempty"`
	Thinking       *ThinkingInfo  `json:"thinking,omitempty"`
	RateLimits     *RateLimits    `json:"rate_limits,omitempty"`
	Vim            *VimInfo       `json:"vim,omitempty"`
	Agent          *AgentInfo     `json:"agent,omitempty"`
	Worktree       *WorktreeInfo  `json:"worktree,omitempty"`
	PR             *PRInfo        `json:"pr,omitempty"`
}

// Parse decodes the JSON payload piped from Claude Code. Input that is not a
// JSON object (or null) is an error. A field of an unexpected type does not fail
// the parse, so one schema change cannot blank the status line: it is coerced
// when the intent is unambiguous and dropped otherwise, and the JSON paths of
// dropped fields are returned.
func Parse(data []byte) (session *Session, dropped []string, err error) {
	var s Session
	err = json.Unmarshal(data, &s)
	if err == nil {
		return &s, nil, nil
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil {
		return nil, nil, err
	}
	s = Session{}
	dropped = decodeLenient(data, reflect.ValueOf(&s).Elem(), "")
	return &s, dropped, nil
}

// VersionAtLeast reports whether the Claude Code version that sent the payload
// is at least major.minor.patch. An absent or unparsable version reports false.
func (s *Session) VersionAtLeast(major, minor, patch int) bool {
	if s == nil {
		return false
	}
	parts := strings.SplitN(s.Version, ".", 3)
	if len(parts) != 3 {
		return false
	}
	got := make([]int, len(parts))
	for i, p := range parts {
		// Tolerate pre-release suffixes such as "2.1.302-beta".
		digits := p[:len(p)-len(strings.TrimLeft(p, "0123456789"))]
		n, err := strconv.Atoi(digits)
		if err != nil {
			return false
		}
		got[i] = n
	}
	return slices.Compare(got, []int{major, minor, patch}) >= 0
}

// WorkingDir returns the directory git and path widgets should operate in,
// mirroring the upstream resolution order: workspace.current_dir, then cwd,
// then workspace.project_dir. Empty when none are set, so callers fall back to
// the process working directory.
func (s *Session) WorkingDir() string {
	if s == nil {
		return ""
	}
	if s.Workspace != nil && s.Workspace.CurrentDir != "" {
		return s.Workspace.CurrentDir
	}
	if s.Cwd != "" {
		return s.Cwd
	}
	if s.Workspace != nil && s.Workspace.ProjectDir != "" {
		return s.Workspace.ProjectDir
	}
	return ""
}

// ModelField handles the model field being either a string or an object.
// Official API always sends an object {id, display_name}, but older versions
// may send a plain string.
type ModelField struct {
	ID          string
	DisplayName string
}

// UnmarshalJSON handles model being either a string or an object.
func (m *ModelField) UnmarshalJSON(data []byte) error {
	var s string
	if json.Unmarshal(data, &s) == nil {
		m.ID = s
		m.DisplayName = inferDisplayName(s)
		return nil
	}
	var obj struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	m.ID = obj.ID
	m.DisplayName = obj.DisplayName
	return nil
}

// MarshalJSON serializes the ModelField as an object.
func (m *ModelField) MarshalJSON() ([]byte, error) {
	obj := struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	}{
		ID:          m.ID,
		DisplayName: m.DisplayName,
	}
	return json.Marshal(obj)
}

func inferDisplayName(id string) string {
	lower := strings.ToLower(id)
	switch {
	case strings.Contains(lower, "opus"):
		return "Opus"
	case strings.Contains(lower, "sonnet"):
		return "Sonnet"
	case strings.Contains(lower, "haiku"):
		return "Haiku"
	default:
		return id
	}
}

// Workspace holds directory information for the Claude Code session.
type Workspace struct {
	CurrentDir  string   `json:"current_dir,omitempty"`
	ProjectDir  string   `json:"project_dir,omitempty"`
	AddedDirs   []string `json:"added_dirs,omitempty"`
	GitWorktree string   `json:"git_worktree,omitempty"`
	Repo        *Repo    `json:"repo,omitempty"`
}

// Repo holds repository identity parsed from the origin remote (for example
// host "github.com", owner "anthropics", name "claude-code"). Absent outside a
// git repository or when no origin remote is configured.
type Repo struct {
	Host  string `json:"host,omitempty"`
	Owner string `json:"owner,omitempty"`
	Name  string `json:"name,omitempty"`
}

// PRInfo holds the open pull request for the current branch. Absent until a PR
// is found and removed once it merges or closes. ReviewState may be
// independently absent even when PRInfo is present.
type PRInfo struct {
	Number      *int   `json:"number,omitempty"`
	URL         string `json:"url,omitempty"`
	ReviewState string `json:"review_state,omitempty"` // approved | pending | changes_requested | draft
}

// OutputStyle holds the output style configuration.
type OutputStyle struct {
	Name string `json:"name,omitempty"`
}

// CostInfo holds session cost and duration metrics.
type CostInfo struct {
	TotalCostUSD       *float64 `json:"total_cost_usd,omitempty"`
	TotalDurationMS    *float64 `json:"total_duration_ms,omitempty"`
	TotalAPIDurationMS *float64 `json:"total_api_duration_ms,omitempty"`
	TotalLinesAdded    *int     `json:"total_lines_added,omitempty"`
	TotalLinesRemoved  *int     `json:"total_lines_removed,omitempty"`
}

// ContextWindow holds context window usage data provided directly by Claude Code.
// This eliminates the need to parse JSONL transcripts for most token-related widgets.
type ContextWindow struct {
	TotalInputTokens    *int          `json:"total_input_tokens,omitempty"`
	TotalOutputTokens   *int          `json:"total_output_tokens,omitempty"`
	ContextWindowSize   *int          `json:"context_window_size,omitempty"`
	UsedPercentage      *float64      `json:"used_percentage,omitempty"`
	RemainingPercentage *float64      `json:"remaining_percentage,omitempty"`
	CurrentUsage        *CurrentUsage `json:"current_usage,omitempty"`
}

// CurrentUsage holds token counts from the most recent API call.
// used_percentage is calculated from input tokens only:
// input_tokens + cache_creation_input_tokens + cache_read_input_tokens.
type CurrentUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// VimInfo is only present when vim mode is enabled in Claude Code.
type VimInfo struct {
	Mode string `json:"mode,omitempty"`
}

// AgentInfo is only present when running with --agent flag or agent settings.
type AgentInfo struct {
	Name string `json:"name,omitempty"`
}

// WorktreeInfo is only present during --worktree sessions.
// The Branch and OriginalBranch fields may be absent for hook-based worktrees.
type WorktreeInfo struct {
	Name           string `json:"name,omitempty"`
	Path           string `json:"path,omitempty"`
	Branch         string `json:"branch,omitempty"`
	OriginalCwd    string `json:"original_cwd,omitempty"`
	OriginalBranch string `json:"original_branch,omitempty"`
}

// EffortInfo holds the live reasoning-effort level. Absent when the current
// model does not support the effort parameter.
type EffortInfo struct {
	Level string `json:"level,omitempty"` // low | medium | high | xhigh | max
}

// ThinkingInfo indicates whether extended thinking is enabled for the session.
// Enabled has no omitempty so an explicit false survives a round-trip (the spec
// treats thinking.enabled as a meaningful, always-present bool).
type ThinkingInfo struct {
	Enabled bool `json:"enabled"`
}

// RateLimits holds Claude.ai subscription rate-limit windows. Present only for
// Pro/Max subscribers after the first API response; each window may be absent.
type RateLimits struct {
	FiveHour *RateLimitWindow `json:"five_hour,omitempty"`
	SevenDay *RateLimitWindow `json:"seven_day,omitempty"`
}

// RateLimitWindow holds usage data for a single rate-limit window.
type RateLimitWindow struct {
	UsedPercentage *float64 `json:"used_percentage,omitempty"` // 0-100
	ResetsAt       *int64   `json:"resets_at,omitempty"`       // Unix epoch seconds
}
