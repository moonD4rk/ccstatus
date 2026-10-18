package status

import (
	"encoding/json"
)

// SubagentPayload is the JSON Claude Code pipes to the subagentStatusLine
// command on every refresh tick: the base hook fields, the usable row width, and
// one task per subagent row in the agent panel.
type SubagentPayload struct {
	SessionID      string `json:"session_id,omitempty"`
	TranscriptPath string `json:"transcript_path,omitempty"`
	Cwd            string `json:"cwd,omitempty"`
	Columns        int    `json:"columns,omitempty"`
	Tasks          []Task `json:"tasks"`
}

// Task is one subagent row. Optional fields are omitted when they have no value.
type Task struct {
	ID                string      `json:"id"`
	Name              string      `json:"name,omitempty"`
	Type              string      `json:"type,omitempty"`      // local_agent
	AgentType         string      `json:"agentType,omitempty"` // e.g. Explore; v2.1.293+
	Status            string      `json:"status,omitempty"`    // running | completed | failed | killed
	Description       string      `json:"description,omitempty"`
	Label             string      `json:"label,omitempty"`
	StartTime         int64       `json:"startTime,omitempty"` // Unix epoch milliseconds
	Model             string      `json:"model,omitempty"`
	Effort            *TaskEffort `json:"effort,omitempty"`
	ContextWindowSize *int        `json:"contextWindowSize,omitempty"`
	TokenCount        int         `json:"tokenCount,omitempty"`
	TokenSamples      []int       `json:"tokenSamples,omitempty"` // up to 16, oldest first
	Cwd               string      `json:"cwd,omitempty"`
}

// TaskEffort is the effort configured for a subagent: a level such as "high",
// or a numeric token budget.
type TaskEffort struct {
	Level  string
	Budget int
}

// UnmarshalJSON accepts either a level string or a numeric budget.
func (e *TaskEffort) UnmarshalJSON(data []byte) error {
	var level string
	if json.Unmarshal(data, &level) == nil {
		*e = TaskEffort{Level: level}
		return nil
	}
	var budget float64
	if err := json.Unmarshal(data, &budget); err != nil {
		return err
	}
	*e = TaskEffort{Budget: int(budget)}
	return nil
}

// MarshalJSON writes the level string, or the budget when no level is set.
func (e TaskEffort) MarshalJSON() ([]byte, error) {
	if e.Level != "" {
		return json.Marshal(e.Level)
	}
	return json.Marshal(e.Budget)
}

// ParseSubagent decodes the subagentStatusLine payload with the same tolerance
// as Parse: a mistyped field drops only itself, so one bad task never hides the
// other rows.
func ParseSubagent(data []byte) (payload *SubagentPayload, dropped []string, err error) {
	var p SubagentPayload
	dropped, err = parseLenient(data, &p)
	if err != nil {
		return nil, nil, err
	}
	return &p, dropped, nil
}
