package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
)

// runRoot feeds stdin to the root command (the status-line path) and returns the
// captured stdout normalized back to plain text (ANSI stripped, the VSCode
// non-breaking spaces restored to ordinary spaces).
func runRoot(t *testing.T, stdin string) (string, error) {
	t.Helper()
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader(stdin))
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	normalized := strings.ReplaceAll(color.StripANSI(out.String()), " ", " ")
	return normalized, err
}

// writeDeterministicConfig points XDG_CONFIG_HOME at a temp dir holding a config
// with only data-driven widgets (no git/clock), so the rendered line is stable.
func writeDeterministicConfig(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := config.DefaultSettings()
	s.ColorLevel = 0
	s.TerminalWidth = 200
	s.Lines = [][]config.WidgetItem{{
		{ID: "1", Type: "model"},
		{ID: "2", Type: "separator"},
		{ID: "3", Type: "context-percentage"},
	}}
	require.NoError(t, config.Save(&s))
}

func TestRunStatusLineEndToEnd(t *testing.T) {
	writeDeterministicConfig(t)

	t.Run("renders the canonical payload", func(t *testing.T) {
		in := `{"model":{"id":"claude-opus-4-8","display_name":"Opus"},` +
			`"context_window":{"used_percentage":25,"context_window_size":200000}}`
		out, err := runRoot(t, in)
		require.NoError(t, err)
		assert.Equal(t, "Opus | Ctx: 25%\n", out)
	})

	t.Run("null payload renders nothing", func(t *testing.T) {
		out, err := runRoot(t, "null")
		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("empty stdin errors without panicking", func(t *testing.T) {
		_, err := runRoot(t, "")
		assert.Error(t, err)
	})
}

func TestRootCommandRegistersSubcommands(t *testing.T) {
	cmd := newRootCmd()
	got := map[string]bool{}
	for _, c := range cmd.Commands() {
		got[c.Name()] = true
	}
	for _, want := range []string{"init", "validate", "install", "uninstall", "dump", "widgets"} {
		assert.Truef(t, got[want], "subcommand %q should be registered", want)
	}
}
