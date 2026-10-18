package main

import (
	"bytes"
	"os"
	"path/filepath"
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

	t.Run("out-of-range globals fail", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		s := config.DefaultSettings()
		s.FlexMode = "fullish"
		s.ColorLevel = 7
		s.CompactThreshold = 150
		require.NoError(t, config.Save(&s))
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Contains(t, errOut, `invalid flexMode "fullish"`)
		assert.Contains(t, errOut, "colorLevel 7 out of range 0-3")
		assert.Contains(t, errOut, "compactThreshold 150 out of range 0-100")
	})

	t.Run("display modes are checked per widget", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{
			{ID: "1", Type: "rate-limit-5h", Metadata: map[string]string{"display": "bars"}},
			{ID: "2", Type: "model", Metadata: map[string]string{"display": "full"}},
			{ID: "3", Type: "block-timer", Metadata: map[string]string{"display": "progress"}},
		}})
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Contains(t, errOut, `line 1 widget 1: invalid metadata.display "bars" (want one of percent, bar, reset, full)`)
		assert.Contains(t, errOut, "line 1 widget 2: model does not use metadata.display")
		assert.NotContains(t, errOut, "widget 3")
	})

	t.Run("thresholds need a numeric widget and valid colors", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{
			{ID: "1", Type: "model", Thresholds: []config.Threshold{{GTE: 1, Color: "red"}}},
			{ID: "2", Type: "context-percentage", Thresholds: []config.Threshold{{GTE: 70, Color: "amber"}}},
			{ID: "3", Type: "spend-limit", Thresholds: []config.Threshold{{GTE: 100, Color: "red", BackgroundColor: "white"}}},
		}})
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Contains(t, errOut, "line 1 widget 1: model does not support thresholds")
		assert.Contains(t, errOut, `line 1 widget 2 threshold 1: invalid color "amber"`)
		assert.NotContains(t, errOut, "widget 3")
	})

	t.Run("malformed JSON reports path, line and column", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "ccstatus"), 0o755))
		require.NoError(t, os.WriteFile(config.Path(), []byte("{\n  \"colorLevel\": 2,\n}"), 0o600))
		_, _, err := execValidate(t)
		require.Error(t, err)
		assert.Contains(t, err.Error(), config.Path()+":3:1: invalid character '}'")
	})

	t.Run("does not print valid when problems exist", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{{ID: "1", Type: "no-such-widget"}}})
		out, _, err := execValidate(t)
		require.Error(t, err)
		assert.NotContains(t, out, "Settings are valid")
	})
}
