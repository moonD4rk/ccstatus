package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moond4rk/ccstatus/internal/config"
)

func TestInstallHideVimIndicatorHint(t *testing.T) {
	const hint = "add a vim-mode widget"

	tests := []struct {
		name     string
		args     []string
		lines    [][]config.WidgetItem
		wantHint bool
	}{
		{name: "hiding with the default layout", args: []string{"--hide-vim-indicator"}, wantHint: true},
		{
			name:  "hiding with a vim-mode widget",
			args:  []string{"--hide-vim-indicator"},
			lines: [][]config.WidgetItem{{{Type: "model"}}, {{Type: "vim-mode"}}},
		},
		{name: "unhiding", args: []string{"--hide-vim-indicator=false"}},
		{name: "flag not given", args: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			if tt.lines != nil {
				s := config.DefaultSettings()
				s.Lines = tt.lines
				require.NoError(t, config.Save(&s))
			}

			_, errOut, err := execRoot(t, "", append([]string{"install"}, tt.args...)...)
			require.NoError(t, err)
			assert.Contains(t, errOut, "ccstatus installed successfully")
			if tt.wantHint {
				assert.Contains(t, errOut, hint)
				assert.Contains(t, errOut, config.Path())
			} else {
				assert.NotContains(t, errOut, hint)
			}
		})
	}
}

func TestInstallWarnsWhenNotOnPath(t *testing.T) {
	const warning = "Warning: ccstatus is not on PATH"

	// onPath makes PATH a directory holding an executable named ccstatus.
	onPath := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		bin := filepath.Join(dir, "ccstatus")
		require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755)) //nolint:gosec // the test needs an executable
		t.Setenv("PATH", dir)
		return bin
	}

	t.Run("warns and says how to fix it", func(t *testing.T) {
		claudeDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", claudeDir)
		t.Setenv("PATH", t.TempDir())

		_, errOut, err := execRoot(t, "", "install")
		require.NoError(t, err, "the install still succeeds")
		assert.Contains(t, errOut, "ccstatus installed successfully")
		assert.Contains(t, errOut, warning)
		assert.Contains(t, errOut, "install keeps an absolute path")
		assert.Contains(t, errOut, filepath.Join(claudeDir, "settings.json"))
	})

	t.Run("quiet when ccstatus is on PATH", func(t *testing.T) {
		t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
		onPath(t)

		_, errOut, err := execRoot(t, "", "install")
		require.NoError(t, err)
		assert.NotContains(t, errOut, warning)
	})

	t.Run("quiet when the command is an absolute path", func(t *testing.T) {
		claudeDir := t.TempDir()
		t.Setenv("CLAUDE_CONFIG_DIR", claudeDir)
		bin := onPath(t)
		t.Setenv("PATH", t.TempDir()) // off PATH again, but named in full below
		settings := filepath.Join(claudeDir, "settings.json")
		require.NoError(t, os.WriteFile(settings, []byte(`{"statusLine": {"type": "command", "command": "`+bin+` dump"}}`), 0o600))

		_, errOut, err := execRoot(t, "", "install", "--subagent")
		require.NoError(t, err)
		assert.NotContains(t, errOut, warning)
		data, err := os.ReadFile(settings)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"command": "`+bin+`"`)
		assert.Contains(t, string(data), `"command": "`+bin+` subagent"`)
	})
}
