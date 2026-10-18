package config

import (
	"fmt"
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
	assert.Equal(t, "full-until-compact", s.FlexMode)
	assert.Equal(t, 90, s.CompactThreshold)
	assert.Equal(t, "|", s.DefaultSeparator)
	assert.Equal(t, " ", s.DefaultPadding)
	require.Len(t, s.Lines, 2)

	// Line 1: model · effort | Fast | ctx-% | branch worktree | +added -removed | $cost
	require.Len(t, s.Lines[0], 14)
	assert.Equal(t, "model", s.Lines[0][0].Type)
	assert.Equal(t, "cyan", s.Lines[0][0].Color)
	assert.Equal(t, "effort", s.Lines[0][1].Type)
	assert.Equal(t, "fast-mode", s.Lines[0][3].Type)
	assert.Equal(t, "context-percentage", s.Lines[0][5].Type)
	usage := []Threshold{{GTE: 30, Color: "white"}, {GTE: 60, Color: "yellow"}, {GTE: 90, Color: "red"}}
	assert.Equal(t, "brightBlack", s.Lines[0][5].Color)
	assert.Equal(t, usage, s.Lines[0][5].Thresholds)
	assert.Equal(t, "git-branch", s.Lines[0][7].Type)
	assert.Equal(t, "git-worktree", s.Lines[0][8].Type)
	assert.Equal(t, "lines-added", s.Lines[0][10].Type)
	assert.Equal(t, "green", s.Lines[0][10].Color)
	assert.Equal(t, "lines-removed", s.Lines[0][11].Type)
	assert.Equal(t, "red", s.Lines[0][11].Color)
	assert.Equal(t, "session-cost", s.Lines[0][13].Type)

	// Line 2: cwd | rate-limits | session-clock | session-name  (plain separator, left-aligned)
	require.Len(t, s.Lines[1], 7)
	assert.Equal(t, "current-working-dir", s.Lines[1][0].Type)
	assert.True(t, s.Lines[1][0].RawValue)
	assert.Equal(t, "separator", s.Lines[1][1].Type)
	assert.Equal(t, "rate-limits", s.Lines[1][2].Type)
	assert.Empty(t, s.Lines[1][2].Prefix) // uses the widget default ("Limit ")
	assert.Equal(t, "white", s.Lines[1][2].Color)
	assert.Equal(t, []Threshold{{GTE: 30, Color: "brightWhite"}, {GTE: 60, Color: "yellow"}, {GTE: 90, Color: "red"}},
		s.Lines[1][2].Thresholds)
	assert.Equal(t, "separator", s.Lines[1][3].Type)
	assert.Equal(t, "session-clock", s.Lines[1][4].Type)
	assert.Equal(t, "session-name", s.Lines[1][6].Type)
	assert.Equal(t, 30, s.Lines[1][6].MaxWidth)

	// Subagent row: name status elapsed | model · effort tokens context% sparkline | label
	require.Len(t, s.SubagentLine, 11)
	assert.Equal(t, "task-name", s.SubagentLine[0].Type)
	assert.Equal(t, "task-effort", s.SubagentLine[5].Type)
	assert.Equal(t, "task-context", s.SubagentLine[7].Type)
	assert.Equal(t, "brightBlack", s.SubagentLine[7].Color)
	assert.Equal(t, usage, s.SubagentLine[7].Thresholds)
	assert.Equal(t, "task-label", s.SubagentLine[10].Type)
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

	s, err := Load()
	require.Error(t, err)
	assert.Equal(t, DefaultSettings(), s, "callers still get a renderable config")
}

func TestParseMergesDefaults(t *testing.T) {
	t.Run("omitted fields keep their defaults", func(t *testing.T) {
		s, err := Parse([]byte(`{"lines":[[{"id":"1","type":"model"}]]}`))
		require.NoError(t, err)
		def := DefaultSettings()
		assert.Equal(t, def.ColorLevel, s.ColorLevel)
		assert.Equal(t, def.DefaultPadding, s.DefaultPadding)
		assert.Equal(t, def.FlexMode, s.FlexMode)
		assert.Equal(t, def.CompactThreshold, s.CompactThreshold)
		assert.Equal(t, [][]WidgetItem{{{ID: "1", Type: "model"}}}, s.Lines)
	})

	t.Run("explicit values win, including zero", func(t *testing.T) {
		s, err := Parse([]byte(`{"colorLevel":0,"defaultPadding":"","compactThreshold":50}`))
		require.NoError(t, err)
		assert.Equal(t, 0, s.ColorLevel)
		assert.Empty(t, s.DefaultPadding)
		assert.Equal(t, 50, s.CompactThreshold)
	})

	t.Run("absent or null lines keep the default layout", func(t *testing.T) {
		for _, input := range []string{`{}`, `{"lines":null}`, `null`} {
			s, err := Parse([]byte(input))
			require.NoError(t, err)
			assert.Equal(t, DefaultSettings().Lines, s.Lines, input)
		}
	})

	t.Run("explicit empty lines render nothing", func(t *testing.T) {
		s, err := Parse([]byte(`{"lines":[]}`))
		require.NoError(t, err)
		assert.Empty(t, s.Lines)
	})

	t.Run("subagentLine defaults the same way", func(t *testing.T) {
		s, err := Parse([]byte(`{"lines":[]}`))
		require.NoError(t, err)
		assert.Equal(t, DefaultSettings().SubagentLine, s.SubagentLine)

		s, err = Parse([]byte(`{"subagentLine":[{"id":"a","type":"task-name"}]}`))
		require.NoError(t, err)
		assert.Equal(t, []WidgetItem{{ID: "a", Type: "task-name"}}, s.SubagentLine)

		s, err = Parse([]byte(`{"subagentLine":[]}`))
		require.NoError(t, err)
		assert.Empty(t, s.SubagentLine)
	})
}

func TestParseReportsPosition(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLine int
		wantCol  int
	}{
		{"syntax error on one line", `{invalid}`, 1, 2},
		{"trailing comma", "{\n  \"lines\": [\n    [{\"id\": \"1\", \"type\": \"model\"},]\n  ]\n}", 3, 35},
		{"type error", "{\n  \"colorLevel\": \"high\"\n}", 2, 22},
		{"truncated input", "{\n  \"lines\": [", 2, 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.input))
			var posErr *PositionError
			require.ErrorAs(t, err, &posErr)
			assert.Equal(t, tt.wantLine, posErr.Line)
			assert.Equal(t, tt.wantCol, posErr.Column)
			assert.Contains(t, err.Error(), fmt.Sprintf("%d:%d: ", tt.wantLine, tt.wantCol))
		})
	}
}

func TestWidgetItem_IsMerged(t *testing.T) {
	tests := []struct {
		name  string
		merge any
		want  bool
	}{
		{name: "nil merge", merge: nil, want: false},
		{name: "true merge", merge: true, want: true},
		{name: "false merge", merge: false, want: false},
		{name: "no-padding merge", merge: "no-padding", want: true},
		{name: "unknown string", merge: "unknown", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := WidgetItem{Merge: tt.merge}
			assert.Equal(t, tt.want, w.IsMerged())
		})
	}
}

func TestWidgetItem_ThresholdFor(t *testing.T) {
	item := WidgetItem{Thresholds: []Threshold{
		{GTE: 90, Color: "red", Bold: true},
		{GTE: 70, Color: "yellow"},
	}}
	tests := []struct {
		value     float64
		wantColor string
		wantOK    bool
	}{
		{value: 12, wantOK: false},
		{value: 70, wantColor: "yellow", wantOK: true},
		{value: 89.9, wantColor: "yellow", wantOK: true},
		{value: 90, wantColor: "red", wantOK: true},
		{value: 140, wantColor: "red", wantOK: true},
	}
	for _, tt := range tests {
		got, ok := item.ThresholdFor(tt.value)
		assert.Equal(t, tt.wantOK, ok, tt.value)
		assert.Equal(t, tt.wantColor, got.Color, tt.value)
	}

	none := WidgetItem{}
	_, ok := none.ThresholdFor(100)
	assert.False(t, ok)
}
