package widget

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/proc"
)

// DefaultCommandTimeout is how long a custom-command may run when its item
// sets no timeout.
const DefaultCommandTimeout = 3 * time.Second

// CustomCommandWidget executes a shell command and displays its output.
type CustomCommandWidget struct{}

// Render executes the configured command and returns its first line of stdout.
// The command receives the full JSON session data via stdin. Everything the
// command started is killed when the timeout expires.
func (w *CustomCommandWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	cmdPath := item.CommandPath
	if cmdPath == "" {
		return ""
	}

	timeout := DefaultCommandTimeout
	if item.Timeout > 0 {
		timeout = time.Duration(item.Timeout) * time.Millisecond
	}

	cmdCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := proc.Command(cmdCtx, "sh", "-c", cmdPath)

	// Forward the exact stdin Claude Code sent so the command sees the full
	// official schema, including fields ccstatus does not model yet. Only when
	// the raw bytes are unavailable (e.g. a synthetic RenderContext in tests)
	// fall back to re-serializing the parsed session, which is necessarily lossy.
	if len(ctx.RawInput) > 0 {
		cmd.Stdin = bytes.NewReader(ctx.RawInput)
	} else if ctx.Data != nil {
		if data, err := json.Marshal(ctx.Data); err == nil {
			cmd.Stdin = bytes.NewReader(data)
		}
	}

	out, err := cmd.Output()
	// A background job that keeps stdout open is not a failure: show what the
	// command itself printed.
	if err != nil && !errors.Is(err, exec.ErrWaitDelay) {
		return ""
	}

	result := strings.TrimSpace(string(out))

	// Take only the first line.
	if idx := strings.IndexByte(result, '\n'); idx >= 0 {
		result = result[:idx]
	}

	if !item.PreserveColors {
		result = color.StripANSI(result)
	}
	return result
}

// DefaultColor returns the default foreground color.
func (w *CustomCommandWidget) DefaultColor() string { return "white" }

// Description returns what this widget shows.
func (w *CustomCommandWidget) Description() string { return "Output from a shell command" }
