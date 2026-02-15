package widget

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

func intPtr(v int) *int           { return &v }
func floatPtr(v float64) *float64 { return &v }

func TestGet(t *testing.T) {
	tests := []struct {
		widgetType string
		wantNil    bool
	}{
		{"model", false},
		{"version", false},
		{"git-branch", false},
		{"custom-text", false},
		{"separator", false},
		{"tokens-input", false},
		{"tokens-output", false},
		{"tokens-cached", false},
		{"tokens-total", false},
		{"context-length", false},
		{"context-percentage", false},
		{"context-percentage-usable", false},
		{"flex-separator", false},
		{"git-changes", false},
		{"current-working-dir", false},
		{"session-cost", false},
		{"session-clock", false},
		{"lines-changed", false},
		{"lines-added", false},
		{"lines-removed", false},
		{"remaining-percentage", false},
		{"api-duration", false},
		{"project-dir", false},
		{"transcript-path", false},
		{"current-usage-input", false},
		{"current-usage-output", false},
		{"cache-creation", false},
		{"output-style", false},
		{"session-id", false},
		{"terminal-width", false},
		{"vim-mode", false},
		{"agent-name", false},
		{"exceeds-200k", false},
		{"custom-command", false},
		{"git-worktree", false},
		{"block-timer", false},
		{"nonexistent", true},
	}

	for _, tt := range tests {
		t.Run(tt.widgetType, func(t *testing.T) {
			w := Get(tt.widgetType)
			if tt.wantNil {
				assert.Nil(t, w)
			} else {
				assert.NotNil(t, w)
			}
		})
	}
}

func TestModelWidget(t *testing.T) {
	w := &ModelWidget{}
	settings := config.DefaultSettings()

	tests := []struct {
		name     string
		item     config.WidgetItem
		data     *status.Session
		expected string
	}{
		{
			name: "display name",
			item: config.WidgetItem{Type: "model"},
			data: &status.Session{
				Model: status.ModelField{ID: "claude-sonnet-4-5", DisplayName: "Sonnet"},
			},
			expected: "Sonnet",
		},
		{
			name: "raw value returns ID",
			item: config.WidgetItem{Type: "model", RawValue: true},
			data: &status.Session{
				Model: status.ModelField{ID: "claude-sonnet-4-5", DisplayName: "Sonnet"},
			},
			expected: "claude-sonnet-4-5",
		},
		{
			name: "fallback to ID when no display name",
			item: config.WidgetItem{Type: "model"},
			data: &status.Session{
				Model: status.ModelField{ID: "custom-model"},
			},
			expected: "custom-model",
		},
		{
			name:     "nil data returns empty",
			item:     config.WidgetItem{Type: "model"},
			data:     nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := RenderContext{Data: tt.data}
			assert.Equal(t, tt.expected, w.Render(&tt.item, ctx, &settings))
		})
	}

	assert.Equal(t, "cyan", w.DefaultColor())
	assert.True(t, w.SupportsRawValue())
}

func TestVersionWidget(t *testing.T) {
	w := &VersionWidget{}
	settings := config.DefaultSettings()

	t.Run("returns version", func(t *testing.T) {
		ctx := RenderContext{Data: &status.Session{Version: "1.0.80"}}
		item := config.WidgetItem{}
		assert.Equal(t, "1.0.80", w.Render(&item, ctx, &settings))
	})

	t.Run("nil data returns empty", func(t *testing.T) {
		ctx := RenderContext{Data: nil}
		item := config.WidgetItem{}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})
}

func TestCustomTextWidget(t *testing.T) {
	w := &CustomTextWidget{}
	settings := config.DefaultSettings()
	ctx := RenderContext{}

	t.Run("returns custom text", func(t *testing.T) {
		item := config.WidgetItem{CustomText: "Hello World"}
		assert.Equal(t, "Hello World", w.Render(&item, ctx, &settings))
	})

	t.Run("empty custom text", func(t *testing.T) {
		item := config.WidgetItem{}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})
}

func TestSeparatorWidget(t *testing.T) {
	w := &SeparatorWidget{}
	ctx := RenderContext{}

	tests := []struct {
		name     string
		item     config.WidgetItem
		settings config.Settings
		expected string
	}{
		{
			name:     "uses widget character",
			item:     config.WidgetItem{Character: "/"},
			settings: config.Settings{DefaultSeparator: "|"},
			expected: "/",
		},
		{
			name:     "falls back to default separator",
			item:     config.WidgetItem{},
			settings: config.Settings{DefaultSeparator: "::"},
			expected: "::",
		},
		{
			name:     "falls back to pipe when no default",
			item:     config.WidgetItem{},
			settings: config.Settings{},
			expected: "|",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, w.Render(&tt.item, ctx, &tt.settings))
		})
	}
}

func TestTokenWidgets(t *testing.T) {
	settings := config.DefaultSettings()
	item := config.WidgetItem{}

	t.Run("tokens-input formats input tokens", func(t *testing.T) {
		w := Get("tokens-input")
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{TotalInputTokens: intPtr(50_000)},
		}}
		assert.Equal(t, "50.0k", w.Render(&item, ctx, &settings))
		assert.Equal(t, defaultDimColor, w.DefaultColor())
	})

	t.Run("tokens-input nil context window", func(t *testing.T) {
		w := Get("tokens-input")
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-input nil data", func(t *testing.T) {
		w := Get("tokens-input")
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-output formats output tokens", func(t *testing.T) {
		w := Get("tokens-output")
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{TotalOutputTokens: intPtr(1_200_000)},
		}}
		assert.Equal(t, "1.2M", w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-output nil returns empty", func(t *testing.T) {
		w := Get("tokens-output")
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-cached formats cached tokens", func(t *testing.T) {
		w := Get("tokens-cached")
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				CurrentUsage: &status.CurrentUsage{CacheReadInputTokens: 8000},
			},
		}}
		assert.Equal(t, "8.0k", w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-cached zero returns empty", func(t *testing.T) {
		w := Get("tokens-cached")
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				CurrentUsage: &status.CurrentUsage{CacheReadInputTokens: 0},
			},
		}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-cached nil current usage", func(t *testing.T) {
		w := Get("tokens-cached")
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{},
		}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-total sums input and output", func(t *testing.T) {
		w := Get("tokens-total")
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				TotalInputTokens:  intPtr(30_000),
				TotalOutputTokens: intPtr(20_000),
			},
		}}
		assert.Equal(t, "50.0k", w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-total input only", func(t *testing.T) {
		w := Get("tokens-total")
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				TotalInputTokens: intPtr(500),
			},
		}}
		assert.Equal(t, "500", w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-total both zero returns empty", func(t *testing.T) {
		w := Get("tokens-total")
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				TotalInputTokens:  intPtr(0),
				TotalOutputTokens: intPtr(0),
			},
		}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})
}

func TestContextLengthWidget(t *testing.T) {
	w := &ContextLengthWidget{}
	settings := config.DefaultSettings()
	item := config.WidgetItem{}

	t.Run("formats context length", func(t *testing.T) {
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				CurrentUsage: &status.CurrentUsage{
					InputTokens:              40_000,
					CacheCreationInputTokens: 5000,
					CacheReadInputTokens:     5000,
				},
			},
		}}
		assert.Equal(t, "50.0k", w.Render(&item, ctx, &settings))
	})

	t.Run("zero returns empty", func(t *testing.T) {
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				CurrentUsage: &status.CurrentUsage{},
			},
		}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("nil data returns empty", func(t *testing.T) {
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.Equal(t, "brightBlack", w.DefaultColor())
}

func TestContextPercentageWidget(t *testing.T) {
	w := Get("context-percentage")
	settings := config.DefaultSettings()

	t.Run("formatted percentage", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{UsedPercentage: floatPtr(25.7)},
		}}
		assert.Equal(t, "26%", w.Render(&item, ctx, &settings))
	})

	t.Run("raw value omits percent sign", func(t *testing.T) {
		item := config.WidgetItem{RawValue: true}
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{UsedPercentage: floatPtr(25.7)},
		}}
		assert.Equal(t, "25.7", w.Render(&item, ctx, &settings))
	})

	t.Run("zero returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.True(t, w.SupportsRawValue())
}

func TestContextPercentageUsableWidget(t *testing.T) {
	w := &ContextPercentageUsableWidget{}
	settings := config.DefaultSettings()

	t.Run("percentage of usable window", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				ContextWindowSize: intPtr(200_000),
				CurrentUsage: &status.CurrentUsage{
					InputTokens: 80_000,
				},
			},
		}}
		// usable = 160_000, pct = 80000/160000*100 = 50%
		assert.Equal(t, "50%", w.Render(&item, ctx, &settings))
	})

	t.Run("raw value", func(t *testing.T) {
		item := config.WidgetItem{RawValue: true}
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				ContextWindowSize: intPtr(200_000),
				CurrentUsage: &status.CurrentUsage{
					InputTokens: 80_000,
				},
			},
		}}
		assert.Equal(t, "50.0", w.Render(&item, ctx, &settings))
	})

	t.Run("capped at 100", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				ContextWindowSize: intPtr(200_000),
				CurrentUsage: &status.CurrentUsage{
					InputTokens: 200_000,
				},
			},
		}}
		// usable = 160_000, pct = 200000/160000*100 = 125% -> capped to 100
		assert.Equal(t, "100%", w.Render(&item, ctx, &settings))
	})

	t.Run("nil data returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.True(t, w.SupportsRawValue())
}

func TestFlexSeparatorWidget(t *testing.T) {
	w := &FlexSeparatorWidget{}
	settings := config.DefaultSettings()
	item := config.WidgetItem{Type: "flex-separator"}
	ctx := RenderContext{}

	assert.Equal(t, "flex-separator", w.Render(&item, ctx, &settings))
	assert.Empty(t, w.DefaultColor())
	assert.False(t, w.SupportsRawValue())
}

func TestCurrentDirWidget(t *testing.T) {
	w := &CurrentDirWidget{}
	settings := config.DefaultSettings()

	t.Run("base dir name", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			Workspace: &status.Workspace{CurrentDir: "/home/user/projects/myapp"},
		}}
		assert.Equal(t, "myapp", w.Render(&item, ctx, &settings))
	})

	t.Run("raw value returns path outside home", func(t *testing.T) {
		item := config.WidgetItem{RawValue: true}
		ctx := RenderContext{Data: &status.Session{
			Workspace: &status.Workspace{CurrentDir: "/opt/projects/myapp"},
		}}
		assert.Equal(t, "/opt/projects/myapp", w.Render(&item, ctx, &settings))
	})

	t.Run("raw value shortens home dir", func(t *testing.T) {
		item := config.WidgetItem{RawValue: true}
		home, _ := os.UserHomeDir()
		ctx := RenderContext{Data: &status.Session{
			Workspace: &status.Workspace{CurrentDir: home + "/projects/myapp"},
		}}
		assert.Equal(t, "~/projects/myapp", w.Render(&item, ctx, &settings))
	})

	t.Run("falls back to cwd", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{Cwd: "/tmp/test"}}
		assert.Equal(t, "test", w.Render(&item, ctx, &settings))
	})

	t.Run("nil data returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.Equal(t, "blue", w.DefaultColor())
	assert.True(t, w.SupportsRawValue())
}

func TestSessionCostWidget(t *testing.T) {
	w := &SessionCostWidget{}
	settings := config.DefaultSettings()

	t.Run("formats cost with dollar sign", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			Cost: &status.CostInfo{TotalCostUSD: floatPtr(1.23)},
		}}
		assert.Equal(t, "$1.23", w.Render(&item, ctx, &settings))
	})

	t.Run("small cost uses 4 decimals", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			Cost: &status.CostInfo{TotalCostUSD: floatPtr(0.0012)},
		}}
		assert.Equal(t, "$0.0012", w.Render(&item, ctx, &settings))
	})

	t.Run("raw value omits dollar sign", func(t *testing.T) {
		item := config.WidgetItem{RawValue: true}
		ctx := RenderContext{Data: &status.Session{
			Cost: &status.CostInfo{TotalCostUSD: floatPtr(1.23)},
		}}
		assert.Equal(t, "1.2300", w.Render(&item, ctx, &settings))
	})

	t.Run("nil cost returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.Equal(t, "green", w.DefaultColor())
	assert.True(t, w.SupportsRawValue())
}

func TestSessionClockWidget(t *testing.T) {
	w := &SessionClockWidget{}
	settings := config.DefaultSettings()

	tests := []struct {
		name     string
		ms       float64
		expected string
	}{
		{"less than a minute", 30_000, "<1m"},
		{"minutes only", 300_000, "5m"},
		{"hours and minutes", 5_460_000, "1h31m"},
		{"hours only", 3_600_000, "1h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := config.WidgetItem{}
			ctx := RenderContext{Data: &status.Session{
				Cost: &status.CostInfo{TotalDurationMS: floatPtr(tt.ms)},
			}}
			assert.Equal(t, tt.expected, w.Render(&item, ctx, &settings))
		})
	}

	t.Run("nil cost returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("raw value returns ms", func(t *testing.T) {
		item := config.WidgetItem{RawValue: true}
		ctx := RenderContext{Data: &status.Session{
			Cost: &status.CostInfo{TotalDurationMS: floatPtr(300_000)},
		}}
		assert.Equal(t, "300000", w.Render(&item, ctx, &settings))
	})

	assert.Equal(t, defaultDimColor, w.DefaultColor())
	assert.True(t, w.SupportsRawValue())
}

func TestLinesChangedWidget(t *testing.T) {
	w := &LinesChangedWidget{}

	t.Run("returns git diff format or empty", func(t *testing.T) {
		// Widget calls real git commands; result depends on working tree state.
		// Verify output is either empty or matches +N/-M format.
		settings := config.DefaultSettings()
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		result := w.Render(&item, ctx, &settings)
		if result != "" {
			assert.Regexp(t, `^\+\d+/-\d+$`, result)
		}
	})

	assert.Equal(t, "green", w.DefaultColor())
	assert.Equal(t, "Lines Changed", w.DisplayName())
	assert.False(t, w.SupportsRawValue())
}

func TestLinesAddedWidget(t *testing.T) {
	w := &LinesAddedWidget{}

	t.Run("returns git diff format or empty", func(t *testing.T) {
		settings := config.DefaultSettings()
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		result := w.Render(&item, ctx, &settings)
		if result != "" {
			assert.Regexp(t, `^\+\d+$`, result)
		}
	})

	assert.Equal(t, "green", w.DefaultColor())
}

func TestLinesRemovedWidget(t *testing.T) {
	w := &LinesRemovedWidget{}

	t.Run("returns git diff format or empty", func(t *testing.T) {
		settings := config.DefaultSettings()
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		result := w.Render(&item, ctx, &settings)
		if result != "" {
			assert.Regexp(t, `^-\d+$`, result)
		}
	})

	assert.Equal(t, "red", w.DefaultColor())
}

func TestTypes(t *testing.T) {
	types := Types()
	require.NotEmpty(t, types)
	assert.Contains(t, types, "model")
	assert.Contains(t, types, "separator")
	assert.Contains(t, types, "tokens-input")
	assert.Contains(t, types, "context-percentage")
	assert.Contains(t, types, "flex-separator")
	assert.Contains(t, types, "git-changes")
	assert.Contains(t, types, "current-working-dir")
	assert.Contains(t, types, "session-cost")
	assert.Contains(t, types, "session-clock")
	assert.Contains(t, types, "lines-changed")
	assert.Contains(t, types, "lines-added")
	assert.Contains(t, types, "lines-removed")
	assert.Contains(t, types, "remaining-percentage")
	assert.Contains(t, types, "api-duration")
	assert.Contains(t, types, "project-dir")
	assert.Contains(t, types, "transcript-path")
	assert.Contains(t, types, "current-usage-input")
	assert.Contains(t, types, "current-usage-output")
	assert.Contains(t, types, "cache-creation")
	assert.Contains(t, types, "output-style")
	assert.Contains(t, types, "session-id")
	assert.Contains(t, types, "terminal-width")
	assert.Contains(t, types, "vim-mode")
	assert.Contains(t, types, "agent-name")
	assert.Contains(t, types, "exceeds-200k")
	assert.Contains(t, types, "custom-command")
	assert.Contains(t, types, "git-worktree")
	assert.Contains(t, types, "block-timer")
}

func TestRemainingPercentageWidget(t *testing.T) {
	w := Get("remaining-percentage")
	settings := config.DefaultSettings()

	t.Run("formatted percentage", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{RemainingPercentage: floatPtr(74.3)},
		}}
		assert.Equal(t, "74%", w.Render(&item, ctx, &settings))
	})

	t.Run("raw value omits percent sign", func(t *testing.T) {
		item := config.WidgetItem{RawValue: true}
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{RemainingPercentage: floatPtr(74.3)},
		}}
		assert.Equal(t, "74.3", w.Render(&item, ctx, &settings))
	})

	t.Run("zero returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("nil data returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.Equal(t, defaultDimColor, w.DefaultColor())
	assert.True(t, w.SupportsRawValue())
}

func TestAPIDurationWidget(t *testing.T) {
	w := &APIDurationWidget{}
	settings := config.DefaultSettings()

	tests := []struct {
		name     string
		ms       float64
		expected string
	}{
		{"less than a minute", 30_000, "<1m"},
		{"minutes only", 300_000, "5m"},
		{"hours and minutes", 5_460_000, "1h31m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := config.WidgetItem{}
			ctx := RenderContext{Data: &status.Session{
				Cost: &status.CostInfo{TotalAPIDurationMS: floatPtr(tt.ms)},
			}}
			assert.Equal(t, tt.expected, w.Render(&item, ctx, &settings))
		})
	}

	t.Run("nil cost returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("nil data returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("raw value returns ms", func(t *testing.T) {
		item := config.WidgetItem{RawValue: true}
		ctx := RenderContext{Data: &status.Session{
			Cost: &status.CostInfo{TotalAPIDurationMS: floatPtr(2300)},
		}}
		assert.Equal(t, "2300", w.Render(&item, ctx, &settings))
	})

	assert.Equal(t, defaultDimColor, w.DefaultColor())
	assert.True(t, w.SupportsRawValue())
}

func TestProjectDirWidget(t *testing.T) {
	w := &ProjectDirWidget{}
	settings := config.DefaultSettings()

	t.Run("base dir name", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			Workspace: &status.Workspace{ProjectDir: "/home/user/projects/myapp"},
		}}
		assert.Equal(t, "myapp", w.Render(&item, ctx, &settings))
	})

	t.Run("raw value returns path outside home", func(t *testing.T) {
		item := config.WidgetItem{RawValue: true}
		ctx := RenderContext{Data: &status.Session{
			Workspace: &status.Workspace{ProjectDir: "/opt/projects/myapp"},
		}}
		assert.Equal(t, "/opt/projects/myapp", w.Render(&item, ctx, &settings))
	})

	t.Run("raw value shortens home dir", func(t *testing.T) {
		item := config.WidgetItem{RawValue: true}
		home, _ := os.UserHomeDir()
		ctx := RenderContext{Data: &status.Session{
			Workspace: &status.Workspace{ProjectDir: home + "/projects/myapp"},
		}}
		assert.Equal(t, "~/projects/myapp", w.Render(&item, ctx, &settings))
	})

	t.Run("empty project dir returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			Workspace: &status.Workspace{},
		}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("nil workspace returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("nil data returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.Equal(t, "blue", w.DefaultColor())
	assert.True(t, w.SupportsRawValue())
}

func TestTranscriptPathWidget(t *testing.T) {
	w := &TranscriptPathWidget{}
	settings := config.DefaultSettings()

	t.Run("base file name", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			TranscriptPath: "/tmp/session/transcript.jsonl",
		}}
		assert.Equal(t, "transcript.jsonl", w.Render(&item, ctx, &settings))
	})

	t.Run("raw value returns path outside home", func(t *testing.T) {
		item := config.WidgetItem{RawValue: true}
		ctx := RenderContext{Data: &status.Session{
			TranscriptPath: "/tmp/session/transcript.jsonl",
		}}
		assert.Equal(t, "/tmp/session/transcript.jsonl", w.Render(&item, ctx, &settings))
	})

	t.Run("raw value shortens home dir", func(t *testing.T) {
		item := config.WidgetItem{RawValue: true}
		home, _ := os.UserHomeDir()
		ctx := RenderContext{Data: &status.Session{
			TranscriptPath: home + "/.claude/transcript.jsonl",
		}}
		assert.Equal(t, "~/.claude/transcript.jsonl", w.Render(&item, ctx, &settings))
	})

	t.Run("empty path returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("nil data returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.Equal(t, defaultDimColor, w.DefaultColor())
	assert.True(t, w.SupportsRawValue())
}

func TestCurrentUsageTokenWidgets(t *testing.T) {
	settings := config.DefaultSettings()
	item := config.WidgetItem{}

	t.Run("current-usage-input formats tokens", func(t *testing.T) {
		w := Get("current-usage-input")
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				CurrentUsage: &status.CurrentUsage{InputTokens: 8500},
			},
		}}
		assert.Equal(t, "8.5k", w.Render(&item, ctx, &settings))
	})

	t.Run("current-usage-input nil current usage", func(t *testing.T) {
		w := Get("current-usage-input")
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{},
		}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("current-usage-input nil data", func(t *testing.T) {
		w := Get("current-usage-input")
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("current-usage-output formats tokens", func(t *testing.T) {
		w := Get("current-usage-output")
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				CurrentUsage: &status.CurrentUsage{OutputTokens: 1200},
			},
		}}
		assert.Equal(t, "1.2k", w.Render(&item, ctx, &settings))
	})

	t.Run("current-usage-output zero returns empty", func(t *testing.T) {
		w := Get("current-usage-output")
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				CurrentUsage: &status.CurrentUsage{OutputTokens: 0},
			},
		}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("cache-creation formats tokens", func(t *testing.T) {
		w := Get("cache-creation")
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				CurrentUsage: &status.CurrentUsage{CacheCreationInputTokens: 5000},
			},
		}}
		assert.Equal(t, "5.0k", w.Render(&item, ctx, &settings))
	})

	t.Run("cache-creation zero returns empty", func(t *testing.T) {
		w := Get("cache-creation")
		ctx := RenderContext{Data: &status.Session{
			ContextWindow: &status.ContextWindow{
				CurrentUsage: &status.CurrentUsage{CacheCreationInputTokens: 0},
			},
		}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("cache-creation nil context window", func(t *testing.T) {
		w := Get("cache-creation")
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})
}

func TestRegister(t *testing.T) {
	original := Get("test-widget")
	defer func() {
		if original == nil {
			delete(registry, "test-widget")
		} else {
			registry["test-widget"] = original
		}
	}()

	assert.Nil(t, Get("test-widget"))
	Register("test-widget", &CustomTextWidget{})
	assert.NotNil(t, Get("test-widget"))
}

func boolPtr(v bool) *bool { return &v }

func TestOutputStyleWidget(t *testing.T) {
	w := Get("output-style")
	settings := config.DefaultSettings()

	t.Run("returns style name", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			OutputStyle: &status.OutputStyle{Name: "text"},
		}}
		assert.Equal(t, "text", w.Render(&item, ctx, &settings))
	})

	t.Run("stream-json style", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			OutputStyle: &status.OutputStyle{Name: "stream-json"},
		}}
		assert.Equal(t, "stream-json", w.Render(&item, ctx, &settings))
	})

	t.Run("nil output style returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("nil data returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.Equal(t, defaultDimColor, w.DefaultColor())
	assert.False(t, w.SupportsRawValue())
}

func TestSessionIDWidget(t *testing.T) {
	w := &SessionIDWidget{}
	settings := config.DefaultSettings()

	t.Run("truncated by default", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			SessionID: "abc12345-6789-0abc-def0-123456789abc",
		}}
		assert.Equal(t, "abc12345", w.Render(&item, ctx, &settings))
	})

	t.Run("raw value returns full UUID", func(t *testing.T) {
		item := config.WidgetItem{RawValue: true}
		ctx := RenderContext{Data: &status.Session{
			SessionID: "abc12345-6789-0abc-def0-123456789abc",
		}}
		assert.Equal(t, "abc12345-6789-0abc-def0-123456789abc", w.Render(&item, ctx, &settings))
	})

	t.Run("short ID not truncated", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			SessionID: "abc",
		}}
		assert.Equal(t, "abc", w.Render(&item, ctx, &settings))
	})

	t.Run("empty session ID returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("nil data returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.Equal(t, defaultDimColor, w.DefaultColor())
	assert.True(t, w.SupportsRawValue())
}

func TestTerminalWidthWidget(t *testing.T) {
	w := &TerminalWidthWidget{}
	settings := config.DefaultSettings()
	item := config.WidgetItem{}

	t.Run("returns width as string", func(t *testing.T) {
		ctx := RenderContext{TerminalWidth: 120}
		assert.Equal(t, "120", w.Render(&item, ctx, &settings))
	})

	t.Run("zero width returns empty", func(t *testing.T) {
		ctx := RenderContext{TerminalWidth: 0}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("negative width returns empty", func(t *testing.T) {
		ctx := RenderContext{TerminalWidth: -1}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.Equal(t, defaultDimColor, w.DefaultColor())
	assert.False(t, w.SupportsRawValue())
}

func TestVimModeWidget(t *testing.T) {
	w := Get("vim-mode")
	settings := config.DefaultSettings()
	item := config.WidgetItem{}

	t.Run("returns NORMAL mode", func(t *testing.T) {
		ctx := RenderContext{Data: &status.Session{
			Vim: &status.VimInfo{Mode: "NORMAL"},
		}}
		assert.Equal(t, "NORMAL", w.Render(&item, ctx, &settings))
	})

	t.Run("returns INSERT mode", func(t *testing.T) {
		ctx := RenderContext{Data: &status.Session{
			Vim: &status.VimInfo{Mode: "INSERT"},
		}}
		assert.Equal(t, "INSERT", w.Render(&item, ctx, &settings))
	})

	t.Run("nil vim returns empty", func(t *testing.T) {
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("nil data returns empty", func(t *testing.T) {
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.Equal(t, "yellow", w.DefaultColor())
	assert.False(t, w.SupportsRawValue())
}

func TestAgentNameWidget(t *testing.T) {
	w := Get("agent-name")
	settings := config.DefaultSettings()
	item := config.WidgetItem{}

	t.Run("returns agent name", func(t *testing.T) {
		ctx := RenderContext{Data: &status.Session{
			Agent: &status.AgentInfo{Name: "security-reviewer"},
		}}
		assert.Equal(t, "security-reviewer", w.Render(&item, ctx, &settings))
	})

	t.Run("nil agent returns empty", func(t *testing.T) {
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("nil data returns empty", func(t *testing.T) {
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.Equal(t, "cyan", w.DefaultColor())
	assert.False(t, w.SupportsRawValue())
}

func TestExceeds200KWidget(t *testing.T) {
	w := &Exceeds200KWidget{}
	settings := config.DefaultSettings()
	item := config.WidgetItem{}

	t.Run("true returns warning", func(t *testing.T) {
		ctx := RenderContext{Data: &status.Session{
			Exceeds200K: boolPtr(true),
		}}
		assert.Equal(t, ">200k", w.Render(&item, ctx, &settings))
	})

	t.Run("false returns empty", func(t *testing.T) {
		ctx := RenderContext{Data: &status.Session{
			Exceeds200K: boolPtr(false),
		}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("nil returns empty", func(t *testing.T) {
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("nil data returns empty", func(t *testing.T) {
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.Equal(t, "red", w.DefaultColor())
	assert.False(t, w.SupportsRawValue())
}

func TestCustomCommandWidget(t *testing.T) {
	w := &CustomCommandWidget{}
	settings := config.DefaultSettings()

	t.Run("executes echo command", func(t *testing.T) {
		item := config.WidgetItem{CommandPath: "echo hello"}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Equal(t, "hello", w.Render(&item, ctx, &settings))
	})

	t.Run("takes first line only", func(t *testing.T) {
		item := config.WidgetItem{CommandPath: "printf 'line1\nline2'"}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Equal(t, "line1", w.Render(&item, ctx, &settings))
	})

	t.Run("applies maxWidth", func(t *testing.T) {
		item := config.WidgetItem{CommandPath: "echo longstring", MaxWidth: 4}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Equal(t, "long", w.Render(&item, ctx, &settings))
	})

	t.Run("strips ANSI by default", func(t *testing.T) {
		item := config.WidgetItem{CommandPath: `printf '\033[32mgreen\033[0m'`}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Equal(t, "green", w.Render(&item, ctx, &settings))
	})

	t.Run("preserves ANSI when configured", func(t *testing.T) {
		item := config.WidgetItem{
			CommandPath:    `printf '\033[32mgreen\033[0m'`,
			PreserveColors: true,
		}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Contains(t, w.Render(&item, ctx, &settings), "green")
	})

	t.Run("empty commandPath returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("failing command returns empty", func(t *testing.T) {
		item := config.WidgetItem{CommandPath: "false"}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("pipes JSON to stdin", func(t *testing.T) {
		item := config.WidgetItem{CommandPath: "cat | jq -r .version 2>/dev/null || echo fallback"}
		ctx := RenderContext{Data: &status.Session{Version: "1.0.80"}}
		result := w.Render(&item, ctx, &settings)
		// jq may not be installed; accept either the parsed value or fallback.
		assert.True(t, result == "1.0.80" || result == "fallback",
			"expected '1.0.80' or 'fallback', got %q", result)
	})

	assert.Equal(t, "white", w.DefaultColor())
	assert.False(t, w.SupportsRawValue())
}

func TestBlockTimerWidget(t *testing.T) {
	w := &BlockTimerWidget{}
	settings := config.DefaultSettings()

	t.Run("time mode from duration_ms", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			Cost: &status.CostInfo{TotalDurationMS: floatPtr(5_400_000)}, // 1h30m
		}}
		assert.Equal(t, "1h30m/5h", w.Render(&item, ctx, &settings))
	})

	t.Run("time mode less than a minute", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			Cost: &status.CostInfo{TotalDurationMS: floatPtr(30_000)},
		}}
		assert.Equal(t, "<1m/5h", w.Render(&item, ctx, &settings))
	})

	t.Run("time mode hours only", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			Cost: &status.CostInfo{TotalDurationMS: floatPtr(7_200_000)}, // 2h
		}}
		assert.Equal(t, "2h/5h", w.Render(&item, ctx, &settings))
	})

	t.Run("clamped at 5h", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			Cost: &status.CostInfo{TotalDurationMS: floatPtr(20_000_000)}, // >5h
		}}
		assert.Equal(t, "5h/5h", w.Render(&item, ctx, &settings))
	})

	t.Run("percentage mode", func(t *testing.T) {
		item := config.WidgetItem{Metadata: map[string]string{"display": "percentage"}}
		ctx := RenderContext{Data: &status.Session{
			Cost: &status.CostInfo{TotalDurationMS: floatPtr(9_000_000)}, // 2.5h = 50%
		}}
		assert.Equal(t, "50%", w.Render(&item, ctx, &settings))
	})

	t.Run("progress mode", func(t *testing.T) {
		item := config.WidgetItem{Metadata: map[string]string{"display": "progress"}}
		ctx := RenderContext{Data: &status.Session{
			Cost: &status.CostInfo{TotalDurationMS: floatPtr(9_000_000)}, // 50%
		}}
		result := w.Render(&item, ctx, &settings)
		assert.Contains(t, result, "[")
		assert.Contains(t, result, "]")
		assert.Contains(t, result, "50%")
	})

	t.Run("nil cost returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("nil data returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("transcript fallback with temp file", func(t *testing.T) {
		// Create a temp JSONL file with a timestamp.
		tmpFile, err := os.CreateTemp("", "transcript-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		// Write a JSONL entry with a timestamp 1 hour ago.
		ts := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
		_, err = tmpFile.WriteString(`{"timestamp":"` + ts + `","type":"start"}` + "\n")
		require.NoError(t, err)
		tmpFile.Close()

		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.Session{
			TranscriptPath: tmpFile.Name(),
		}}
		result := w.Render(&item, ctx, &settings)
		// Should show approximately 1h/5h (could be 59m or 1h depending on timing).
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "/5h")
	})

	assert.Equal(t, defaultDimColor, w.DefaultColor())
	assert.False(t, w.SupportsRawValue())
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no ANSI", "hello", "hello"},
		{"color code", "\033[32mgreen\033[0m", "green"},
		{"bold", "\033[1mbold\033[0m", "bold"},
		{"multiple codes", "\033[1;32mbold green\033[0m", "bold green"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, stripANSI(tt.input))
		})
	}
}
