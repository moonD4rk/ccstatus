package main

import (
	"bytes"
	"os"
	"path/filepath"
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
	cmd.SetErr(&bytes.Buffer{})
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

	t.Run("invalid input renders an error line and exits 0", func(t *testing.T) {
		out, err := runRoot(t, "")
		require.NoError(t, err)
		assert.Equal(t, "ccstatus: invalid input: unexpected end of JSON input\n", out)
	})

	t.Run("a field of the wrong type does not blank the line", func(t *testing.T) {
		in := `{"model":{"id":"m","display_name":"Opus"},"context_window":{"used_percentage":"25"},"cost":{"total_cost_usd":[]}}`
		out, err := runRoot(t, in)
		require.NoError(t, err)
		assert.Equal(t, "Opus | Ctx: 25%\n", out)
	})
}

func TestRunStatusLineWithBrokenSettings(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("COLUMNS", "300")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "ccstatus"), 0o755))
	broken := "{\n  \"lines\": [[{\"id\": \"1\", \"type\": \"model\"},]]\n}"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ccstatus", "settings.json"), []byte(broken), 0o600))

	out, err := runRoot(t, `{"model":{"id":"m","display_name":"Opus"}}`)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	require.GreaterOrEqual(t, len(lines), 2, "error line plus the default layout")
	assert.Equal(t, `ccstatus: settings.json:2:43: invalid character ']' looking for beginning of value (run ccstatus validate)`, lines[0])
	assert.True(t, strings.HasPrefix(lines[1], "Opus"), "default layout still renders: %q", lines[1])
}

func TestDumpWritesPayloadAndRenders(t *testing.T) {
	writeDeterministicConfig(t)
	path := filepath.Join(t.TempDir(), "dump.json")

	cmd := newRootCmd()
	cmd.SetArgs([]string{"dump", "-o", path})
	cmd.SetIn(strings.NewReader(`{"model":{"id":"m","display_name":"Opus"}}`))
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	require.NoError(t, cmd.Execute())

	assert.Equal(t, "Opus\n", strings.ReplaceAll(color.StripANSI(out.String()), " ", " "))
	assert.Contains(t, errOut.String(), "Dumped JSON to "+path)
	dumped, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, `{"model":{"id":"m","display_name":"Opus"}}`, string(dumped))
}

func TestRootCommandRegistersSubcommands(t *testing.T) {
	cmd := newRootCmd()
	got := map[string]bool{}
	for _, c := range cmd.Commands() {
		got[c.Name()] = true
	}
	for _, want := range []string{"init", "validate", "install", "uninstall", "dump", "widgets", "subagent", "preview"} {
		assert.Truef(t, got[want], "subcommand %q should be registered", want)
	}
}
