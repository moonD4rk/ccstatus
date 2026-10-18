package status

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSubagent(t *testing.T) {
	input := `{
		"session_id": "abc123",
		"cwd": "/repo",
		"hook_event_name": "SubagentStatusLine",
		"columns": 96,
		"tasks": [
			{"id": "t1", "name": "reviewer", "type": "local_agent", "agentType": "code-reviewer",
			 "status": "running", "description": "Review the diff", "label": "Reading files",
			 "startTime": 1738425600000, "model": "claude-sonnet-5-5", "effort": "high",
			 "contextWindowSize": 200000, "tokenCount": 45200, "tokenSamples": [1000, 9000, 45200],
			 "cwd": "/repo/.claude/worktrees/review"},
			{"id": "t2", "type": "local_agent", "agentType": "Explore", "status": "completed",
			 "description": "Find callers", "label": "Find callers", "startTime": 1738425500000,
			 "effort": 32000, "tokenCount": 1200, "tokenSamples": [1200]}
		]
	}`
	p, dropped, err := ParseSubagent([]byte(input))
	require.NoError(t, err)
	assert.Empty(t, dropped)
	assert.Equal(t, 96, p.Columns)
	require.Len(t, p.Tasks, 2)

	first := p.Tasks[0]
	assert.Equal(t, "reviewer", first.Name)
	assert.Equal(t, "code-reviewer", first.AgentType)
	assert.Equal(t, int64(1738425600000), first.StartTime)
	assert.Equal(t, &TaskEffort{Level: "high"}, first.Effort)
	assert.Equal(t, 200000, *first.ContextWindowSize)
	assert.Equal(t, []int{1000, 9000, 45200}, first.TokenSamples)

	second := p.Tasks[1]
	assert.Empty(t, second.Name)
	assert.Equal(t, &TaskEffort{Budget: 32000}, second.Effort)
	assert.Nil(t, second.ContextWindowSize)
}

func TestParseSubagentToleratesBadTasks(t *testing.T) {
	input := `{"columns": 80, "tasks": [
		{"id": "ok", "tokenCount": 10},
		"not a task",
		{"id": "drift", "tokenCount": "12", "startTime": 1738425600000.5, "tokenSamples": [1, "x", 3]}
	]}`
	p, dropped, err := ParseSubagent([]byte(input))
	require.NoError(t, err)
	assert.Equal(t, []string{"tasks[1]", "tasks[2].tokenSamples[1]"}, dropped)
	require.Len(t, p.Tasks, 2)
	assert.Equal(t, "ok", p.Tasks[0].ID)
	assert.Equal(t, "drift", p.Tasks[1].ID)
	assert.Equal(t, 12, p.Tasks[1].TokenCount)
	assert.Equal(t, int64(1738425600000), p.Tasks[1].StartTime)
	assert.Equal(t, []int{1, 3}, p.Tasks[1].TokenSamples)

	_, _, err = ParseSubagent([]byte(`[]`))
	assert.Error(t, err)
}

func TestTaskEffortRoundTrip(t *testing.T) {
	for _, in := range []string{`"max"`, `16000`} {
		var e TaskEffort
		require.NoError(t, json.Unmarshal([]byte(in), &e))
		out, err := json.Marshal(e)
		require.NoError(t, err)
		assert.JSONEq(t, in, string(out))
	}
	var e TaskEffort
	assert.Error(t, json.Unmarshal([]byte(`{}`), &e))
}
