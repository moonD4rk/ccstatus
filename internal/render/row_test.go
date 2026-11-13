package render

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
	"github.com/moond4rk/ccstatus/internal/widget"
)

func TestRenderRow(t *testing.T) {
	settings := config.Settings{ColorLevel: 0, DefaultPadding: " ", DefaultSeparator: "|", FlexMode: "full-until-compact"}
	task := &status.Task{Name: "reviewer", Status: "running", Label: "Reading internal/auth/session.go"}
	ctx := widget.RenderContext{Task: task}
	items := []config.WidgetItem{
		{Type: "task-name"},
		{Type: "task-status"},
		{Type: "separator"},
		{Type: "task-label"},
	}

	t.Run("uses the full width, with no flex-mode margin", func(t *testing.T) {
		assert.Equal(t, "reviewer running | Reading internal/auth/session.go", RenderRow(items, &settings, ctx, 52))
	})

	t.Run("truncates at the given width", func(t *testing.T) {
		got := RenderRow(items, &settings, ctx, 30)
		assert.Equal(t, 30, color.VisibleWidth(got))
		assert.Equal(t, "reviewer running | Reading ...", color.StripANSI(got))
	})

	t.Run("zero width does not truncate", func(t *testing.T) {
		assert.Equal(t, "reviewer running | Reading internal/auth/session.go", RenderRow(items, &settings, ctx, 0))
	})

	t.Run("flex separator fills exactly the width", func(t *testing.T) {
		flex := []config.WidgetItem{{Type: "task-name"}, {Type: "flex-separator"}, {Type: "task-status"}}
		assert.Equal(t, "reviewer         running", RenderRow(flex, &settings, ctx, 24))
	})

	t.Run("task-status takes its state color", func(t *testing.T) {
		colored := settings
		colored.ColorLevel = 2
		got := RenderRow([]config.WidgetItem{{Type: "task-status"}}, &colored, ctx, 40)
		assert.Equal(t, color.Apply("running", "yellow", "", false, 2), got)
	})
}
