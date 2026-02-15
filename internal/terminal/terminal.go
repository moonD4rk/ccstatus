// Package terminal provides terminal width detection.
package terminal

import (
	"os"

	"golang.org/x/term"
)

// Width returns the terminal width in columns, or 0 if detection fails.
func Width() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 0
	}
	return width
}
