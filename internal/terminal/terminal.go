// Package terminal provides terminal width detection.
package terminal

import (
	"os"
	"strconv"

	"golang.org/x/term"
)

// defaultWidth is the fallback terminal width when detection fails.
const defaultWidth = 80

// Width returns the terminal width in columns.
// Detection order: stdout fd, stderr fd, /dev/tty, COLUMNS env, default 80.
// When running as a piped subprocess (e.g. Claude Code status line command),
// stdout/stderr are pipes, so /dev/tty is typically the first successful source.
func Width() int {
	// Try stdout and stderr fds (works when at least one is a real terminal).
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 { //nolint:gosec // fd values are small, no overflow risk
			return w
		}
	}
	// Try the controlling terminal directly. This works even when all
	// standard fds are pipes, as long as a controlling terminal exists.
	if w := widthFromTTY(); w > 0 {
		return w
	}
	// Fall back to the COLUMNS environment variable.
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if n, err := strconv.Atoi(cols); err == nil && n > 0 {
			return n
		}
	}
	return defaultWidth
}

// widthFromTTY opens /dev/tty and queries its width.
func widthFromTTY() int {
	f, err := os.Open("/dev/tty")
	if err != nil {
		return 0
	}
	defer f.Close()
	w, _, err := term.GetSize(int(f.Fd())) //nolint:gosec // fd values are small, no overflow risk
	if err != nil {
		return 0
	}
	return w
}
