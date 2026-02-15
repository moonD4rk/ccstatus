package widget

import (
	"testing"

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
		data     *status.StatusJSON
		expected string
	}{
		{
			name: "display name",
			item: config.WidgetItem{Type: "model"},
			data: &status.StatusJSON{
				Model: status.ModelField{ID: "claude-sonnet-4-5", DisplayName: "Sonnet"},
			},
			expected: "Sonnet",
		},
		{
			name: "raw value returns ID",
			item: config.WidgetItem{Type: "model", RawValue: true},
			data: &status.StatusJSON{
				Model: status.ModelField{ID: "claude-sonnet-4-5", DisplayName: "Sonnet"},
			},
			expected: "claude-sonnet-4-5",
		},
		{
			name: "fallback to ID when no display name",
			item: config.WidgetItem{Type: "model"},
			data: &status.StatusJSON{
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
		ctx := RenderContext{Data: &status.StatusJSON{Version: "1.0.80"}}
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
		ctx := RenderContext{Data: &status.StatusJSON{
			ContextWindow: &status.ContextWindow{TotalInputTokens: intPtr(50_000)},
		}}
		assert.Equal(t, "50.0k", w.Render(&item, ctx, &settings))
		assert.Equal(t, defaultDimColor, w.DefaultColor())
	})

	t.Run("tokens-input nil context window", func(t *testing.T) {
		w := Get("tokens-input")
		ctx := RenderContext{Data: &status.StatusJSON{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-input nil data", func(t *testing.T) {
		w := Get("tokens-input")
		ctx := RenderContext{Data: nil}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-output formats output tokens", func(t *testing.T) {
		w := Get("tokens-output")
		ctx := RenderContext{Data: &status.StatusJSON{
			ContextWindow: &status.ContextWindow{TotalOutputTokens: intPtr(1_200_000)},
		}}
		assert.Equal(t, "1.2M", w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-output nil returns empty", func(t *testing.T) {
		w := Get("tokens-output")
		ctx := RenderContext{Data: &status.StatusJSON{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-cached formats cached tokens", func(t *testing.T) {
		w := Get("tokens-cached")
		ctx := RenderContext{Data: &status.StatusJSON{
			ContextWindow: &status.ContextWindow{
				CurrentUsage: &status.CurrentUsage{CacheReadInputTokens: 8000},
			},
		}}
		assert.Equal(t, "8.0k", w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-cached zero returns empty", func(t *testing.T) {
		w := Get("tokens-cached")
		ctx := RenderContext{Data: &status.StatusJSON{
			ContextWindow: &status.ContextWindow{
				CurrentUsage: &status.CurrentUsage{CacheReadInputTokens: 0},
			},
		}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-cached nil current usage", func(t *testing.T) {
		w := Get("tokens-cached")
		ctx := RenderContext{Data: &status.StatusJSON{
			ContextWindow: &status.ContextWindow{},
		}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-total sums input and output", func(t *testing.T) {
		w := Get("tokens-total")
		ctx := RenderContext{Data: &status.StatusJSON{
			ContextWindow: &status.ContextWindow{
				TotalInputTokens:  intPtr(30_000),
				TotalOutputTokens: intPtr(20_000),
			},
		}}
		assert.Equal(t, "50.0k", w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-total input only", func(t *testing.T) {
		w := Get("tokens-total")
		ctx := RenderContext{Data: &status.StatusJSON{
			ContextWindow: &status.ContextWindow{
				TotalInputTokens: intPtr(500),
			},
		}}
		assert.Equal(t, "500", w.Render(&item, ctx, &settings))
	})

	t.Run("tokens-total both zero returns empty", func(t *testing.T) {
		w := Get("tokens-total")
		ctx := RenderContext{Data: &status.StatusJSON{
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
		ctx := RenderContext{Data: &status.StatusJSON{
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
		ctx := RenderContext{Data: &status.StatusJSON{
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
	w := &ContextPercentageWidget{}
	settings := config.DefaultSettings()

	t.Run("formatted percentage", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.StatusJSON{
			ContextWindow: &status.ContextWindow{UsedPercentage: floatPtr(25.7)},
		}}
		assert.Equal(t, "26%", w.Render(&item, ctx, &settings))
	})

	t.Run("raw value omits percent sign", func(t *testing.T) {
		item := config.WidgetItem{RawValue: true}
		ctx := RenderContext{Data: &status.StatusJSON{
			ContextWindow: &status.ContextWindow{UsedPercentage: floatPtr(25.7)},
		}}
		assert.Equal(t, "25.7", w.Render(&item, ctx, &settings))
	})

	t.Run("zero returns empty", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.StatusJSON{}}
		assert.Empty(t, w.Render(&item, ctx, &settings))
	})

	assert.True(t, w.SupportsRawValue())
}

func TestContextPercentageUsableWidget(t *testing.T) {
	w := &ContextPercentageUsableWidget{}
	settings := config.DefaultSettings()

	t.Run("percentage of usable window", func(t *testing.T) {
		item := config.WidgetItem{}
		ctx := RenderContext{Data: &status.StatusJSON{
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
		ctx := RenderContext{Data: &status.StatusJSON{
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
		ctx := RenderContext{Data: &status.StatusJSON{
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

func TestTypes(t *testing.T) {
	types := Types()
	require.NotEmpty(t, types)
	assert.Contains(t, types, "model")
	assert.Contains(t, types, "separator")
	assert.Contains(t, types, "tokens-input")
	assert.Contains(t, types, "context-percentage")
	assert.Contains(t, types, "flex-separator")
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
