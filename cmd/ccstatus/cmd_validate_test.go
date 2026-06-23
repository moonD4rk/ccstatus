package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moond4rk/ccstatus/internal/config"
)

// saveConfigWithLines points XDG_CONFIG_HOME at a temp dir and writes a config
// whose Lines are replaced with the given widgets.
func saveConfigWithLines(t *testing.T, lines [][]config.WidgetItem) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := config.DefaultSettings()
	s.Lines = lines
	require.NoError(t, config.Save(&s))
}

func execValidate(t *testing.T) (stdout, stderr string, err error) {
	t.Helper()
	cmd := newRootCmd()
	cmd.SetArgs([]string{"validate"})
	var o, e bytes.Buffer
	cmd.SetOut(&o)
	cmd.SetErr(&e)
	err = cmd.Execute()
	return o.String(), e.String(), err
}

func TestValidate(t *testing.T) {
	t.Run("default config is valid", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // no file -> defaults
		out, _, err := execValidate(t)
		require.NoError(t, err)
		assert.Contains(t, out, "Settings are valid")
	})

	t.Run("unknown widget type fails", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{{ID: "1", Type: "no-such-widget"}}})
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Contains(t, errOut, "unknown widget type")
	})

	t.Run("missing id fails", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{{Type: "model"}}})
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Contains(t, errOut, "missing id")
	})

	t.Run("duplicate id fails", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{
			{ID: "1", Type: "model"},
			{ID: "1", Type: "version"},
		}})
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Contains(t, errOut, "duplicate id")
	})

	t.Run("invalid color fails", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{{ID: "1", Type: "model", Color: "chartreuse"}}})
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Contains(t, errOut, "invalid color")
	})

	t.Run("does not print valid when problems exist", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{{ID: "1", Type: "no-such-widget"}}})
		out, _, err := execValidate(t)
		require.Error(t, err)
		assert.NotContains(t, out, "Settings are valid")
	})
}
