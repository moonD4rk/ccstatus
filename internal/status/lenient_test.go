package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseToleratesTypeDrift(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantDropped []string
		check       func(t *testing.T, s *Session)
	}{
		{
			name:  "fractional epoch is truncated into an integer field",
			input: `{"model":{"id":"m","display_name":"Opus"},"rate_limits":{"five_hour":{"used_percentage":23.5,"resets_at":1738425600.75}}}`,
			check: func(t *testing.T, s *Session) {
				assert.Equal(t, "Opus", s.Model.DisplayName)
				require.NotNil(t, s.RateLimits.FiveHour.ResetsAt)
				assert.Equal(t, int64(1738425600), *s.RateLimits.FiveHour.ResetsAt)
				assert.InDelta(t, 23.5, *s.RateLimits.FiveHour.UsedPercentage, 0.001)
			},
		},
		{
			name:  "quoted numbers fill numeric fields",
			input: `{"cost":{"total_cost_usd":"1.25","total_lines_added":"12"}}`,
			check: func(t *testing.T, s *Session) {
				assert.InDelta(t, 1.25, *s.Cost.TotalCostUSD, 0.001)
				assert.Equal(t, 12, *s.Cost.TotalLinesAdded)
			},
		},
		{
			name:  "number into a string field keeps its text",
			input: `{"session_id":123,"version":"2.1.302"}`,
			check: func(t *testing.T, s *Session) {
				assert.Equal(t, "123", s.SessionID)
				assert.Equal(t, "2.1.302", s.Version)
			},
		},
		{
			name:  "quoted boolean fills a boolean field",
			input: `{"fast_mode":"true","exceeds_200k_tokens":false}`,
			check: func(t *testing.T, s *Session) {
				require.NotNil(t, s.FastMode)
				assert.True(t, *s.FastMode)
			},
		},
		{
			name:        "unusable field is dropped, siblings survive",
			input:       `{"model":{"id":"m","display_name":"Opus"},"context_window":{"used_percentage":"high","context_window_size":200000}}`,
			wantDropped: []string{"context_window.used_percentage"},
			check: func(t *testing.T, s *Session) {
				assert.Equal(t, "Opus", s.Model.DisplayName)
				assert.Nil(t, s.ContextWindow.UsedPercentage)
				assert.Equal(t, 200000, *s.ContextWindow.ContextWindowSize)
			},
		},
		{
			name:        "object of the wrong shape leaves the pointer nil",
			input:       `{"rate_limits":"unavailable","session_name":"feat"}`,
			wantDropped: []string{"rate_limits"},
			check: func(t *testing.T, s *Session) {
				assert.Nil(t, s.RateLimits)
				assert.Equal(t, "feat", s.SessionName)
			},
		},
		{
			name:        "custom unmarshaler failure drops only that field",
			input:       `{"model":42,"cwd":"/repo"}`,
			wantDropped: []string{"model"},
			check: func(t *testing.T, s *Session) {
				assert.Empty(t, s.Model.ID)
				assert.Equal(t, "/repo", s.Cwd)
			},
		},
		{
			name:        "array where a list of strings is expected is dropped",
			input:       `{"workspace":{"current_dir":"/repo","added_dirs":[1,2]}}`,
			wantDropped: []string{"workspace.added_dirs"},
			check: func(t *testing.T, s *Session) {
				assert.Equal(t, "/repo", s.Workspace.CurrentDir)
				assert.Empty(t, s.Workspace.AddedDirs)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, dropped, err := Parse([]byte(tt.input))
			require.NoError(t, err)
			require.NotNil(t, s)
			assert.Equal(t, tt.wantDropped, dropped)
			tt.check(t, s)
		})
	}
}

func TestParseRejectsNonObjects(t *testing.T) {
	for _, input := range []string{``, `{`, `[]`, `"text"`, `42`, `{"model":}`} {
		t.Run(input, func(t *testing.T) {
			s, _, err := Parse([]byte(input))
			require.Error(t, err)
			assert.Nil(t, s)
		})
	}
}

func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"2.1.98", true},
		{"2.1.302", true},
		{"2.2.0", true},
		{"3.0.0", true},
		{"2.1.97", false},
		{"2.0.99", false},
		{"2.1.302-beta", true},
		{"", false},
		{"2.1", false},
		{"v2.1.302", false},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			s := &Session{Version: tt.version}
			assert.Equal(t, tt.want, s.VersionAtLeast(2, 1, 98))
		})
	}

	var nilSession *Session
	assert.False(t, nilSession.VersionAtLeast(0, 0, 0))
}

func FuzzParse(f *testing.F) {
	f.Add(`{"model":{"id":"claude-opus-5-5","display_name":"Opus"},"context_window":{"used_percentage":8}}`)
	f.Add(`{"rate_limits":{"five_hour":{"resets_at":1.5e9},"seven_day":"x"},"session_id":7}`)
	f.Add(`{"cost":{"total_cost_usd":"NaN","total_lines_added":1e300},"model":"claude-sonnet-5-5"}`)
	f.Add(`{"workspace":{"added_dirs":"x","repo":[]},"pr":{"number":"12"},"thinking":{"enabled":"yes"}}`)
	f.Add(`null`)
	f.Fuzz(func(t *testing.T, input string) {
		s, _, err := Parse([]byte(input))
		if err == nil && s == nil {
			t.Fatal("nil session without an error")
		}
	})
}
