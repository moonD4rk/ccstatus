package main

import (
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
