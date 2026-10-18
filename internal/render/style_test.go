package render

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
	"github.com/moond4rk/ccstatus/internal/widget"
)

func contextAt(pct float64) widget.RenderContext {
	return widget.RenderContext{Data: &status.Session{ContextWindow: &status.ContextWindow{UsedPercentage: &pct}}}
}

func TestRenderLineThresholds(t *testing.T) {
	settings := config.Settings{ColorLevel: 2, DefaultPadding: " "}
	item := config.WidgetItem{
		ID: "1", Type: "context-percentage", Color: "green",
		Thresholds: []config.Threshold{
			{GTE: 70, Color: "yellow"},
			{GTE: 90, Color: "red", Bold: true},
		},
	}
	tests := []struct {
		pct  float64
		text string
		fg   string
		bold bool
	}{
		{pct: 12, text: "Ctx: 12%", fg: "green"},
		{pct: 70, text: "Ctx: 70%", fg: "yellow"},
		{pct: 95, text: "Ctx: 95%", fg: "red", bold: true},
	}
	for _, tt := range tests {
		got := RenderLine([]config.WidgetItem{item}, &settings, contextAt(tt.pct))
		assert.Equal(t, color.Apply(tt.text, tt.fg, "", tt.bold, 2), got, tt.pct)
	}

	t.Run("global override still wins", func(t *testing.T) {
		s := settings
		s.OverrideForegroundColor = "white"
		got := RenderLine([]config.WidgetItem{item}, &s, contextAt(95))
		assert.Equal(t, color.Apply("Ctx: 95%", "white", "", true, 2), got)
	})

	t.Run("non-numeric widgets ignore thresholds", func(t *testing.T) {
		model := config.WidgetItem{ID: "1", Type: "model", Color: "cyan", Thresholds: item.Thresholds}
		ctx := widget.RenderContext{Data: &status.Session{Model: status.ModelField{DisplayName: "Opus"}}}
		assert.Equal(t, color.Apply("Opus", "cyan", "", false, 2), RenderLine([]config.WidgetItem{model}, &settings, ctx))
	})
}

func TestRenderLineDynamicDefaultColor(t *testing.T) {
	settings := config.Settings{ColorLevel: 2, DefaultPadding: " "}
	items := []config.WidgetItem{{ID: "1", Type: "prompt-cache"}}
	warm, cold := true, false
	expires := time.Now().Add(10 * time.Minute).Unix()

	warmCtx := widget.RenderContext{Data: &status.Session{PromptCache: &status.PromptCache{Warm: &warm, ExpiresAt: &expires}}}
	assert.Equal(t, color.Apply("Cache: warm 9m", "green", "", false, 2), RenderLine(items, &settings, warmCtx))

	coldCtx := widget.RenderContext{Data: &status.Session{PromptCache: &status.PromptCache{Warm: &cold}}}
	assert.Equal(t, color.Apply("Cache: cold", "brightBlack", "", false, 2), RenderLine(items, &settings, coldCtx))

	explicit := []config.WidgetItem{{ID: "1", Type: "prompt-cache", Color: "magenta"}}
	assert.Equal(t, color.Apply("Cache: cold", "magenta", "", false, 2), RenderLine(explicit, &settings, coldCtx))
}

func TestRenderLineMerge(t *testing.T) {
	settings := config.Settings{ColorLevel: 0, DefaultPadding: " ", DefaultSeparator: "|"}
	ctx := widget.RenderContext{Data: &status.Session{
		Model:   status.ModelField{DisplayName: "Opus"},
		Version: "2.1.302",
	}}
	tests := []struct {
		name  string
		merge any
		want  string
	}{
		{"no merge keeps padding", nil, "Opus v2.1.302"},
		{"merge true joins without padding", true, "Opusv2.1.302"},
		{"legacy no-padding behaves the same", "no-padding", "Opusv2.1.302"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := []config.WidgetItem{
				{ID: "1", Type: "model", Merge: tt.merge},
				{ID: "2", Type: "version", Prefix: "v"},
			}
			assert.Equal(t, tt.want, RenderLine(items, &settings, ctx))
		})
	}

	t.Run("merge before a flex separator does not matter", func(t *testing.T) {
		items := []config.WidgetItem{
			{ID: "1", Type: "model", Merge: true},
			{ID: "2", Type: "flex-separator"},
			{ID: "3", Type: "version"},
		}
		flexCtx := ctx
		flexCtx.TerminalWidth = 20
		s := settings
		s.FlexMode = "none"
		assert.Equal(t, "Opus         2.1.302", RenderLine(items, &s, flexCtx))
	})
}
