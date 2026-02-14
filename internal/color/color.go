// Package color provides ANSI color code generation and manipulation.
package color

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	fcolor "github.com/fatih/color"
)

const hexColorLen = 6

// colorCode holds ANSI codes for a named color.
type colorCode struct {
	fg int
	bg int
}

//nolint:mnd // ANSI color codes are standard values
var namedColors = map[string]colorCode{
	"black":         {30, 40},
	"red":           {31, 41},
	"green":         {32, 42},
	"yellow":        {33, 43},
	"blue":          {34, 44},
	"magenta":       {35, 45},
	"cyan":          {36, 46},
	"white":         {37, 47},
	"brightBlack":   {90, 100},
	"brightRed":     {91, 101},
	"brightGreen":   {92, 102},
	"brightYellow":  {93, 103},
	"brightBlue":    {94, 104},
	"brightMagenta": {95, 105},
	"brightCyan":    {96, 106},
	"brightWhite":   {97, 107},
}

// Apply wraps text with ANSI color codes based on the given color level.
// Returns unmodified text when color level is 0 or colors are disabled.
func Apply(text, fg, bg string, bold bool, level int) string {
	if level == 0 || text == "" {
		return text
	}

	var codes []string
	if bold {
		codes = append(codes, "1")
	}
	if fg != "" {
		if c := resolveFG(fg, level); c != "" {
			codes = append(codes, c)
		}
	}
	if bg != "" {
		if c := resolveBG(bg, level); c != "" {
			codes = append(codes, c)
		}
	}
	if len(codes) == 0 {
		return text
	}
	return fmt.Sprintf("\x1b[%sm%s\x1b[0m", strings.Join(codes, ";"), text)
}

// IsDisabled returns true if color output should be suppressed (NO_COLOR env var).
func IsDisabled() bool {
	return fcolor.NoColor
}

// StripANSI removes all ANSI escape sequences from a string.
func StripANSI(s string) string {
	var result strings.Builder
	result.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && !isTerminator(s[j]) {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j
			continue
		}
		result.WriteByte(s[i])
		i++
	}
	return result.String()
}

// VisibleWidth returns the number of visible characters (runes) in a string,
// ignoring ANSI escape sequences.
func VisibleWidth(s string) int {
	return utf8.RuneCountInString(StripANSI(s))
}

func resolveFG(name string, level int) string {
	if c, ok := namedColors[name]; ok {
		return strconv.Itoa(c.fg)
	}
	if strings.HasPrefix(name, "ansi256:") {
		if level < 2 {
			return ""
		}
		return "38;5;" + strings.TrimPrefix(name, "ansi256:")
	}
	if strings.HasPrefix(name, "hex:") {
		if level < 3 {
			return ""
		}
		hex := strings.TrimPrefix(name, "hex:")
		r, g, b := parseHex(hex)
		return fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
	}
	return ""
}

func resolveBG(name string, level int) string {
	if c, ok := namedColors[name]; ok {
		return strconv.Itoa(c.bg)
	}
	if strings.HasPrefix(name, "ansi256:") {
		if level < 2 {
			return ""
		}
		return "48;5;" + strings.TrimPrefix(name, "ansi256:")
	}
	if strings.HasPrefix(name, "hex:") {
		if level < 3 {
			return ""
		}
		hex := strings.TrimPrefix(name, "hex:")
		r, g, b := parseHex(hex)
		return fmt.Sprintf("48;2;%d;%d;%d", r, g, b)
	}
	return ""
}

func parseHex(hex string) (r, g, b int) {
	if len(hex) != hexColorLen {
		return 0, 0, 0
	}
	rv, _ := strconv.ParseInt(hex[0:2], 16, 32)
	gv, _ := strconv.ParseInt(hex[2:4], 16, 32)
	bv, _ := strconv.ParseInt(hex[4:6], 16, 32)
	return int(rv), int(gv), int(bv)
}

func isTerminator(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}
