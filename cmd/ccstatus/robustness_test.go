package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moond4rk/ccstatus/internal/claude"
	"github.com/moond4rk/ccstatus/internal/config"
)

// panicOnce panics on its first write and records the later ones, standing in
// for any code on the render path that panics.
type panicOnce struct {
	bytes.Buffer
	fired bool
}

func (p *panicOnce) Write(b []byte) (int, error) {
	if !p.fired {
		p.fired = true
		panic("boom")
	}
	return p.Buffer.Write(b)
}

func TestRenderStatusLineSurvivesPanic(t *testing.T) {
	writeDeterministicConfig(t)
	// A field of the wrong type is reported on stderr, which panics here.
	payload := `{"model":{"id":"m","display_name":"Opus"},"cost":{"total_cost_usd":[]}}`

	var stdout bytes.Buffer
	stderr := &panicOnce{}
	require.NotPanics(t, func() {
		renderStatusLine(&stdout, stderr, []byte(payload), nil, 120)
	})
	assert.Equal(t, "ccstatus: internal error: boom (please report it)\n", plain(stdout.String()))
	assert.Contains(t, stderr.String(), "ccstatus: panic: boom")
	assert.Contains(t, stderr.String(), "renderStatusLine", "the stack is logged for claude --debug")
}

func TestSubagentRowsSurvivePanic(t *testing.T) {
	writeSubagentConfig(t)
	payload := `{"columns":80,"tasks":[{"id":"t1","name":"a","status":"running","tokenCount":"many"}]}`

	var stdout bytes.Buffer
	stderr := &panicOnce{}
	require.NotPanics(t, func() {
		writeSubagentRows(&stdout, stderr, []byte(payload), nil)
	})
	assert.Empty(t, stdout.String(), "no rows, so Claude Code keeps its own")
	assert.Contains(t, stderr.String(), "ccstatus: panic: boom")
}

func TestInitReportsThroughTheCommand(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stdout, stderr, err := execRoot(t, "", "init")
	require.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Equal(t, "Created "+config.Path()+"\n", stderr)
}

func TestCommandsWithoutAHomeDirectory(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Chdir(t.TempDir())

	for _, args := range [][]string{{"init"}, {"validate"}} {
		_, _, err := execRoot(t, "", args...)
		require.ErrorIs(t, err, config.ErrNoConfigDir, strings.Join(args, " "))
	}
	for _, args := range [][]string{{"install"}, {"uninstall"}} {
		_, _, err := execRoot(t, "", args...)
		require.ErrorIs(t, err, claude.ErrNoConfigDir, strings.Join(args, " "))
	}

	t.Run("rendering falls back to the defaults", func(t *testing.T) {
		t.Setenv("COLUMNS", "200")
		out, err := runRoot(t, `{"model":{"id":"m","display_name":"Opus"}}`)
		require.NoError(t, err)
		assert.Equal(t, "Opus\n", out)
	})

	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	assert.Empty(t, entries, "nothing is written relative to the working directory")
}

func TestAwaitsTypedInput(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetIn(strings.NewReader("{}"))
	assert.False(t, awaitsTypedInput(cmd), "a reader that is not a file")

	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()
	defer w.Close()
	cmd.SetIn(r)
	assert.False(t, awaitsTypedInput(cmd), "a pipe, as Claude Code provides")
}
