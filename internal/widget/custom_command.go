package widget

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/moond4rk/ccstatus/internal/config"
)

const defaultCommandTimeout = 3 * time.Second

// CustomCommandWidget executes a shell command and displays its output.
type CustomCommandWidget struct{}

// Render executes the configured command and returns its first line of stdout.
// The command receives the full JSON session data via stdin.
func (w *CustomCommandWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	cmdPath := item.CommandPath
	if cmdPath == "" {
		return ""
	}

	timeout := defaultCommandTimeout
	if item.Timeout > 0 {
		timeout = time.Duration(item.Timeout) * time.Millisecond
	}

	cmdCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "sh", "-c", cmdPath)

	// Pipe JSON data to stdin so the command can use it.
	if ctx.Data != nil {
		if data, err := json.Marshal(ctx.Data); err == nil {
			cmd.Stdin = strings.NewReader(string(data))
		}
	}

	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	result := strings.TrimSpace(string(out))

	// Take only the first line.
	if idx := strings.IndexByte(result, '\n'); idx >= 0 {
		result = result[:idx]
	}

	// Apply maxWidth truncation.
	if item.MaxWidth > 0 && len(result) > item.MaxWidth {
		result = result[:item.MaxWidth]
	}

	// Strip ANSI codes unless preserveColors is set.
	if !item.PreserveColors {
		result = stripANSI(result)
	}

	return result
}

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip until we find the terminating letter.
			j := i + 2
			for j < len(s) && (s[j] < 'A' || s[j] > 'Z') && (s[j] < 'a' || s[j] > 'z') {
				j++
			}
			if j < len(s) {
				j++ // skip the terminating letter
			}
			i = j
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// DefaultColor returns the default foreground color.
func (w *CustomCommandWidget) DefaultColor() string { return "white" }

// DisplayName returns the human-readable name.
func (w *CustomCommandWidget) DisplayName() string { return "Custom Command" }

// Description returns what this widget shows.
func (w *CustomCommandWidget) Description() string { return "Output from a shell command" }

// SupportsRawValue returns false since this widget has no compact mode.
func (w *CustomCommandWidget) SupportsRawValue() bool { return false }
