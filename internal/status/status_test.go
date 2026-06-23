package status

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		check   func(t *testing.T, s *Session)
		wantErr bool
	}{
		{
			name:  "model as object",
			input: `{"model":{"id":"claude-sonnet-4-5","display_name":"Sonnet"}}`,
			check: func(t *testing.T, s *Session) {
				t.Helper()
				assert.Equal(t, "claude-sonnet-4-5", s.Model.ID)
				assert.Equal(t, "Sonnet", s.Model.DisplayName)
			},
		},
		{
			name:  "model as string",
			input: `{"model":"claude-opus-4-6"}`,
			check: func(t *testing.T, s *Session) {
				t.Helper()
				assert.Equal(t, "claude-opus-4-6", s.Model.ID)
				assert.Equal(t, "Opus", s.Model.DisplayName)
			},
		},
		{
			name:  "full official schema",
			input: `{"cwd":"/test","session_id":"abc123","version":"1.0.80","model":{"id":"claude-opus-4-6","display_name":"Opus"},"workspace":{"current_dir":"/test","project_dir":"/project"},"cost":{"total_cost_usd":0.01234,"total_duration_ms":45000},"context_window":{"total_input_tokens":15234,"total_output_tokens":4521,"context_window_size":200000,"used_percentage":8,"remaining_percentage":92,"current_usage":{"input_tokens":8500,"output_tokens":1200,"cache_creation_input_tokens":5000,"cache_read_input_tokens":2000}},"exceeds_200k_tokens":false,"vim":{"mode":"NORMAL"},"agent":{"name":"security-reviewer"}}`,
			check: func(t *testing.T, s *Session) {
				t.Helper()
				assert.Equal(t, "/test", s.Cwd)
				assert.Equal(t, "abc123", s.SessionID)
				assert.Equal(t, "1.0.80", s.Version)
				assert.Equal(t, "Opus", s.Model.DisplayName)

				require.NotNil(t, s.Workspace)
				assert.Equal(t, "/test", s.Workspace.CurrentDir)
				assert.Equal(t, "/project", s.Workspace.ProjectDir)

				require.NotNil(t, s.Cost)
				require.NotNil(t, s.Cost.TotalCostUSD)
				assert.InDelta(t, 0.01234, *s.Cost.TotalCostUSD, 0.00001)
				require.NotNil(t, s.Cost.TotalDurationMS)
				assert.InDelta(t, 45000.0, *s.Cost.TotalDurationMS, 0.1)

				require.NotNil(t, s.ContextWindow)
				require.NotNil(t, s.ContextWindow.TotalInputTokens)
				assert.Equal(t, 15234, *s.ContextWindow.TotalInputTokens)
				require.NotNil(t, s.ContextWindow.ContextWindowSize)
				assert.Equal(t, 200000, *s.ContextWindow.ContextWindowSize)
				require.NotNil(t, s.ContextWindow.UsedPercentage)
				assert.InDelta(t, 8.0, *s.ContextWindow.UsedPercentage, 0.1)

				require.NotNil(t, s.ContextWindow.CurrentUsage)
				assert.Equal(t, 8500, s.ContextWindow.CurrentUsage.InputTokens)
				assert.Equal(t, 1200, s.ContextWindow.CurrentUsage.OutputTokens)
				assert.Equal(t, 5000, s.ContextWindow.CurrentUsage.CacheCreationInputTokens)
				assert.Equal(t, 2000, s.ContextWindow.CurrentUsage.CacheReadInputTokens)

				require.NotNil(t, s.Exceeds200K)
				assert.False(t, *s.Exceeds200K)

				require.NotNil(t, s.Vim)
				assert.Equal(t, "NORMAL", s.Vim.Mode)

				require.NotNil(t, s.Agent)
				assert.Equal(t, "security-reviewer", s.Agent.Name)
			},
		},
		{
			name:  "new schema fields",
			input: `{"session_name":"my-feature","model":{"id":"claude-opus-4-7","display_name":"Opus"},"workspace":{"current_dir":"/x","added_dirs":["/a","/b"],"git_worktree":"feat-1"},"effort":{"level":"high"},"thinking":{"enabled":true},"rate_limits":{"five_hour":{"used_percentage":45.5,"resets_at":1738425600},"seven_day":{"used_percentage":30,"resets_at":1738857600}}}`,
			check: func(t *testing.T, s *Session) {
				t.Helper()
				assert.Equal(t, "my-feature", s.SessionName)

				require.NotNil(t, s.Workspace)
				assert.Equal(t, []string{"/a", "/b"}, s.Workspace.AddedDirs)
				assert.Equal(t, "feat-1", s.Workspace.GitWorktree)

				require.NotNil(t, s.Effort)
				assert.Equal(t, "high", s.Effort.Level)

				require.NotNil(t, s.Thinking)
				assert.True(t, s.Thinking.Enabled)

				require.NotNil(t, s.RateLimits)
				require.NotNil(t, s.RateLimits.FiveHour)
				require.NotNil(t, s.RateLimits.FiveHour.UsedPercentage)
				assert.InDelta(t, 45.5, *s.RateLimits.FiveHour.UsedPercentage, 0.01)
				require.NotNil(t, s.RateLimits.FiveHour.ResetsAt)
				assert.Equal(t, int64(1738425600), *s.RateLimits.FiveHour.ResetsAt)
				require.NotNil(t, s.RateLimits.SevenDay)
				require.NotNil(t, s.RateLimits.SevenDay.UsedPercentage)
				assert.InDelta(t, 30.0, *s.RateLimits.SevenDay.UsedPercentage, 0.01)
			},
		},
		{
			name:  "pr and workspace.repo",
			input: `{"workspace":{"repo":{"host":"github.com","owner":"anthropics","name":"claude-code"}},"pr":{"number":1234,"url":"https://github.com/anthropics/claude-code/pull/1234","review_state":"approved"}}`,
			check: func(t *testing.T, s *Session) {
				t.Helper()
				require.NotNil(t, s.Workspace)
				require.NotNil(t, s.Workspace.Repo)
				assert.Equal(t, "github.com", s.Workspace.Repo.Host)
				assert.Equal(t, "anthropics", s.Workspace.Repo.Owner)
				assert.Equal(t, "claude-code", s.Workspace.Repo.Name)

				require.NotNil(t, s.PR)
				require.NotNil(t, s.PR.Number)
				assert.Equal(t, 1234, *s.PR.Number)
				assert.Equal(t, "https://github.com/anthropics/claude-code/pull/1234", s.PR.URL)
				assert.Equal(t, "approved", s.PR.ReviewState)
			},
		},
		{
			name:  "pr present but review_state independently absent",
			input: `{"pr":{"number":7}}`,
			check: func(t *testing.T, s *Session) {
				t.Helper()
				require.NotNil(t, s.PR)
				require.NotNil(t, s.PR.Number)
				assert.Equal(t, 7, *s.PR.Number)
				assert.Empty(t, s.PR.ReviewState)
			},
		},
		{
			name:  "empty JSON object",
			input: `{}`,
			check: func(t *testing.T, s *Session) {
				t.Helper()
				assert.Empty(t, s.Model.ID)
				assert.Nil(t, s.ContextWindow)
				assert.Nil(t, s.Vim)
				assert.Nil(t, s.Agent)
				assert.Empty(t, s.SessionName)
				assert.Nil(t, s.Effort)
				assert.Nil(t, s.Thinking)
				assert.Nil(t, s.RateLimits)
				assert.Nil(t, s.PR)
			},
		},
		{
			name:  "null context_window.current_usage",
			input: `{"context_window":{"used_percentage":null,"current_usage":null}}`,
			check: func(t *testing.T, s *Session) {
				t.Helper()
				require.NotNil(t, s.ContextWindow)
				assert.Nil(t, s.ContextWindow.UsedPercentage)
				assert.Nil(t, s.ContextWindow.CurrentUsage)
			},
		},
		{
			name:    "invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := Parse([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, s)
			if tt.check != nil {
				tt.check(t, s)
			}
		})
	}
}

func TestThinkingEnabledRoundTrips(t *testing.T) {
	// enabled has no omitempty, so an explicit false survives marshaling and a
	// re-serialized session does not lose "thinking explicitly off".
	data, err := json.Marshal(&Session{Thinking: &ThinkingInfo{Enabled: false}})
	require.NoError(t, err)
	assert.Contains(t, string(data), `"enabled":false`)
}

func TestModelField_MarshalJSON(t *testing.T) {
	m := ModelField{ID: "claude-sonnet-4-5", DisplayName: "Sonnet"}
	data, err := json.Marshal(&m)
	require.NoError(t, err)

	var result map[string]string
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, "claude-sonnet-4-5", result["id"])
	assert.Equal(t, "Sonnet", result["display_name"])
}

func TestInferDisplayName(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"claude-opus-4-6", "Opus"},
		{"claude-sonnet-4-5", "Sonnet"},
		{"claude-haiku-4-5", "Haiku"},
		{"unknown-model", "unknown-model"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			assert.Equal(t, tt.want, inferDisplayName(tt.id))
		})
	}
}
