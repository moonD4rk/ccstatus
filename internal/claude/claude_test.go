package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDir(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", "")
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(home, ".claude"), Dir())
	})

	t.Run("env override", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", "/tmp/custom-claude")
		assert.Equal(t, "/tmp/custom-claude", Dir())
	})
}

func TestSettingsPath(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "/tmp/test-claude")
	assert.Equal(t, "/tmp/test-claude/settings.json", SettingsPath())
}

func TestInstall(t *testing.T) {
	t.Run("fresh install", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		path, err := Install(0, false)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(tmpDir, "settings.json"), path)

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var settings map[string]any
		require.NoError(t, json.Unmarshal(data, &settings))

		sl, ok := settings["statusLine"].(map[string]any)
		require.True(t, ok, "statusLine should be a map")
		assert.Equal(t, "command", sl["type"])
		assert.Equal(t, "ccstatus", sl["command"])
		assert.InDelta(t, 0, sl["padding"], 0.01)
		_, hasRefresh := sl["refreshInterval"]
		assert.False(t, hasRefresh, "refreshInterval should be omitted when 0")
		_, hasHideVim := sl["hideVimModeIndicator"]
		assert.False(t, hasHideVim, "hideVimModeIndicator should be omitted when false")
	})

	t.Run("with flags", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		_, err := Install(5, true)
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
		require.NoError(t, err)

		var settings map[string]any
		require.NoError(t, json.Unmarshal(data, &settings))

		sl, ok := settings["statusLine"].(map[string]any)
		require.True(t, ok)
		assert.InDelta(t, 5, sl["refreshInterval"], 0.01)
		assert.Equal(t, true, sl["hideVimModeIndicator"])
	})

	t.Run("preserves existing fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		existing := `{
  "theme": "dark",
  "autoUpdaterStatus": "disabled"
}
`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "settings.json"), []byte(existing), 0o600))

		_, err := Install(0, false)
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
		require.NoError(t, err)

		var settings map[string]any
		require.NoError(t, json.Unmarshal(data, &settings))

		assert.Equal(t, "dark", settings["theme"])
		assert.Equal(t, "disabled", settings["autoUpdaterStatus"])

		sl, ok := settings["statusLine"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "ccstatus", sl["command"])
	})

	t.Run("idempotent", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		_, err := Install(0, false)
		require.NoError(t, err)

		_, err = Install(0, false)
		require.NoError(t, err)

		data, err := os.ReadFile(filepath.Join(tmpDir, "settings.json"))
		require.NoError(t, err)

		var settings map[string]any
		require.NoError(t, json.Unmarshal(data, &settings))

		sl, ok := settings["statusLine"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "ccstatus", sl["command"])
	})
}

func TestInstallInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "settings.json"), []byte("{invalid}"), 0o600))

	_, err := Install(0, false)
	assert.Error(t, err)
}

func TestUninstall(t *testing.T) {
	t.Run("removes statusLine", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		existing := `{
  "theme": "dark",
  "statusLine": {
    "type": "command",
    "command": "ccstatus",
    "padding": 0
  }
}
`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "settings.json"), []byte(existing), 0o600))

		path, err := Uninstall()
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(tmpDir, "settings.json"), path)

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var settings map[string]any
		require.NoError(t, json.Unmarshal(data, &settings))

		assert.Equal(t, "dark", settings["theme"])
		_, ok := settings["statusLine"]
		assert.False(t, ok, "statusLine should be removed")
	})

	t.Run("no-op when file does not exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		_, err := Uninstall()
		require.NoError(t, err)

		_, err = os.Stat(filepath.Join(tmpDir, "settings.json"))
		assert.True(t, os.IsNotExist(err), "file should not be created")
	})

	t.Run("no-op when statusLine not present", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

		existing := `{
  "theme": "dark"
}
`
		settingsPath := filepath.Join(tmpDir, "settings.json")
		require.NoError(t, os.WriteFile(settingsPath, []byte(existing), 0o600))

		_, err := Uninstall()
		require.NoError(t, err)

		data, err := os.ReadFile(settingsPath)
		require.NoError(t, err)

		var settings map[string]any
		require.NoError(t, json.Unmarshal(data, &settings))
		assert.Equal(t, "dark", settings["theme"])
	})
}

func TestUninstallInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", tmpDir)

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "settings.json"), []byte("{invalid}"), 0o600))

	_, err := Uninstall()
	assert.Error(t, err)
}
