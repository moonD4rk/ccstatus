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
// It tries term.GetSize first, then falls back to the COLUMNS environment
// variable, and finally returns defaultWidth (80) if both fail.
func Width() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil && width > 0 {
		return width
	}
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if n, err := strconv.Atoi(cols); err == nil && n > 0 {
			return n
		}
	}
	return defaultWidth
}
