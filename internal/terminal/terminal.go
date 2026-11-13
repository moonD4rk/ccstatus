// Package terminal provides terminal width detection.
package terminal

import (
	"io"
	"os"
	"strconv"

	"golang.org/x/term"
)

// defaultWidth is the fallback terminal width when detection fails.
const defaultWidth = 80

// Width returns the terminal width in columns.
//
// If override > 0 it wins (the `terminalWidth` setting). Otherwise the order is:
//  1. COLUMNS env var: Claude Code (v2.1.153+) captures the status line
//     command's output, so its fds are not a terminal, and sets COLUMNS/LINES to
//     the live dimensions before each run.
//  2. stdout/stderr fd: direct use in a terminal, such as `ccstatus preview`.
//  3. the default 80.
func Width(override int) int {
	if override > 0 {
		return override
	}
	if w := widthFromColumns(); w > 0 {
		return w
	}
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 { //nolint:gosec // fd values are small, no overflow risk
			return w
		}
	}
	return defaultWidth
}

// IsTerminal reports whether r is a terminal, where a read would wait for the
// user to type rather than receive piped data.
func IsTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	return ok && term.IsTerminal(int(f.Fd())) //nolint:gosec // fd values are small, no overflow risk
}

// widthFromColumns parses the COLUMNS environment variable, returning 0 when it
// is unset or not a positive integer.
func widthFromColumns() int {
	cols := os.Getenv("COLUMNS")
	if cols == "" {
		return 0
	}
	if n, err := strconv.Atoi(cols); err == nil && n > 0 {
		return n
	}
	return 0
}
