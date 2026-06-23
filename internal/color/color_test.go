package color

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApply(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		fg    string
		bg    string
		bold  bool
		level int
		want  string
	}{
		{
			name:  "cyan foreground",
			text:  "hello",
			fg:    "cyan",
			level: 2,
			want:  "\x1b[36mhello\x1b[0m",
		},
		{
			name:  "bold with color",
			text:  "hello",
			fg:    "red",
			bold:  true,
			level: 2,
			want:  "\x1b[1;31mhello\x1b[22;0m",
		},
		{
			name:  "foreground and background",
			text:  "hello",
			fg:    "white",
			bg:    "blue",
			level: 2,
			want:  "\x1b[37;44mhello\x1b[0;0m",
		},
		{
			name:  "bright color",
			text:  "hello",
			fg:    "brightBlack",
			level: 2,
			want:  "\x1b[90mhello\x1b[0m",
		},
		{
			name:  "level 0 disables colors",
			text:  "hello",
			fg:    "red",
			level: 0,
			want:  "hello",
		},
		{
			name:  "empty text returns empty",
			text:  "",
			fg:    "red",
			level: 2,
			want:  "",
		},
		{
			name:  "no color specified",
			text:  "hello",
			level: 2,
			want:  "hello",
		},
		{
			name:  "unknown color name ignored",
			text:  "hello",
			fg:    "nonexistent",
			level: 2,
			want:  "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Apply(tt.text, tt.fg, tt.bg, tt.bold, tt.level)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no ANSI codes",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "single color code",
			input: "\x1b[36mhello\x1b[0m",
			want:  "hello",
		},
		{
			name:  "multiple color codes",
			input: "\x1b[1;31mbold red\x1b[0m normal \x1b[32mgreen\x1b[0m",
			want:  "bold red normal green",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only ANSI codes",
			input: "\x1b[0m\x1b[36m\x1b[0m",
			want:  "",
		},
		{
			name:  "OSC 8 hyperlink (BEL-terminated)",
			input: "\x1b]8;;https://example.com\x07link\x1b]8;;\x07",
			want:  "link",
		},
		{
			name:  "OSC 8 hyperlink (ST-terminated)",
			input: "\x1b]8;;https://example.com\x1b\\link\x1b]8;;\x1b\\",
			want:  "link",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, StripANSI(tt.input))
		})
	}
}

func TestVisibleWidth(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{name: "plain text", input: "hello", want: 5},
		{name: "colored text", input: "\x1b[36mhello\x1b[0m", want: 5},
		{name: "empty", input: "", want: 0},
		{name: "only ANSI", input: "\x1b[0m", want: 0},
		{name: "combining mark is zero-width", input: "cafe\u0301", want: 4},
		{name: "CJK is double-width", input: "\u4f60\u597d\u4e16\u754c", want: 8},
		{name: "emoji is double-width", input: "\U0001F680", want: 2},
		{name: "OSC 8 hyperlink counts only visible text", input: "\x1b]8;;https://x.com\x07link\x1b]8;;\x07", want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, VisibleWidth(tt.input))
		})
	}
}
