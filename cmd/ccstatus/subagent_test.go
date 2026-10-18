package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
)

// execRoot runs the root command with args and stdin, returning stdout and
// stderr.
func execRoot(t *testing.T, stdin string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := newRootCmd()
	cmd.SetArgs(args)
	cmd.SetIn(strings.NewReader(stdin))
	var o, e bytes.Buffer
	cmd.SetOut(&o)
	cmd.SetErr(&e)
	err = cmd.Execute()
	return o.String(), e.String(), err
}

// plain strips ANSI codes and restores the non-breaking spaces PostProcess adds.
func plain(s string) string {
	return strings.ReplaceAll(color.StripANSI(s), " ", " ")
}

// writeSubagentConfig saves a colorless config with a time-independent subagentLine.
func writeSubagentConfig(t *testing.T) {
	t.Helper()
	writeDeterministicConfig(t)
	s, err := config.Load()
	require.NoError(t, err)
	s.SubagentLine = []config.WidgetItem{
		{ID: "s1", Type: "task-name"},
		{ID: "s2", Type: "task-status"},
		{ID: "s3", Type: "separator"},
		{ID: "s4", Type: "task-context"},
	}
	require.NoError(t, config.Save(&s))
}

func TestSubagentCommand(t *testing.T) {
	writeSubagentConfig(t)

	t.Run("prints one JSON line per rendered row", func(t *testing.T) {
		in := `{"columns": 60, "tasks": [
			{"id": "t1", "name": "reviewer", "status": "running", "tokenCount": 50000, "contextWindowSize": 200000},
			{"id": "t2"},
			{"name": "no id"}
		]}`
		out, _, err := execRoot(t, in, "subagent")
		require.NoError(t, err)

		var rows []subagentRow
		sc := bufio.NewScanner(strings.NewReader(out))
		for sc.Scan() {
			var row subagentRow
			require.NoError(t, json.Unmarshal(sc.Bytes(), &row), sc.Text())
			rows = append(rows, row)
		}
		require.Len(t, rows, 1, "empty rows and tasks without an id are left to Claude Code")
		assert.Equal(t, "t1", rows[0].ID)
		assert.Equal(t, "reviewer running | 25%", plain(rows[0].Content))
	})

	t.Run("invalid input prints nothing and succeeds", func(t *testing.T) {
		out, errOut, err := execRoot(t, "not json", "subagent")
		require.NoError(t, err)
		assert.Empty(t, out)
		assert.Contains(t, errOut, "invalid subagent input")
	})

	t.Run("broken settings print nothing and succeed", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "ccstatus"), 0o755))
		require.NoError(t, os.WriteFile(config.Path(), []byte("{"), 0o600))
		out, _, err := execRoot(t, `{"tasks":[{"id":"t1","name":"x"}]}`, "subagent")
		require.NoError(t, err)
		assert.Empty(t, out)
	})
}

func TestPreviewCommand(t *testing.T) {
	writeSubagentConfig(t)

	t.Run("renders the docs sample", func(t *testing.T) {
		out, _, err := execRoot(t, "", "preview")
		require.NoError(t, err)
		assert.Equal(t, "Opus | Ctx: 8%\n", plain(out))
	})

	t.Run("width override truncates", func(t *testing.T) {
		out, _, err := execRoot(t, "", "preview", "--width", "18")
		require.NoError(t, err)
		assert.Equal(t, "Opus ...\n", plain(out), "18 columns minus the 10-column flex margin")
	})

	t.Run("subagent rows", func(t *testing.T) {
		out, _, err := execRoot(t, "", "preview", "--subagent")
		require.NoError(t, err)
		assert.Equal(t, "reviewer running | 23%\nExplore completed | 9%\n", plain(out))
	})

	t.Run("input file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "payload.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"model":{"display_name":"Haiku"}}`), 0o600))
		out, _, err := execRoot(t, "", "preview", "--input", path)
		require.NoError(t, err)
		assert.Equal(t, "Haiku\n", plain(out))
	})
}

func TestDumpAppend(t *testing.T) {
	writeDeterministicConfig(t)
	t.Setenv("COLUMNS", "132")
	path := filepath.Join(t.TempDir(), "capture.jsonl")

	for _, in := range []string{`{"model":{"display_name":"Opus"}}`, "not json"} {
		_, _, err := execRoot(t, in, "dump", "--append", "-o", path)
		require.NoError(t, err)
	}

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	require.Len(t, lines, 2)

	var first, second dumpRecord
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &first))
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &second))
	assert.NotEmpty(t, first.TS)
	require.NotNil(t, first.Columns)
	assert.Equal(t, 132, *first.Columns)
	assert.JSONEq(t, `{"model":{"display_name":"Opus"}}`, string(first.Input))
	assert.JSONEq(t, `"not json"`, string(second.Input), "invalid input is kept as a string")
}

func TestValidateWidgetScopes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := config.DefaultSettings()
	s.Lines = [][]config.WidgetItem{{{ID: "1", Type: "model"}, {ID: "2", Type: "task-name"}}}
	s.SubagentLine = []config.WidgetItem{
		{ID: "s1", Type: "task-name"},
		{ID: "s2", Type: "separator"},
		{ID: "s3", Type: "git-branch"},
		{ID: "1", Type: "custom-text", CustomText: "dup"},
	}
	require.NoError(t, config.Save(&s))

	_, errOut, err := execValidate(t)
	require.Error(t, err)
	assert.Contains(t, errOut, "line 1 widget 2: task-name only works in subagentLine")
	assert.Contains(t, errOut, "subagentLine widget 3: git-branch only works on the main status line")
	assert.Contains(t, errOut, `subagentLine widget 4: duplicate id "1"`)
	assert.NotContains(t, errOut, "subagentLine widget 2")
}

func TestInstallSubagentFlagAndUninstallReport(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)

	_, errOut, err := execRoot(t, "", "install", "--subagent")
	require.NoError(t, err)
	assert.Contains(t, errOut, "ccstatus installed successfully")
	data, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"command": "ccstatus subagent"`)
	assert.Contains(t, string(data), `"refreshInterval": 30`)

	_, errOut, err = execRoot(t, "", "uninstall")
	require.NoError(t, err)
	assert.Contains(t, errOut, "ccstatus uninstalled (statusLine, subagentStatusLine)")

	_, errOut, err = execRoot(t, "", "uninstall")
	require.NoError(t, err)
	assert.Contains(t, errOut, "ccstatus is not installed")
}
