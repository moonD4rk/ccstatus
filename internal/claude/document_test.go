package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeSettingsFile(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "settings.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestInstallPreservesKeyOrderAndValues(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	path := writeSettingsFile(t, dir, `{
  "zebra": 1.50,
  "permissions": {"allow": ["Bash(git:*)", "Read(//tmp/**)"]},
  "statusLine": {"type": "command", "command": "ccstatus", "padding": 0},
  "alpha": "a & b <c>"
}
`)

	_, err := Install(InstallOptions{})
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	want := `{
  "zebra": 1.50,
  "permissions": {
    "allow": [
      "Bash(git:*)",
      "Read(//tmp/**)"
    ]
  },
  "statusLine": {
    "type": "command",
    "command": "ccstatus",
    "padding": 0
  },
  "alpha": "a & b <c>"
}
`
	assert.Equal(t, want, string(data), "keys keep their order, numbers and strings their exact text")
}

func TestInstallFollowsSymlinkAndKeepsMode(t *testing.T) {
	dotfiles := t.TempDir()
	target := filepath.Join(dotfiles, "claude-settings.json")
	require.NoError(t, os.WriteFile(target, []byte(`{"theme": "dark"}`), 0o644)) //nolint:gosec // a non-default mode to check it survives

	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	link := filepath.Join(dir, "settings.json")
	require.NoError(t, os.Symlink(target, link))

	_, err := Install(InstallOptions{})
	require.NoError(t, err)

	info, err := os.Lstat(link)
	require.NoError(t, err)
	assert.Equal(t, os.ModeSymlink, info.Mode()&os.ModeSymlink, "settings.json is still a symlink")

	data, err := os.ReadFile(target)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"theme": "dark"`)
	assert.Contains(t, string(data), `"command": "ccstatus"`)

	targetInfo, err := os.Stat(target)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), targetInfo.Mode().Perm())

	leftovers, err := filepath.Glob(filepath.Join(dotfiles, ".settings-*"))
	require.NoError(t, err)
	assert.Empty(t, leftovers, "no temp files left behind")
}

func TestInstallSubagent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)

	path, err := Install(InstallOptions{Subagent: ptr(true)})
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"subagentStatusLine": {
    "type": "command",
    "command": "ccstatus subagent"
  }`)

	_, err = Install(InstallOptions{Subagent: ptr(false)})
	require.NoError(t, err)
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "subagentStatusLine")

	t.Run("false leaves another tool's subagent line alone", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", dir)
		path := writeSettingsFile(t, dir, `{"subagentStatusLine": {"type": "command", "command": "~/bin/rows.sh"}}`)
		_, err := Install(InstallOptions{Subagent: ptr(false)})
		require.NoError(t, err)
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(data), "~/bin/rows.sh")
	})
}

func TestInstallReplacingAnotherToolIsFresh(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	path := writeSettingsFile(t, dir, `{"statusLine": {"type": "command", "command": "~/.claude/statusline.sh", "padding": 2}}`)

	_, err := Install(InstallOptions{})
	require.NoError(t, err)
	sl := readStatusLine(t, path)
	assert.Equal(t, "ccstatus", sl["command"])
	assert.InDelta(t, 2, sl["padding"], 0.01, "existing padding survives")
	assert.InDelta(t, DefaultRefreshInterval, sl["refreshInterval"], 0.01)

	t.Run("refresh 0 removes it", func(t *testing.T) {
		_, err := Install(InstallOptions{RefreshInterval: ptr(0)})
		require.NoError(t, err)
		_, has := readStatusLine(t, path)["refreshInterval"]
		assert.False(t, has)
	})

	t.Run("re-install keeps an install without refreshInterval as is", func(t *testing.T) {
		_, err := Install(InstallOptions{})
		require.NoError(t, err)
		_, has := readStatusLine(t, path)["refreshInterval"]
		assert.False(t, has)
	})
}

func TestInstallEmptyFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	path := writeSettingsFile(t, dir, "")
	_, err := Install(InstallOptions{})
	require.NoError(t, err)
	assert.Equal(t, "ccstatus", readStatusLine(t, path)["command"])
}

func TestUninstallOnlyRemovesCCStatus(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", dir)
	path := writeSettingsFile(t, dir, `{
  "statusLine": {"type": "command", "command": "~/.claude/statusline.sh"},
  "subagentStatusLine": {"type": "command", "command": "/opt/homebrew/bin/ccstatus subagent"},
  "theme": "dark"
}`)

	res, err := Uninstall()
	require.NoError(t, err)
	assert.Equal(t, []string{"subagentStatusLine"}, res.Removed)
	assert.Equal(t, []string{"statusLine"}, res.Kept)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "statusline.sh")
	assert.NotContains(t, string(data), "subagentStatusLine")

	t.Run("nothing of ours leaves the file byte for byte", func(t *testing.T) {
		before, err := os.ReadFile(path)
		require.NoError(t, err)
		res, err := Uninstall()
		require.NoError(t, err)
		assert.Empty(t, res.Removed)
		after, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, before, after)
	})
}

func TestRunsCCStatus(t *testing.T) {
	tests := map[string]bool{
		`{"command": "ccstatus"}`:                   true,
		`{"command": "ccstatus dump --append"}`:     true,
		`{"command": "/opt/homebrew/bin/ccstatus"}`: true,
		`{"command": "  ccstatus subagent "}`:       true,
		`{"command": "~/.claude/statusline.sh"}`:    false,
		`{"command": "ccstatusline"}`:               false,
		`{"command": ""}`:                           false,
		`"ccstatus"`:                                false,
	}
	for raw, want := range tests {
		assert.Equal(t, want, runsCCStatus([]byte(raw)), raw)
	}
}

func TestParseDocumentRejectsNonObjects(t *testing.T) {
	for _, input := range []string{`[]`, `"x"`, `{"a":1} {"b":2}`, `{"a":`} {
		_, err := parseDocument([]byte(input))
		require.Error(t, err, input)
	}
	doc, err := parseDocument([]byte(`{"a":1,"a":2}`))
	require.NoError(t, err)
	assert.Equal(t, []string{"a"}, doc.keys, "duplicate keys collapse to the last value")
	out, err := doc.encode()
	require.NoError(t, err)
	assert.Equal(t, "{\n  \"a\": 2\n}\n", string(out))
	assert.True(t, strings.HasSuffix(string(out), "\n"))
}
