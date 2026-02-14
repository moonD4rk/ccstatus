package render

import (
	"strings"
	"unicode/utf8"

	"github.com/moond4rk/ccstatus/internal/color"
)

const truncSuffix = "..."

// Truncate shortens a line to fit within maxWidth visible characters.
// ANSI escape sequences are preserved in the output but do not count toward width.
// A "..." suffix is appended when truncation occurs.
func Truncate(line string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	visible := color.VisibleWidth(line)
	if visible <= maxWidth {
		return line
	}

	suffixWidth := utf8.RuneCountInString(truncSuffix)
	target := maxWidth - suffixWidth
	if target <= 0 {
		return truncSuffix[:maxWidth]
	}

	var result strings.Builder
	result.Grow(len(line))
	visCount := 0
	i := 0
	for i < len(line) && visCount < target {
		// Skip ANSI escape sequences
		if i+1 < len(line) && line[i] == '\x1b' && line[i+1] == '[' {
			j := i + 2
			for j < len(line) && !isTerminator(line[j]) {
				j++
			}
			if j < len(line) {
				j++
			}
			result.WriteString(line[i:j])
			i = j
			continue
		}
		// Copy one rune
		_, size := utf8.DecodeRuneInString(line[i:])
		result.WriteString(line[i : i+size])
		visCount++
		i += size
	}

	// Reset colors before suffix
	result.WriteString("\x1b[0m")
	result.WriteString(truncSuffix)
	return result.String()
}

func isTerminator(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}
