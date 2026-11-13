package render

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
	"github.com/moond4rk/ccstatus/internal/widget"
)

func text(s string) config.WidgetItem {
	return config.WidgetItem{Type: "custom-text", CustomText: s}
}

func TestFlexSeparatorLayout(t *testing.T) {
	settings := config.Settings{ColorLevel: 0, DefaultPadding: " ", DefaultSeparator: "|"}
	flex := config.WidgetItem{Type: "flex-separator"}
	sep := config.WidgetItem{Type: "separator"}
	gone := config.WidgetItem{Type: "version"} // renders empty: the payload has no version

	tests := []struct {
		name  string
		items []config.WidgetItem
		width int
		want  string
	}{
		{"fills the width", []config.WidgetItem{text("AAAA"), flex, text("BBBB")}, 12, "AAAA    BBBB"},
		{"keeps one column when the row just fits", []config.WidgetItem{text("AAAA"), flex, text("BBBB")}, 9, "AAAA BBBB"},
		{"truncates rather than joining the groups", []config.WidgetItem{text("AAAA"), flex, text("BBBB")}, 8, "AAAA ..."},
		{
			"every gap keeps a column when the free width is short",
			[]config.WidgetItem{text("A"), flex, text("B"), flex, text("C"), flex, text("D")},
			7, "A B C D",
		},
		{
			"the remainder goes to the leading gaps",
			[]config.WidgetItem{text("A"), flex, text("B"), flex, text("C")},
			8, "A   B  C",
		},
		{"a trailing flex separator does not force truncation", []config.WidgetItem{text("AAAA"), flex}, 4, "AAAA"},
		{"an empty middle group keeps the outer groups apart", []config.WidgetItem{text("AAAA"), flex, gone, flex, text("BBBB")}, 9, "AAAA BBBB"},
		{
			"no separator is left dangling beside the flex",
			[]config.WidgetItem{text("L"), sep, gone, flex, gone, sep, text("R")},
			10, "L        R",
		},
		{"unknown width joins the groups with single spaces", []config.WidgetItem{text("A"), flex, gone, flex, text("B")}, 0, "A B"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderRow(tt.items, &settings, widget.RenderContext{Data: &status.Session{}}, tt.width)
			assert.Equal(t, tt.want, color.StripANSI(got))
			if tt.width > 0 {
				assert.LessOrEqual(t, color.VisibleWidth(got), tt.width)
			}
		})
	}
}

func TestRowWidthIsCapped(t *testing.T) {
	settings := config.Settings{ColorLevel: 0, DefaultPadding: " "}
	items := []config.WidgetItem{text("L"), {Type: "flex-separator"}, text("R")}
	got := RenderRow(items, &settings, widget.RenderContext{Data: &status.Session{}}, 1<<40)
	assert.Equal(t, maxRowWidth, color.VisibleWidth(got))
}

func TestInheritedSeparatorColorFollowsThresholds(t *testing.T) {
	settings := config.Settings{ColorLevel: 2, DefaultPadding: " ", DefaultSeparator: "|", InheritSeparatorColors: true}
	items := []config.WidgetItem{
		{Type: "context-percentage", Color: "green", Thresholds: []config.Threshold{{GTE: 90, Color: "red"}}},
		{Type: "separator"},
		{Type: "custom-text", CustomText: "x", Color: "blue"},
	}
	tests := []struct {
		pct  float64
		want string
	}{
		{12, "green"},
		{95, "red"},
	}
	for _, tt := range tests {
		got := RenderLine(items, &settings, contextAt(tt.pct))
		assert.Contains(t, got, color.Apply("|", tt.want, "", false, 2), "%v%%", tt.pct)
	}
}

func TestTruncateCountsGraphemeClusters(t *testing.T) {
	warn := "\u26a0\ufe0f" // one cluster, two columns, though its first rune alone is one
	tests := []struct {
		name string
		line string
	}{
		{"emoji with variation selectors", strings.Repeat(warn, 20)},
		{"combining marks", strings.Repeat("e\u0301", 30)},
		{"zero-width-joiner sequences", strings.Repeat("\U0001F468\u200d\U0001F469\u200d\U0001F467", 10)},
		{"regional-indicator flags", strings.Repeat("\U0001F1E9\U0001F1EA", 10)},
		{"colored emoji", "\x1b[33m" + strings.Repeat(warn, 20) + "\x1b[0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for width := 1; width <= 12; width++ {
				assert.LessOrEqualf(t, color.VisibleWidth(Truncate(tt.line, width)), width, "Truncate at %d", width)
				assert.LessOrEqualf(t, color.VisibleWidth(truncateWidget(tt.line, width)), width, "truncateWidget at %d", width)
			}
		})
	}

	t.Run("a cluster is kept whole", func(t *testing.T) {
		assert.Equal(t, warn+warn+"…", truncateWidget(strings.Repeat(warn, 5), 5))
	})
}
