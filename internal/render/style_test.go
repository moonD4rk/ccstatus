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

func TestRenderLineMaxWidth(t *testing.T) {
	settings := config.Settings{ColorLevel: 0, DefaultPadding: " ", DefaultSeparator: "|"}
	ctx := widget.RenderContext{Data: &status.Session{}}
	tests := []struct {
		name string
		item config.WidgetItem
		want string
	}{
		{"short text is untouched", config.WidgetItem{Type: "custom-text", CustomText: "main", MaxWidth: 10}, "main"},
		{"long text ends in an ellipsis", config.WidgetItem{Type: "custom-text", CustomText: "feat/window-redesign", MaxWidth: 10}, "feat/wind…"},
		{"wide runes count as two columns", config.WidgetItem{Type: "custom-text", CustomText: "Terraform 中的 badgebar 仓库", MaxWidth: 16}, "Terraform 中的…"},
		{"the prefix is not counted", config.WidgetItem{Type: "custom-text", CustomText: "abcdef", Prefix: "S: ", MaxWidth: 4}, "S: abc…"},
		{"command output is capped the same way", config.WidgetItem{Type: "custom-command", CommandPath: "echo longstring", MaxWidth: 4}, "lon…"},
		{"a cut never splits a multibyte rune", config.WidgetItem{Type: "custom-command", CommandPath: `printf 'a\303\251bc'`, MaxWidth: 3}, "aé…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.item.ID = "1"
			assert.Equal(t, tt.want, RenderLine([]config.WidgetItem{tt.item}, &settings, ctx))
		})
	}
}

func TestTruncateWidget(t *testing.T) {
	assert.Equal(t, "abc", truncateWidget("abc", 0), "0 means no limit")
	assert.Equal(t, "abc", truncateWidget("abc", 3))
	assert.Equal(t, "…", truncateWidget("abc", 1))
	link := "\x1b]8;;https://example.com/pull/1234\x07#1234\x1b]8;;\x07"
	assert.Equal(t, "\x1b]8;;https://example.com/pull/1234\x07#1\x1b]8;;\x07…", truncateWidget(link, 3),
		"a cut link is closed before the ellipsis")
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
