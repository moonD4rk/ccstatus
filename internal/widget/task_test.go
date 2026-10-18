package widget

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

func TestTaskWidgets(t *testing.T) {
	settings := config.DefaultSettings()
	running := &status.Task{
		ID: "t1", Name: "reviewer", Type: "local_agent", AgentType: "code-reviewer",
		Status: "running", Description: "Review the diff", Label: "Reading files",
		StartTime: time.Now().Add(-(83*time.Second + 400*time.Millisecond)).UnixMilli(),
		Model:     "claude-sonnet-5-5", Effort: &status.TaskEffort{Level: "high"},
		ContextWindowSize: intPtr(200_000), TokenCount: 45_200,
		TokenSamples: []int{0, 11_300, 22_600, 45_200}, Cwd: "/repo/.claude/worktrees/review",
	}
	done := &status.Task{
		ID: "t2", Type: "local_agent", AgentType: "Explore", Status: "completed",
		Description: "Find callers", StartTime: time.Now().Add(-time.Hour).UnixMilli(),
		Model: "claude-haiku-5-5", Effort: &status.TaskEffort{Budget: 32_000}, TokenCount: 1200,
		TokenSamples: []int{900, 1200, 1200},
	}

	tests := []struct {
		widget string
		task   *status.Task
		item   config.WidgetItem
		want   string
	}{
		{"task-name", running, config.WidgetItem{}, "reviewer"},
		{"task-name", done, config.WidgetItem{}, "Explore"},
		{"task-name", &status.Task{Type: "local_agent"}, config.WidgetItem{}, "local_agent"},
		{"task-type", running, config.WidgetItem{}, "code-reviewer"},
		{"task-status", done, config.WidgetItem{}, "completed"},
		{"task-label", running, config.WidgetItem{}, "Reading files"},
		{"task-label", done, config.WidgetItem{}, "Find callers"},
		{"task-elapsed", running, config.WidgetItem{}, "1m23s"},
		{"task-elapsed", done, config.WidgetItem{}, ""},
		{"task-model", running, config.WidgetItem{}, "Sonnet 5.5"},
		{"task-model", running, config.WidgetItem{RawValue: true}, "claude-sonnet-5-5"},
		{"task-effort", running, config.WidgetItem{}, "high"},
		{"task-effort", done, config.WidgetItem{}, "32k"},
		{"task-tokens", running, config.WidgetItem{}, "45.2k"},
		{"task-context", running, config.WidgetItem{}, "23%"},
		{"task-context", running, config.WidgetItem{Metadata: map[string]string{"display": "bar"}}, "▓▓░░░░░░░░ 23%"},
		{"task-context", done, config.WidgetItem{}, ""},
		{"task-sparkline", running, config.WidgetItem{}, "▅▅█"},
		{"task-sparkline", done, config.WidgetItem{}, ""},
		{"task-cwd", running, config.WidgetItem{}, "review"},
	}
	for _, tt := range tests {
		t.Run(tt.widget+"/"+tt.want, func(t *testing.T) {
			w := Get(tt.widget)
			require.NotNil(t, w)
			assert.Equal(t, tt.want, w.Render(&tt.item, RenderContext{Task: tt.task}, &settings))
		})
	}

	t.Run("task widgets are empty on the main status line", func(t *testing.T) {
		for _, typ := range Types() {
			if IsTaskWidget(typ) {
				assert.Emptyf(t, Get(typ).Render(&config.WidgetItem{}, RenderContext{Data: &status.Session{}}, &settings), typ)
			}
		}
	})

	t.Run("status colors by state", func(t *testing.T) {
		dyn, ok := Get("task-status").(DynamicDefaultColor)
		require.True(t, ok)
		for state, want := range map[string]string{
			"running": "yellow", "completed": "green", "failed": "red", "killed": "brightBlack", "paused": "",
		} {
			assert.Equal(t, want, dyn.DefaultColorFor(RenderContext{Task: &status.Task{Status: state}}), state)
		}
		assert.Empty(t, Get("task-name").(DynamicDefaultColor).DefaultColorFor(RenderContext{Task: running}))
	})

	t.Run("numeric task widgets expose values", func(t *testing.T) {
		v, ok := Get("task-context").(Valuer).Value(&config.WidgetItem{}, RenderContext{Task: running})
		require.True(t, ok)
		assert.InDelta(t, 22.6, v, 0.001)
		v, ok = Get("task-tokens").(Valuer).Value(&config.WidgetItem{}, RenderContext{Task: running})
		require.True(t, ok)
		assert.InDelta(t, 45_200, v, 0.001)
	})
}

func TestWidgetScopes(t *testing.T) {
	assert.True(t, IsTaskWidget("task-name"))
	assert.True(t, IsTaskWidget("task-context"))
	assert.False(t, IsTaskWidget("model"))
	assert.False(t, IsTaskWidget("no-such-widget"))
	assert.True(t, IsSharedWidget("separator"))
	assert.True(t, IsSharedWidget("custom-text"))
	assert.False(t, IsSharedWidget("model"))
	assert.False(t, IsSharedWidget("task-name"))
}

func TestShortModelName(t *testing.T) {
	tests := map[string]string{
		"claude-sonnet-5-5":          "Sonnet 5.5",
		"claude-opus-4-7[1m]":        "Opus 4.7",
		"claude-haiku-4-5-20251001":  "Haiku 4.5",
		"claude-3-5-sonnet-20241022": "Sonnet 3.5",
		"claude-fable-5-1":           "Fable 5.1",
		"claude-opus":                "Opus",
		"gpt-4o":                     "gpt-4o",
		"":                           "",
	}
	for id, want := range tests {
		assert.Equal(t, want, shortModelName(id), id)
	}
}

func TestFormatElapsed(t *testing.T) {
	assert.Equal(t, "0s", formatElapsed(0))
	assert.Equal(t, "45s", formatElapsed(45*time.Second))
	assert.Equal(t, "1m05s", formatElapsed(65*time.Second))
	assert.Equal(t, "1h02m", formatElapsed(time.Hour+2*time.Minute+30*time.Second))
}

func TestSparkline(t *testing.T) {
	tests := []struct {
		name    string
		samples []int
		want    string
	}{
		{"no samples", nil, ""},
		{"one sample has no step", []int{5}, ""},
		{"no growth is flat", []int{7, 7, 7, 7}, "▁▁▁"},
		{"steady growth is even", []int{0, 50, 100}, "██"},
		{"an inherited context does not fill the bar", []int{200_000, 201_000, 203_000, 203_000}, "▅█▁"},
		{"any growth sits above an idle tick", []int{0, 1, 1000}, "▂█"},
		{"a compaction counts as no growth", []int{5000, 1000, 3000}, "▁█"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, sparkline(tt.samples))
		})
	}
}

func TestCompactTokens(t *testing.T) {
	assert.Equal(t, "32k", compactTokens(32_000))
	assert.Equal(t, "1.5k", compactTokens(1500))
	assert.Equal(t, "900", compactTokens(900))
	assert.Equal(t, "2.0M", compactTokens(2_000_000))
}
