package widget

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

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

func TestTypes(t *testing.T) {
	types := Types()
	require.NotEmpty(t, types)
	assert.Contains(t, types, "model")
	assert.Contains(t, types, "separator")
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
