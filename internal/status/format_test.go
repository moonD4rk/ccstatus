package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  string
	}{
		{"zero", 0, "0"},
		{"small", 500, "500"},
		{"exact thousand", 1000, "1.0k"},
		{"thousands", 1500, "1.5k"},
		{"large thousands", 50000, "50.0k"},
		{"exact million", 1_000_000, "1.0M"},
		{"millions", 1_200_000, "1.2M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatTokens(tt.count))
		})
	}
}

func TestGetContextConfig(t *testing.T) {
	intPtr := func(v int) *int { return &v }

	tests := []struct {
		name       string
		data       *StatusJSON
		wantMax    int
		wantUsable int
	}{
		{
			name: "from context_window_size",
			data: &StatusJSON{
				ContextWindow: &ContextWindow{ContextWindowSize: intPtr(200_000)},
			},
			wantMax:    200_000,
			wantUsable: 160_000,
		},
		{
			name: "1M context window",
			data: &StatusJSON{
				ContextWindow: &ContextWindow{ContextWindowSize: intPtr(1_000_000)},
			},
			wantMax:    1_000_000,
			wantUsable: 800_000,
		},
		{
			name:       "fallback to default",
			data:       &StatusJSON{Model: ModelField{ID: "claude-sonnet-4-5"}},
			wantMax:    200_000,
			wantUsable: 160_000,
		},
		{
			name:       "empty data defaults",
			data:       &StatusJSON{},
			wantMax:    200_000,
			wantUsable: 160_000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := GetContextConfig(tt.data)
			assert.Equal(t, tt.wantMax, cfg.MaxTokens)
			assert.Equal(t, tt.wantUsable, cfg.UsableTokens)
		})
	}
}

func TestGetContextPercentage(t *testing.T) {
	floatPtr := func(v float64) *float64 { return &v }
	intPtr := func(v int) *int { return &v }

	tests := []struct {
		name string
		data *StatusJSON
		want float64
	}{
		{
			name: "from used_percentage",
			data: &StatusJSON{
				ContextWindow: &ContextWindow{UsedPercentage: floatPtr(25.5)},
			},
			want: 25.5,
		},
		{
			name: "calculated from current_usage",
			data: &StatusJSON{
				ContextWindow: &ContextWindow{
					ContextWindowSize: intPtr(200_000),
					CurrentUsage: &CurrentUsage{
						InputTokens:              40_000,
						CacheCreationInputTokens: 5000,
						CacheReadInputTokens:     5000,
					},
				},
			},
			want: 25,
		},
		{
			name: "capped at 100",
			data: &StatusJSON{
				ContextWindow: &ContextWindow{
					ContextWindowSize: intPtr(100),
					CurrentUsage: &CurrentUsage{
						InputTokens: 200,
					},
				},
			},
			want: 100,
		},
		{
			name: "nil context window",
			data: &StatusJSON{},
			want: 0,
		},
		{
			name: "no current usage",
			data: &StatusJSON{
				ContextWindow: &ContextWindow{},
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.want, GetContextPercentage(tt.data), 0.01)
		})
	}
}

func TestGetContextLength(t *testing.T) {
	tests := []struct {
		name string
		data *StatusJSON
		want int
	}{
		{
			name: "sums all input tokens",
			data: &StatusJSON{
				ContextWindow: &ContextWindow{
					CurrentUsage: &CurrentUsage{
						InputTokens:              10_000,
						CacheCreationInputTokens: 2000,
						CacheReadInputTokens:     3000,
					},
				},
			},
			want: 15_000,
		},
		{
			name: "nil context window",
			data: &StatusJSON{},
			want: 0,
		},
		{
			name: "nil current usage",
			data: &StatusJSON{ContextWindow: &ContextWindow{}},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetContextLength(tt.data))
		})
	}
}
