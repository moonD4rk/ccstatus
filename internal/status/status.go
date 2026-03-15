// Package status defines the JSON input schema from Claude Code.
package status

import (
	"encoding/json"
	"strings"
)

// Session represents the JSON payload piped from Claude Code.
// All fields are optional and may be absent or null.
type Session struct {
	Cwd            string         `json:"cwd,omitempty"`
	SessionID      string         `json:"session_id,omitempty"`
	TranscriptPath string         `json:"transcript_path,omitempty"`
	Model          ModelField     `json:"model,omitempty"`
	Workspace      *Workspace     `json:"workspace,omitempty"`
	Version        string         `json:"version,omitempty"`
	OutputStyle    *OutputStyle   `json:"output_style,omitempty"`
	Cost           *CostInfo      `json:"cost,omitempty"`
	ContextWindow  *ContextWindow `json:"context_window,omitempty"`
	Exceeds200K    *bool          `json:"exceeds_200k_tokens,omitempty"`
	Vim            *VimInfo       `json:"vim,omitempty"`
	Agent          *AgentInfo     `json:"agent,omitempty"`
	Worktree       *WorktreeInfo  `json:"worktree,omitempty"`
}

// Parse parses JSON data into Session.
func Parse(data []byte) (*Session, error) {
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
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
	CurrentDir string `json:"current_dir,omitempty"`
	ProjectDir string `json:"project_dir,omitempty"`
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
