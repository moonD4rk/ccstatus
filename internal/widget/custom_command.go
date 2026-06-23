package widget

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
)

const defaultCommandTimeout = 3 * time.Second

// CustomCommandWidget executes a shell command and displays its output.
type CustomCommandWidget struct{ noAffix }

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
	if err != nil {
		return ""
	}

	result := strings.TrimSpace(string(out))

	// Take only the first line.
	if idx := strings.IndexByte(result, '\n'); idx >= 0 {
		result = result[:idx]
	}

	// Strip ANSI codes unless preserveColors is set, before measuring width so
	// maxWidth counts visible characters rather than escape bytes.
	if !item.PreserveColors {
		result = color.StripANSI(result)
	}

	// Apply maxWidth truncation by runes so multibyte output is never cut
	// mid-codepoint.
	if item.MaxWidth > 0 {
		if runes := []rune(result); len(runes) > item.MaxWidth {
			result = string(runes[:item.MaxWidth])
		}
	}

	return result
}

// DefaultColor returns the default foreground color.
func (w *CustomCommandWidget) DefaultColor() string { return "white" }

// DisplayName returns the human-readable name.
func (w *CustomCommandWidget) DisplayName() string { return "Custom Command" }

// Description returns what this widget shows.
func (w *CustomCommandWidget) Description() string { return "Output from a shell command" }

// SupportsRawValue returns false since this widget has no compact mode.
func (w *CustomCommandWidget) SupportsRawValue() bool { return false }
