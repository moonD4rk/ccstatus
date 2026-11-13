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
		{name: "text-style symbol is one column", input: "\u26a0", want: 1},
		{name: "emoji presentation selector widens it to two", input: "\u26a0\ufe0f", want: 2},
		{name: "a selector split off by a color code still widens", input: "\u26a0\x1b[0m\ufe0f", want: 2},
		{name: "zero-width-joiner sequence is one glyph", input: "\U0001F468\u200d\U0001F469\u200d\U0001F467", want: 2},
		{name: "flag is one glyph", input: "\U0001F1E9\U0001F1EA", want: 2},
		{name: "control characters are zero-width", input: "a\r\nb\t", want: 2},
		{name: "invalid bytes count one column each", input: "a\xff\xfe", want: 3},
		{name: "a truncated multibyte tail is still counted", input: "ab\xe4\xb8", want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, VisibleWidth(tt.input))
		})
	}
}

func TestClusters(t *testing.T) {
	type cluster struct {
		text  string
		width int
	}
	tests := []struct {
		name  string
		input string
		want  []cluster
	}{
		{"empty", "", nil},
		{"ascii", "ab", []cluster{{"a", 1}, {"b", 1}}},
		{"combining mark stays with its letter", "e\u0301x", []cluster{{"e\u0301", 1}, {"x", 1}}},
		{"emoji presentation sequence", "\u26a0\ufe0f!", []cluster{{"\u26a0\ufe0f", 2}, {"!", 1}}},
		{"wide rune", "\u4e2d", []cluster{{"\u4e2d", 2}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []cluster
			for text, width := range Clusters(tt.input) {
				got = append(got, cluster{text, width})
			}
			assert.Equal(t, tt.want, got)
		})
	}

	t.Run("covers every byte, valid or not", func(t *testing.T) {
		for _, input := range []string{"ab\xe4\xb8", "\xf0\x9f", "a\xffb", "\u4e2d\xe4", "x\xc3"} {
			var joined string
			for text := range Clusters(input) {
				joined += text
			}
			assert.Equal(t, input, joined)
		}
	})

	t.Run("stops when the caller breaks", func(t *testing.T) {
		n := 0
		for range Clusters("abc") {
			n++
			break
		}
		assert.Equal(t, 1, n)
	})
}
