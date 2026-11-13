package render

import (
	"strings"

	"github.com/moond4rk/ccstatus/internal/color"
)

const (
	truncSuffix       = "..."
	widgetTruncSuffix = "…"
	hyperlinkClose    = "\x1b]8;;\x07"
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
	return cut(line, target) + "\x1b[0m" + truncSuffix
}

// truncateWidget caps one widget's text at maxWidth columns, ending it with a one-column "…";
// the segment's own colors still apply, so no reset is added.
func truncateWidget(text string, maxWidth int) string {
	if maxWidth <= 0 || color.VisibleWidth(text) <= maxWidth {
		return text
	}
	return strings.TrimRight(cut(text, maxWidth-color.VisibleWidth(widgetTruncSuffix)), " ") + widgetTruncSuffix
}

// cut keeps the longest prefix of line that fits in width columns, copying escape
// sequences through and closing a hyperlink it cuts mid-text. It never splits a
// grapheme cluster, and measures as color.VisibleWidth does.
func cut(line string, width int) string {
	keep := fittingBytes(color.StripANSI(line), width)
	var b strings.Builder
	b.Grow(len(line))
	linkOpen := false
	for i := 0; i < len(line); {
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
		if keep == 0 {
			break
		}
		b.WriteByte(line[i])
		keep--
		i++
	}

	if linkOpen {
		b.WriteString(hyperlinkClose)
	}
	return b.String()
}

// fittingBytes returns the byte length of the longest run of whole grapheme
// clusters at the start of plain that fits in width columns.
func fittingBytes(plain string, width int) int {
	used, n := 0, 0
	for cluster, w := range color.Clusters(plain) {
		if used+w > width {
			break
		}
		used += w
		n += len(cluster)
	}
	return n
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
