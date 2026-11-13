package color

import (
	"regexp"
	"slices"
	"strconv"
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

func TestApplyColorLevels(t *testing.T) {
	tests := []struct {
		name   string
		fg, bg string
		want   [3]string // the SGR parameters at levels 1, 2 and 3
	}{
		{"a named color is the same at every level", "cyan", "", [3]string{"36", "36", "36"}},
		{"a bright background", "", "brightRed", [3]string{"101", "101", "101"}},
		{"a palette index below 16 is its ANSI color", "9", "4", [3]string{"91;44", "91;44", "91;44"}},
		{"a palette color falls back to the nearest ANSI color", "208", "", [3]string{"33", "38;5;208", "38;5;208"}},
		{"a palette background", "", "236", [3]string{"40", "48;5;236", "48;5;236"}},
		{"a hex color is exact only in true color", "#ff8700", "", [3]string{"33", "38;5;208", "38;2;255;135;0"}},
		{"a short hex color", "#f00", "", [3]string{"91", "38;5;196", "38;2;255;0;0"}},
		{"hex is case-insensitive", "#FF0000", "", [3]string{"91", "38;5;196", "38;2;255;0;0"}},
		{"a hex gray maps onto the gray ramp", "#808080", "", [3]string{"90", "38;5;244", "38;2;128;128;128"}},
		{"a hex background", "#1e1e2e", "#cdd6f4", [3]string{"30;47", "38;5;235;48;5;189", "38;2;30;30;46;48;2;205;214;244"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, want := range tt.want {
				level := i + 1
				assert.Equalf(t, "\x1b["+want+"mx\x1b[0m", Apply("x", tt.fg, tt.bg, false, level), "level %d", level)
			}
			assert.Equal(t, "x", Apply("x", tt.fg, tt.bg, false, LevelNone))
		})
	}

	t.Run("a level above true color is true color", func(t *testing.T) {
		assert.Equal(t, Apply("x", "#123456", "", false, LevelTrueColor), Apply("x", "#123456", "", false, 9))
	})
}

func TestIsValid(t *testing.T) {
	for _, s := range []string{"red", "brightWhite", "0", "7", "42", "255", "#fff", "#FFF", "#a1b2c3"} {
		assert.Truef(t, IsValid(s), "%q", s)
	}
	for _, s := range []string{
		"", "Red", "bright-white", "chartreuse", "256", "1000", "-1", "+1", "1.5", "0x10", "007", "00",
		"#", "#ff", "#ffff", "#fffff", "#fffffff", "#ggg", "#12345g", "ffffff", "rgb(1,2,3)",
	} {
		assert.Falsef(t, IsValid(s), "%q", s)
	}
}

func TestPatternsAgreeWithIsValid(t *testing.T) {
	palette := regexp.MustCompile(PalettePattern)
	hex := regexp.MustCompile(HexPattern)
	for i := -1; i <= 300; i++ {
		s := strconv.Itoa(i)
		assert.Equalf(t, IsValid(s), palette.MatchString(s), "%q", s)
	}
	for _, s := range []string{"007", "00", "", "#fff", "#FFFFFF", "#ffff", "#12345g", "fff", "#", "red"} {
		assert.Equalf(t, IsValid(s) && !slices.Contains(Names(), s), palette.MatchString(s) || hex.MatchString(s), "%q", s)
	}
}

func TestNames(t *testing.T) {
	got := Names()
	assert.Len(t, got, 16)
	assert.Equal(t, "black", got[0])
	assert.Equal(t, "brightWhite", got[15])
	for _, name := range got {
		assert.True(t, IsValid(name), name)
	}
	got[0] = "changed"
	assert.Equal(t, "black", Names()[0], "the caller gets a copy")
}

func TestPalette(t *testing.T) {
	t.Run("entries map to their colors", func(t *testing.T) {
		assert.Equal(t, rgb{0, 0, 0}, paletteRGB(16))
		assert.Equal(t, rgb{255, 255, 255}, paletteRGB(231))
		assert.Equal(t, rgb{255, 135, 0}, paletteRGB(208))
		assert.Equal(t, rgb{8, 8, 8}, paletteRGB(232))
		assert.Equal(t, rgb{238, 238, 238}, paletteRGB(255))
		assert.Equal(t, rgb{205, 0, 0}, paletteRGB(1))
	})

	t.Run("every cube and gray entry is its own nearest color", func(t *testing.T) {
		for index := cubeStart; index <= 255; index++ {
			got := nearestPalette(paletteRGB(index))
			assert.Equalf(t, paletteRGB(index), paletteRGB(got), "entry %d mapped to %d", index, got)
		}
	})

	t.Run("every ANSI color is its own nearest color", func(t *testing.T) {
		for index := range ansiColors {
			assert.Equal(t, index, nearestANSI(ansiRGB[index]), index)
		}
	})

	t.Run("out-of-gamut extremes stay in range", func(t *testing.T) {
		for _, c := range []rgb{{0, 0, 0}, {255, 255, 255}, {1, 1, 1}, {254, 254, 254}, {255, 0, 255}} {
			got := nearestPalette(c)
			assert.GreaterOrEqual(t, got, cubeStart)
			assert.LessOrEqual(t, got, 255)
		}
	})
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
