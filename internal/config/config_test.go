package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()

	assert.Equal(t, CurrentVersion, s.Version)
	assert.Equal(t, 2, s.ColorLevel)
	assert.Equal(t, "full-minus-40", s.FlexMode)
	assert.Equal(t, 60, s.CompactThreshold)
	assert.Equal(t, "|", s.DefaultSeparator)
	assert.Equal(t, " ", s.DefaultPadding)
	require.Len(t, s.Lines, 2)
	// Line 1: model | ctx-% | git-branch | +added -removed | cost
	require.Len(t, s.Lines[0], 10)
	assert.Equal(t, "model", s.Lines[0][0].Type)
	assert.Equal(t, "cyan", s.Lines[0][0].Color)
	assert.Equal(t, "context-percentage", s.Lines[0][2].Type)
	assert.Equal(t, "git-branch", s.Lines[0][4].Type)
	assert.Equal(t, "lines-added", s.Lines[0][6].Type)
	assert.Equal(t, "green", s.Lines[0][6].Color)
	assert.Equal(t, "lines-removed", s.Lines[0][7].Type)
	assert.Equal(t, "red", s.Lines[0][7].Color)
	assert.Equal(t, "session-cost", s.Lines[0][9].Type)
	// Line 2: full path
	require.Len(t, s.Lines[1], 1)
	assert.Equal(t, "current-working-dir", s.Lines[1][0].Type)
	assert.True(t, s.Lines[1][0].RawValue)
}

func TestLoadMissingFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	s, err := Load()
	require.NoError(t, err)
	assert.Equal(t, DefaultSettings(), s)
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	original := DefaultSettings()
	original.ColorLevel = 3
	original.FlexMode = "full"

	err := Save(&original)
	require.NoError(t, err)

	// Verify file exists
	configPath := filepath.Join(tmpDir, "ccstatus", "settings.json")
	_, err = os.Stat(configPath)
	require.NoError(t, err)

	loaded, err := Load()
	require.NoError(t, err)
	assert.Equal(t, original.Version, loaded.Version)
	assert.Equal(t, 3, loaded.ColorLevel)
	assert.Equal(t, "full", loaded.FlexMode)
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	dir := filepath.Join(tmpDir, "ccstatus")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "settings.json"), []byte("{invalid}"), 0o600))

	_, err := Load()
	assert.Error(t, err)
}

func TestWidgetItem_IsMerged(t *testing.T) {
	tests := []struct {
		name  string
		merge any
		want  bool
		noPad bool
	}{
		{name: "nil merge", merge: nil, want: false, noPad: false},
		{name: "true merge", merge: true, want: true, noPad: false},
		{name: "false merge", merge: false, want: false, noPad: false},
		{name: "no-padding merge", merge: "no-padding", want: true, noPad: true},
		{name: "unknown string", merge: "unknown", want: false, noPad: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := WidgetItem{Merge: tt.merge}
			assert.Equal(t, tt.want, w.IsMerged())
			assert.Equal(t, tt.noPad, w.MergeNoPadding())
		})
	}
}
