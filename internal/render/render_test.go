package render

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
	"github.com/moond4rk/ccstatus/internal/widget"
)

func intPtr(v int) *int { return &v }

func TestRenderLine(t *testing.T) {
	tests := []struct {
		name          string
		items         []config.WidgetItem
		settings      config.Settings
		data          *status.StatusJSON
		terminalWidth int
		check         func(t *testing.T, result string)
	}{
		{
			name: "single model widget",
			items: []config.WidgetItem{
				{ID: "1", Type: "model", Color: "cyan"},
			},
			settings: config.Settings{ColorLevel: 2, DefaultPadding: " "},
			data: &status.StatusJSON{
				Model: status.ModelField{ID: "claude-sonnet-4-5", DisplayName: "Sonnet"},
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				stripped := color.StripANSI(result)
				assert.Equal(t, "Sonnet", stripped)
			},
		},
		{
			name: "model and separator and version",
			items: []config.WidgetItem{
				{ID: "1", Type: "model", Color: "cyan"},
				{ID: "2", Type: "separator"},
				{ID: "3", Type: "version", Color: "brightBlack"},
			},
			settings: config.Settings{ColorLevel: 2, DefaultSeparator: "|", DefaultPadding: " "},
			data: &status.StatusJSON{
				Model:   status.ModelField{ID: "claude-sonnet-4-5", DisplayName: "Sonnet"},
				Version: "1.0.80",
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				stripped := color.StripANSI(result)
				assert.Equal(t, "Sonnet | 1.0.80", stripped)
			},
		},
		{
			name: "empty widgets are skipped with separators cleaned",
			items: []config.WidgetItem{
				{ID: "1", Type: "model", Color: "cyan"},
				{ID: "2", Type: "separator"},
				{ID: "3", Type: "version"},
				{ID: "4", Type: "separator"},
				{ID: "5", Type: "custom-text", CustomText: "hello"},
			},
			settings: config.Settings{ColorLevel: 2, DefaultSeparator: "|", DefaultPadding: " "},
			data: &status.StatusJSON{
				Model: status.ModelField{DisplayName: "Sonnet"},
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				stripped := color.StripANSI(result)
				assert.Equal(t, "Sonnet | hello", stripped)
			},
		},
		{
			name: "all empty produces empty string",
			items: []config.WidgetItem{
				{ID: "1", Type: "version"},
				{ID: "2", Type: "separator"},
				{ID: "3", Type: "version"},
			},
			settings: config.Settings{ColorLevel: 2, DefaultSeparator: "|", DefaultPadding: " "},
			data:     &status.StatusJSON{},
			check: func(t *testing.T, result string) {
				t.Helper()
				assert.Empty(t, result)
			},
		},
		{
			name: "color level 0 produces no ANSI codes",
			items: []config.WidgetItem{
				{ID: "1", Type: "model", Color: "cyan"},
			},
			settings: config.Settings{ColorLevel: 0, DefaultPadding: " "},
			data: &status.StatusJSON{
				Model: status.ModelField{DisplayName: "Sonnet"},
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				assert.Equal(t, "Sonnet", result)
			},
		},
		{
			name: "unknown widget type is skipped",
			items: []config.WidgetItem{
				{ID: "1", Type: "model", Color: "cyan"},
				{ID: "2", Type: "separator"},
				{ID: "3", Type: "nonexistent-widget"},
			},
			settings: config.Settings{ColorLevel: 2, DefaultSeparator: "|", DefaultPadding: " "},
			data: &status.StatusJSON{
				Model: status.ModelField{DisplayName: "Sonnet"},
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				stripped := color.StripANSI(result)
				assert.Equal(t, "Sonnet", stripped)
			},
		},
		{
			name: "truncation when terminal width is small",
			items: []config.WidgetItem{
				{ID: "1", Type: "custom-text", CustomText: "This is a very long status line text"},
			},
			settings:      config.Settings{ColorLevel: 0, DefaultPadding: " "},
			data:          &status.StatusJSON{},
			terminalWidth: 10,
			check: func(t *testing.T, result string) {
				t.Helper()
				assert.LessOrEqual(t, color.VisibleWidth(result), 10)
				assert.Contains(t, result, "...")
			},
		},
		{
			name: "token widgets render formatted values",
			items: []config.WidgetItem{
				{ID: "1", Type: "tokens-input"},
				{ID: "2", Type: "separator"},
				{ID: "3", Type: "context-percentage"},
			},
			settings: config.Settings{ColorLevel: 0, DefaultSeparator: "|", DefaultPadding: " "},
			data: &status.StatusJSON{
				ContextWindow: &status.ContextWindow{
					TotalInputTokens: intPtr(50_000),
					UsedPercentage:   func() *float64 { v := 25.0; return &v }(),
				},
			},
			check: func(t *testing.T, result string) {
				t.Helper()
				assert.Equal(t, "50.0k | 25%", result)
			},
		},
		{
			name: "flex separator expands to fill width",
			items: []config.WidgetItem{
				{ID: "1", Type: "custom-text", CustomText: "L"},
				{ID: "2", Type: "flex-separator"},
				{ID: "3", Type: "custom-text", CustomText: "R"},
			},
			settings:      config.Settings{ColorLevel: 0, DefaultPadding: " "},
			data:          &status.StatusJSON{},
			terminalWidth: 20,
			check: func(t *testing.T, result string) {
				t.Helper()
				assert.Equal(t, 20, color.VisibleWidth(result))
				assert.Equal(t, byte('L'), result[0])
				assert.Equal(t, byte('R'), result[len(result)-1])
			},
		},
		{
			name: "flex separator without terminal width uses single space",
			items: []config.WidgetItem{
				{ID: "1", Type: "custom-text", CustomText: "L"},
				{ID: "2", Type: "flex-separator"},
				{ID: "3", Type: "custom-text", CustomText: "R"},
			},
			settings: config.Settings{ColorLevel: 0, DefaultPadding: " "},
			data:     &status.StatusJSON{},
			check: func(t *testing.T, result string) {
				t.Helper()
				assert.Equal(t, "L R", result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := widget.RenderContext{Data: tt.data, TerminalWidth: tt.terminalWidth}
			result := RenderLine(tt.items, &tt.settings, ctx)
			tt.check(t, result)
		})
	}
}

func TestPostProcess(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "replaces spaces with NBSP and prepends reset",
			input: "hello world",
			want:  "\x1b[0mhello\u00A0world",
		},
		{
			name:  "empty visible text returns empty",
			input: "\x1b[0m\x1b[36m\x1b[0m",
			want:  "",
		},
		{
			name:  "whitespace only returns empty",
			input: "   ",
			want:  "",
		},
		{
			name:  "colored text gets NBSP and reset",
			input: "\x1b[36mSonnet\x1b[0m | \x1b[35mmain\x1b[0m",
			want:  "\x1b[0m\x1b[36mSonnet\x1b[0m\u00A0|\u00A0\x1b[35mmain\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PostProcess(tt.input))
		})
	}
}

func TestCleanSeparators(t *testing.T) {
	sep := segment{text: "|", item: &config.WidgetItem{Type: "separator"}, isSep: true}
	w1 := segment{text: "A", item: &config.WidgetItem{Type: "model"}, isSep: false}
	w2 := segment{text: "B", item: &config.WidgetItem{Type: "version"}, isSep: false}
	empty := segment{text: "", item: &config.WidgetItem{Type: "version"}, isSep: false}
	flex := segment{text: "flex-separator", item: &config.WidgetItem{Type: "flex-separator"}, isSep: false}

	tests := []struct {
		name     string
		input    []segment
		wantText []string
	}{
		{
			name:     "normal: widget sep widget",
			input:    []segment{w1, sep, w2},
			wantText: []string{"A", "|", "B"},
		},
		{
			name:     "removes trailing separator",
			input:    []segment{w1, sep},
			wantText: []string{"A"},
		},
		{
			name:     "removes leading separator",
			input:    []segment{sep, w1},
			wantText: []string{"A"},
		},
		{
			name:     "removes empty non-separators and cleans seps",
			input:    []segment{w1, sep, empty, sep, w2},
			wantText: []string{"A", "|", "B"},
		},
		{
			name:     "all empty produces empty",
			input:    []segment{empty, sep, empty},
			wantText: nil,
		},
		{
			name:     "consecutive separators reduced",
			input:    []segment{w1, sep, sep, sep, w2},
			wantText: []string{"A", "|", "B"},
		},
		{
			name:     "flex separator preserved",
			input:    []segment{w1, flex, w2},
			wantText: []string{"A", "flex-separator", "B"},
		},
		{
			name:     "empty widgets around flex are cleaned",
			input:    []segment{w1, flex, empty},
			wantText: []string{"A", "flex-separator"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanSeparators(tt.input)
			var texts []string
			for _, s := range result {
				texts = append(texts, s.text)
			}
			assert.Equal(t, tt.wantText, texts)
		})
	}
}

func TestCalculateFlexWidth(t *testing.T) {
	tests := []struct {
		name             string
		detected         int
		flexMode         string
		compactThreshold int
		contextPct       float64
		want             int
	}{
		{"full mode", 100, "full", 60, 0, 94},
		{"full-minus-40 mode", 100, "full-minus-40", 60, 0, 60},
		{"full-until-compact below threshold", 100, "full-until-compact", 60, 30, 94},
		{"full-until-compact at threshold", 100, "full-until-compact", 60, 60, 60},
		{"full-until-compact above threshold", 100, "full-until-compact", 60, 80, 60},
		{"unknown mode returns detected", 100, "unknown", 60, 0, 100},
		{"empty mode returns detected", 100, "", 60, 0, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, CalculateFlexWidth(tt.detected, tt.flexMode, tt.compactThreshold, tt.contextPct))
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		maxWidth int
		want     string
	}{
		{
			name:     "no truncation needed",
			line:     "hello",
			maxWidth: 10,
			want:     "hello",
		},
		{
			name:     "exact fit",
			line:     "hello",
			maxWidth: 5,
			want:     "hello",
		},
		{
			name:     "truncates with suffix",
			line:     "hello world",
			maxWidth: 8,
			want:     "hello...",
		},
		{
			name:     "very small width",
			line:     "hello world",
			maxWidth: 2,
			want:     "..",
		},
		{
			name:     "zero width",
			line:     "hello",
			maxWidth: 0,
			want:     "",
		},
		{
			name:     "preserves ANSI codes in truncated output",
			line:     "\x1b[36mhello world\x1b[0m",
			maxWidth: 8,
			want:     "\x1b[36mhello\x1b[0m...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Truncate(tt.line, tt.maxWidth)
			visible := color.VisibleWidth(result)
			assert.LessOrEqual(t, visible, tt.maxWidth)
			if tt.maxWidth > 0 {
				stripped := color.StripANSI(result)
				expectedStripped := color.StripANSI(tt.want)
				assert.Equal(t, expectedStripped, stripped)
			}
		})
	}
}
