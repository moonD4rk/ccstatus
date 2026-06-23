// Package color provides ANSI color code generation and manipulation.
package color

import (
	"strings"

	fcolor "github.com/fatih/color"
	"github.com/mattn/go-runewidth"
)

// cellWidth measures terminal display width with East Asian "ambiguous" runes
// treated as width 1, so results are stable regardless of the host locale.
var cellWidth = func() *runewidth.Condition {
	c := runewidth.NewCondition()
	c.EastAsianWidth = false
	return c
}()

// fgToBGOffset is the ANSI standard offset from foreground to background codes.
// FG: 30-37, BG: 40-47 (diff=10); FG: 90-97, BG: 100-107 (diff=10).
const fgToBGOffset fcolor.Attribute = 10

// namedColors maps color names to fatih/color foreground attributes.
// Background attributes are derived by adding fgToBGOffset.
var namedColors = map[string]fcolor.Attribute{
	"black":         fcolor.FgBlack,
	"red":           fcolor.FgRed,
	"green":         fcolor.FgGreen,
	"yellow":        fcolor.FgYellow,
	"blue":          fcolor.FgBlue,
	"magenta":       fcolor.FgMagenta,
	"cyan":          fcolor.FgCyan,
	"white":         fcolor.FgWhite,
	"brightBlack":   fcolor.FgHiBlack,
	"brightRed":     fcolor.FgHiRed,
	"brightGreen":   fcolor.FgHiGreen,
	"brightYellow":  fcolor.FgHiYellow,
	"brightBlue":    fcolor.FgHiBlue,
	"brightMagenta": fcolor.FgHiMagenta,
	"brightCyan":    fcolor.FgHiCyan,
	"brightWhite":   fcolor.FgHiWhite,
}

// IsNamed reports whether name is a recognized color name. The empty string is
// not a named color: callers treat empty as "use the widget default".
func IsNamed(name string) bool {
	_, ok := namedColors[name]
	return ok
}

// Apply wraps text with ANSI color codes based on the given color level.
// Returns unmodified text when color level is 0 or text is empty.
func Apply(text, fg, bg string, bold bool, level int) string {
	if level <= 0 || text == "" {
		return text
	}

	c := fcolor.New()
	c.EnableColor()

	added := false
	if bold {
		c.Add(fcolor.Bold)
		added = true
	}
	if a, ok := namedColors[fg]; ok {
		c.Add(a)
		added = true
	}
	if a, ok := namedColors[bg]; ok {
		c.Add(a + fgToBGOffset)
		added = true
	}
	if !added {
		return text
	}
	return c.Sprint(text)
}

// StripANSI removes ANSI escape sequences from a string, including CSI color
// codes and OSC sequences such as OSC 8 hyperlinks.
func StripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			i = ScanEscape(s, i)
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// VisibleWidth returns the terminal display width of s, ignoring ANSI escape
// sequences and counting wide (CJK, emoji) runes as 2 and zero-width (combining)
// runes as 0.
func VisibleWidth(s string) int {
	return cellWidth.StringWidth(StripANSI(s))
}

// RuneWidth returns the terminal display width of a single rune (0, 1, or 2).
func RuneWidth(r rune) int {
	return cellWidth.RuneWidth(r)
}

// ScanEscape returns the index just past the ANSI escape sequence beginning at
// s[i] (which must be ESC, 0x1b): CSI (ESC [ ... final letter) or OSC
// (ESC ] ... terminated by BEL or ST). An unrecognized or truncated sequence
// consumes the lone ESC and returns i+1.
func ScanEscape(s string, i int) int {
	if i+1 >= len(s) || s[i] != '\x1b' {
		return i + 1
	}
	switch s[i+1] {
	case '[': // CSI
		j := i + 2
		for j < len(s) && !isTerminator(s[j]) {
			j++
		}
		if j < len(s) {
			j++ // include the final byte
		}
		return j
	case ']': // OSC, terminated by BEL (0x07) or ST (ESC \)
		j := i + 2
		for j < len(s) {
			if s[j] == '\x07' {
				return j + 1
			}
			if s[j] == '\x1b' && j+1 < len(s) && s[j+1] == '\\' {
				return j + 2
			}
			j++
		}
		return j
	default:
		return i + 1
	}
}

func isTerminator(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}
