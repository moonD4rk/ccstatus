package render

import (
	"strings"
	"unicode/utf8"

	"github.com/moond4rk/ccstatus/internal/color"
)

const (
	truncSuffix    = "..."
	hyperlinkClose = "\x1b]8;;\x07"
)

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
	linkOpen := false
	i := 0
	for i < len(line) {
		// Copy escape sequences through without counting their width.
		if line[i] == '\x1b' {
			j := color.ScanEscape(line, i)
			if isLink, opens := hyperlinkState(line[i:j]); isLink {
				linkOpen = opens
			}
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

	// Close a hyperlink cut mid-text, then reset colors before the suffix.
	if linkOpen {
		b.WriteString(hyperlinkClose)
	}
	b.WriteString("\x1b[0m")
	b.WriteString(truncSuffix)
	return b.String()
}

// hyperlinkState reports whether seq is an OSC 8 hyperlink sequence and, if so,
// whether it opens a link (non-empty URI) rather than closing one.
func hyperlinkState(seq string) (isLink, opens bool) {
	rest, ok := strings.CutPrefix(seq, "\x1b]8;")
	if !ok {
		return false, false
	}
	_, uri, _ := strings.Cut(rest, ";")
	uri = strings.TrimSuffix(strings.TrimSuffix(uri, "\x07"), "\x1b\\")
	return true, uri != ""
}
