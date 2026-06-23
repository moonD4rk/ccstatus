package render

import (
	"strings"
	"unicode/utf8"

	"github.com/moond4rk/ccstatus/internal/color"
)

const truncSuffix = "..."

// Truncate shortens a line to fit within maxWidth display columns. ANSI escape
// sequences are preserved in the output but do not count toward width; wide
// runes count as 2 cells. A "..." suffix is appended when truncation occurs.
func Truncate(line string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	if color.VisibleWidth(line) <= maxWidth {
		return line
	}

	suffixWidth := color.VisibleWidth(truncSuffix)
	target := maxWidth - suffixWidth
	if target <= 0 {
		return truncSuffix[:maxWidth]
	}

	var b strings.Builder
	b.Grow(len(line))
	width := 0
	i := 0
	for i < len(line) {
		// Copy escape sequences through without counting their width.
		if line[i] == '\x1b' {
			j := color.ScanEscape(line, i)
			b.WriteString(line[i:j])
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		rw := color.RuneWidth(r)
		if width+rw > target {
			break
		}
		b.WriteString(line[i : i+size])
		width += rw
		i += size
	}

	// Reset colors before the suffix.
	b.WriteString("\x1b[0m")
	b.WriteString(truncSuffix)
	return b.String()
}
