// Package color provides ANSI color code generation and manipulation.
package color

import (
	"fmt"
	"iter"
	"slices"
	"strconv"
	"strings"

	"github.com/clipperhouse/uax29/v2/graphemes"
	"github.com/mattn/go-runewidth"
)

// cellWidth measures terminal display width with East Asian "ambiguous" runes
// treated as width 1, so results are stable regardless of the host locale.
var cellWidth = func() *runewidth.Condition {
	c := runewidth.NewCondition()
	c.EastAsianWidth = false
	return c
}()

// The color levels: how many colors Apply may ask the terminal for.
const (
	LevelNone      = 0 // no escape codes at all
	Level16        = 1 // the 16 ANSI colors
	Level256       = 2 // the 256-color palette
	LevelTrueColor = 3 // 24-bit color
)

// names lists the 16 ANSI colors in palette order: a color's index is its
// palette index.
var names = []string{
	"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white",
	"brightBlack", "brightRed", "brightGreen", "brightYellow", "brightBlue", "brightMagenta", "brightCyan", "brightWhite",
}

// Names returns the 16 color names, basic colors first.
func Names() []string {
	return slices.Clone(names)
}

// kind is the way a color was written in the settings.
type kind int

const (
	kindNamed   kind = iota // one of names
	kindPalette             // an index into the 256-color palette, "0" to "255"
	kindRGB                 // "#rgb" or "#rrggbb"
)

// value is a parsed color.
type value struct {
	kind  kind
	index int // palette index, for named and palette colors
	rgb   rgb // for RGB colors
}

type rgb struct{ r, g, b int }

// parse reads a color as the settings write it: a name, a palette index or a
// hex triplet.
func parse(s string) (value, bool) {
	if i := slices.Index(names, s); i >= 0 {
		return value{kind: kindNamed, index: i}, true
	}
	if c, ok := parseHex(s); ok {
		return value{kind: kindRGB, rgb: c}, true
	}
	if i, ok := parsePaletteIndex(s); ok {
		return value{kind: kindPalette, index: i}, true
	}
	return value{}, false
}

// parseHex reads "#rgb" or "#rrggbb", in either case.
func parseHex(s string) (rgb, bool) {
	hex, ok := strings.CutPrefix(s, "#")
	if !ok {
		return rgb{}, false
	}
	const short, long = 3, 6
	if len(hex) == short {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != long {
		return rgb{}, false
	}
	n, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return rgb{}, false
	}
	return rgb{r: int(n >> 16), g: int(n >> 8 & 0xff), b: int(n & 0xff)}, true
}

// PalettePattern and HexPattern are the regular expressions of the two color
// forms that are not names, for the settings JSON Schema.
const (
	PalettePattern = `^(\d|[1-9]\d|1\d{2}|2[0-4]\d|25[0-5])$`
	HexPattern     = `^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`
)

// parsePaletteIndex reads a decimal palette index, "0" to "255", written
// without leading zeros.
func parsePaletteIndex(s string) (int, bool) {
	const maxDigits, maxIndex = 3, 255
	if s == "" || len(s) > maxDigits || strings.TrimLeft(s, "0123456789") != "" || (len(s) > 1 && s[0] == '0') {
		return 0, false
	}
	i, err := strconv.Atoi(s)
	return i, err == nil && i <= maxIndex
}

// IsValid reports whether s names a color: one of Names, a palette index "0" to
// "255", or a hex triplet "#rgb" / "#rrggbb". The empty string is not a color:
// callers treat empty as "use the widget default".
func IsValid(s string) bool {
	_, ok := parse(s)
	return ok
}

// Apply wraps text with ANSI color codes for the given color level. Colors the
// level cannot express are mapped to the nearest one it can; unknown color
// names are ignored. It returns text unmodified when the level is LevelNone,
// text is empty, or nothing is left to apply.
func Apply(text, fg, bg string, bold bool, level int) string {
	if level <= LevelNone || text == "" {
		return text
	}
	var codes []string
	if bold {
		codes = append(codes, "1")
	}
	if c, ok := parse(fg); ok {
		codes = append(codes, c.sgr(level, false))
	}
	if c, ok := parse(bg); ok {
		codes = append(codes, c.sgr(level, true))
	}
	if len(codes) == 0 {
		return text
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + text + "\x1b[0m"
}

// SGR parameters: foreground bases, the offset from a foreground code to its
// background code, and the selectors for extended colors.
const (
	sgrBasic      = 30
	sgrBright     = 90
	sgrBackground = 10
	sgrExtended   = 38
	sgrPalette    = 5
	sgrRGB        = 2

	ansiColors = 16 // the palette entries Level16 can address
	basicCount = 8  // of which the first eight are the basic colors
)

// sgr returns the SGR parameters that select the color at level, as a
// background color when background is set.
func (v value) sgr(level int, background bool) string {
	offset := 0
	if background {
		offset = sgrBackground
	}
	if v.kind == kindRGB && level >= LevelTrueColor {
		return fmt.Sprintf("%d;%d;%d;%d;%d", sgrExtended+offset, sgrRGB, v.rgb.r, v.rgb.g, v.rgb.b)
	}

	index := v.index
	switch {
	case v.kind == kindRGB && level >= Level256:
		index = nearestPalette(v.rgb)
	case v.kind == kindRGB:
		index = nearestANSI(v.rgb)
	case index >= ansiColors && level < Level256:
		index = nearestANSI(paletteRGB(index))
	}

	switch {
	case index < basicCount:
		return strconv.Itoa(sgrBasic + offset + index)
	case index < ansiColors:
		return strconv.Itoa(sgrBright + offset + index - basicCount)
	default:
		return fmt.Sprintf("%d;%d;%d", sgrExtended+offset, sgrPalette, index)
	}
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
