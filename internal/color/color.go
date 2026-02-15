// Package color provides ANSI color code generation and manipulation.
package color

import (
	"strings"
	"unicode/utf8"

	fcolor "github.com/fatih/color"
)

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

func isTerminator(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}
