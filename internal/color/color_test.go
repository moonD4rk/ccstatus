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
			want:  "\x1b[1;31mhello\x1b[0m",
		},
		{
			name:  "foreground and background",
			text:  "hello",
			fg:    "white",
			bg:    "blue",
			level: 2,
			want:  "\x1b[37;44mhello\x1b[0m",
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
			name:  "ansi256 at level 2",
			text:  "hello",
			fg:    "ansi256:208",
			level: 2,
			want:  "\x1b[38;5;208mhello\x1b[0m",
		},
		{
			name:  "ansi256 at level 1 ignored",
			text:  "hello",
			fg:    "ansi256:208",
			level: 1,
			want:  "hello",
		},
		{
			name:  "hex at level 3",
			text:  "hello",
			fg:    "hex:FF8000",
			level: 3,
			want:  "\x1b[38;2;255;128;0mhello\x1b[0m",
		},
		{
			name:  "hex at level 2 ignored",
			text:  "hello",
			fg:    "hex:FF8000",
			level: 2,
			want:  "hello",
		},
		{
			name:  "ansi256 background",
			text:  "hello",
			bg:    "ansi256:16",
			level: 2,
			want:  "\x1b[48;5;16mhello\x1b[0m",
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
		{name: "unicode", input: "cafe\u0301", want: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, VisibleWidth(tt.input))
		})
	}
}

func TestParseHex(t *testing.T) {
	tests := []struct {
		hex     string
		r, g, b int
	}{
		{"FF0000", 255, 0, 0},
		{"00FF00", 0, 255, 0},
		{"0000FF", 0, 0, 255},
		{"FF8000", 255, 128, 0},
		{"bad", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.hex, func(t *testing.T) {
			r, g, b := parseHex(tt.hex)
			assert.Equal(t, tt.r, r)
			assert.Equal(t, tt.g, g)
			assert.Equal(t, tt.b, b)
		})
	}
}
