// Package color provides ANSI color code generation and manipulation.
package color

import (
	"iter"
	"strings"

	"github.com/clipperhouse/uax29/v2/graphemes"
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

// variationSelector16 asks for the emoji presentation of the symbol before it.
const variationSelector16 = "\ufe0f"

// VisibleWidth returns the terminal display width of s, ignoring ANSI escape
// sequences. It is the sum of the widths of the grapheme clusters of s: wide
// (CJK, emoji) clusters count as 2 and zero-width (combining) ones as 0.
func VisibleWidth(s string) int {
	width := 0
	for _, w := range Clusters(StripANSI(s)) {
		width += w
	}
	return width
}

// Clusters iterates over the grapheme clusters of s, which must hold no escape
// sequences, with the display width of each. A cluster is what a terminal draws
// as one glyph: an emoji with its modifiers, a letter with its combining marks.
// Together the clusters cover every byte of s.
func Clusters(s string) iter.Seq2[string, int] {
	return func(yield func(string, int) bool) {
		g := graphemes.FromString(s)
		end := 0
		for g.Next() {
			end = g.End()
			if !yield(g.Value(), clusterWidth(g.Value())) {
				return
			}
		}
		// The segmenter stops at a multibyte sequence cut short by the end of s.
		for ; end < len(s); end++ {
			if !yield(s[end:end+1], clusterWidth(s[end:end+1])) {
				return
			}
		}
	}
}

// clusterWidth returns the display width of one grapheme cluster. go-runewidth
// keeps a text-style symbol at one column when a variation selector turns it
// into an emoji, such as U+26A0 U+FE0F; terminals draw that emoji two wide.
func clusterWidth(cluster string) int {
	w := cellWidth.StringWidth(cluster)
	if w == 1 && strings.Contains(cluster, variationSelector16) {
		return 2
	}
	return w
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
