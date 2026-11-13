package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

// writeSettings points XDG_CONFIG_HOME at a temp dir and writes document there
// verbatim, for what only a hand-written file can hold.
func writeSettings(t *testing.T, document string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "ccstatus"), 0o755))
	require.NoError(t, os.WriteFile(config.Path(), []byte(document), 0o600))
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

// problemLines returns validate's stderr as its list of problems.
func problemLines(stderr string) []string {
	var problems []string
	for line := range strings.Lines(stderr) {
		problems = append(problems, strings.TrimSuffix(strings.TrimPrefix(line, "Error: "), "\n"))
	}
	return problems
}

func TestValidate(t *testing.T) {
	t.Run("no settings file means the defaults", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		out, _, err := execValidate(t)
		require.NoError(t, err)
		assert.Equal(t, "No settings file at "+config.Path()+": the default settings are in use\n", out)
	})

	t.Run("the default settings are valid", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		_, _, err := execRoot(t, "", "init")
		require.NoError(t, err)
		out, errOut, err := execValidate(t)
		require.NoError(t, err, errOut)
		assert.Equal(t, "Settings are valid: "+config.Path()+"\n", out)
	})

	t.Run("unknown widget type fails", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{{Type: "no-such-widget"}, {Type: "git-brnch"}, {}}})
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Equal(t, []string{
			`line 1 widget 1: unknown widget type "no-such-widget"`,
			`line 1 widget 2: unknown widget type "git-brnch" (did you mean "git-branch"?)`,
			`line 1 widget 3: missing widget type`,
		}, problemLines(errOut))
	})

	t.Run("colors are names, hex triplets or palette indexes", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{
			{Type: "model", Color: "chartreuse"},
			{Type: "model", Color: "#ff8700", BackgroundColor: "236"},
			{Type: "model", Color: "#f80", BackgroundColor: "brightBlack"},
			{Type: "model", Color: "#ff87", BackgroundColor: "256"},
		}})
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Equal(t, []string{
			`line 1 widget 1: invalid color "chartreuse" (want a color name, #rrggbb or a palette index 0-255)`,
			`line 1 widget 4: invalid color "#ff87" (want a color name, #rrggbb or a palette index 0-255)`,
			`line 1 widget 4: invalid backgroundColor "256" (want a color name, #rrggbb or a palette index 0-255)`,
		}, problemLines(errOut))
	})

	t.Run("out-of-range globals fail", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		s := config.DefaultSettings()
		s.FlexMode = "none"
		s.ColorLevel = 7
		s.CompactThreshold = 150
		s.TerminalWidth = -1
		require.NoError(t, config.Save(&s))
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Equal(t, []string{
			`settings: invalid flexMode "none" (want one of full, full-minus-40, full-until-compact, exact)`,
			"settings: colorLevel 7 out of range 0-3",
			"settings: compactThreshold 150 out of range 0-100",
			"settings: terminalWidth must not be negative",
		}, problemLines(errOut))
	})

	t.Run("subagentHideStatuses takes documented statuses only", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())
		s := config.DefaultSettings()
		s.SubagentHideStatuses = []string{"completed", "complete", "killed"}
		require.NoError(t, config.Save(&s))
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Equal(t, []string{
			`settings: unknown subagentHideStatuses entry "complete" (want one of running, completed, failed, killed)`,
		}, problemLines(errOut))
	})

	t.Run("display modes are checked per widget", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{
			{Type: "rate-limit-5h", Display: "bars"},
			{Type: "model", Display: "full"},
			{Type: "block-timer", Display: "progress"},
			{Type: "block-timer", Display: "bar"},
		}})
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Equal(t, []string{
			`line 1 widget 1: invalid display "bars" (want one of percent, bar, reset, full)`,
			"line 1 widget 2: model has no display modes",
			`line 1 widget 3: invalid display "progress" (want one of time, bar, percent)`,
		}, problemLines(errOut))
	})

	t.Run("barWidth needs a bar to size", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{
			{Type: "context-percentage", Display: "bar", BarWidth: 20},
			{Type: "context-percentage", BarWidth: 20},
			{Type: "rate-limit-5h", Display: "reset", BarWidth: 20},
			{Type: "context-percentage", Display: "bar", BarWidth: 500},
			{Type: "context-percentage", Display: "bar", BarWidth: -2},
			{Type: "model", BarWidth: 5},
		}})
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Equal(t, []string{
			`line 1 widget 2: barWidth has no effect unless display is "bar"`,
			`line 1 widget 3: barWidth has no effect unless display is "bar"`,
			"line 1 widget 4: barWidth 500 out of range 1-100",
			"line 1 widget 5: barWidth -2 out of range 1-100",
			"line 1 widget 6: barWidth has no effect on model",
		}, problemLines(errOut))
	})

	t.Run("options that do nothing on their widget", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{
			{Type: "version", RawValue: true},
			{Type: "git-branch", RawValue: true, Character: "*"},
			{Type: "separator", Character: "·"},
			{Type: "model", Character: "x", CustomText: "y", CommandPath: "z", Timeout: 100, PreserveColors: true},
			{Type: "custom-command", CommandPath: "date", Timeout: 100, PreserveColors: true},
		}})
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Equal(t, []string{
			"line 1 widget 1: rawValue has no effect on version",
			"line 1 widget 4: character has no effect on model (it applies to separator, git-branch)",
			"line 1 widget 4: customText has no effect on model (it applies to custom-text)",
			"line 1 widget 4: commandPath has no effect on model (it applies to custom-command)",
			"line 1 widget 4: timeout has no effect on model (it applies to custom-command)",
			"line 1 widget 4: preserveColors has no effect on model (it applies to custom-command)",
		}, problemLines(errOut))
	})

	t.Run("custom widgets need their text or command", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{
			{Type: "custom-text"},
			{Type: "custom-command", Timeout: -5},
			{Type: "custom-text", CustomText: "ok"},
		}})
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Equal(t, []string{
			"line 1 widget 1: custom-text needs customText",
			"line 1 widget 2: timeout must not be negative",
			"line 1 widget 2: custom-command needs commandPath",
		}, problemLines(errOut))
	})

	t.Run("thresholds need a numeric widget and valid colors", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{
			{Type: "model", Thresholds: []config.Threshold{{GTE: 1, Color: "red"}}},
			{Type: "context-percentage", Thresholds: []config.Threshold{{GTE: 70, Color: "amber"}}},
			{Type: "spend-limit", Thresholds: []config.Threshold{{GTE: 100, Color: "red", BackgroundColor: "white"}}},
			{Type: "block-timer", Thresholds: []config.Threshold{{GTE: 80, Color: "yellow"}}},
		}})
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Equal(t, []string{
			"line 1 widget 1: model does not support thresholds",
			`line 1 widget 2 threshold 1: invalid color "amber" (want a color name, #rrggbb or a palette index 0-255)`,
		}, problemLines(errOut))
	})

	t.Run("maxWidth must not be negative", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{
			{Type: "session-name", MaxWidth: -1},
			{Type: "current-working-dir", MaxWidth: 20},
		}})
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Equal(t, []string{"line 1 widget 1: maxWidth must not be negative"}, problemLines(errOut))
	})

	t.Run("malformed JSON reports path, line and column", func(t *testing.T) {
		writeSettings(t, "{\n  \"colorLevel\": 2,\n}")
		_, _, err := execValidate(t)
		require.Error(t, err)
		assert.Contains(t, err.Error(), config.Path()+":3:1: invalid character '}'")
	})

	t.Run("a value of the wrong type reports its position", func(t *testing.T) {
		for _, item := range []string{
			`{"type": "model", "merge": "yes"}`,
			`{"type": "rate-limit-5h", "display": "bar", "barWidth": "abc"}`,
		} {
			writeSettings(t, "{\n  \"lines\": [[\n    "+item+"\n  ]]\n}")
			_, _, err := execValidate(t)
			require.Error(t, err, item)
			assert.Contains(t, err.Error(), config.Path()+":3:", item)
			assert.Contains(t, err.Error(), "cannot unmarshal string", item)
		}
	})

	t.Run("does not print valid when problems exist", func(t *testing.T) {
		saveConfigWithLines(t, [][]config.WidgetItem{{{Type: "no-such-widget"}}})
		out, _, err := execValidate(t)
		require.Error(t, err)
		assert.Empty(t, out)
		assert.EqualError(t, err, "1 problem(s) found in "+config.Path())
	})
}

func TestValidateReadsTheDocumentStrictly(t *testing.T) {
	t.Run("unknown, removed, misspelled and repeated keys", func(t *testing.T) {
		writeSettings(t, `{
  "version": 4,
  "theme": "dark",
  "colorLevl": 2,
  "lines": [[
    {"id": "1", "type": "model", "coler": "red", "tresholds": []},
    {"type": "rate-limit-5h", "metadata": {"display": "bar", "barWidht": "abc"}},
    {"type": "context-percentage", "barWidht": 20, "thresholds": [{"gt": 70, "coler": "red"}, {"gte": 90, "color": "red"}]},
    {"type": "model", "color": "red", "color": "blue"}
  ]],
  "subagentLine": [
    {"id": "s1", "type": "task-name", "bolt": true}
  ]
}`)
		_, errOut, err := execValidate(t)
		require.Error(t, err)
		assert.Equal(t, []string{
			`settings: unknown field "version" (no longer used: remove it)`,
			`settings: unknown field "theme"`,
			`settings: unknown field "colorLevl" (did you mean "colorLevel"?)`,
			`line 1 widget 1 and 1 more: unknown field "id" (no longer used: remove it)`,
			`line 1 widget 1: unknown field "coler" (did you mean "color"?)`,
			`line 1 widget 1: unknown field "tresholds" (did you mean "thresholds"?)`,
			`line 1 widget 2: unknown field "metadata" (replaced by "display" and "barWidth" on the item itself)`,
			`line 1 widget 3: unknown field "barWidht" (did you mean "barWidth"?)`,
			`line 1 widget 3 threshold 1: unknown field "gt" (did you mean "gte"?)`,
			`line 1 widget 3 threshold 1: unknown field "coler" (did you mean "color"?)`,
			`line 1 widget 3 threshold 1: missing "gte"`,
			`line 1 widget 4: field "color" is set twice`,
			`subagentLine widget 1: unknown field "bolt" (did you mean "bold"?)`,
		}, problemLines(errOut))
		assert.EqualError(t, err, "13 problem(s) found in "+config.Path())
	})

	t.Run("every documented key is known", func(t *testing.T) {
		writeSettings(t, `{
  "colorLevel": 2, "hyperlinks": true, "flexMode": "exact", "compactThreshold": 80, "terminalWidth": 120,
  "defaultSeparator": "|", "defaultPadding": " ", "inheritSeparatorColors": true,
  "overrideForegroundColor": "white", "overrideBackgroundColor": "black", "globalBold": false,
  "lines": [[
    {"type": "model", "color": "cyan", "backgroundColor": "black", "bold": true, "prefix": "", "suffix": " ", "rawValue": true, "maxWidth": 20, "merge": true},
    {"type": "separator", "character": "·"},
    {"type": "rate-limit-5h", "display": "bar", "barWidth": 20, "thresholds": [{"gte": 80, "color": "red", "backgroundColor": "white", "bold": true}]},
    {"type": "custom-text", "customText": "hi"},
    {"type": "custom-command", "commandPath": "date", "timeout": 500, "preserveColors": true}
  ]],
  "subagentLine": [{"type": "task-name"}],
  "subagentHideStatuses": ["completed"]
}`)
		out, errOut, err := execValidate(t)
		require.NoError(t, err, errOut)
		assert.Equal(t, "Settings are valid: "+config.Path()+"\n", out)
	})

	t.Run("keys match as the decoder matches them, ignoring case", func(t *testing.T) {
		writeSettings(t, `{"Lines": [[{"TYPE": "model", "Color": "cyan"}]]}`)
		_, errOut, err := execValidate(t)
		require.NoError(t, err, errOut)
	})
}

func TestEditDistance(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"color", "color", 0},
		{"coler", "color", 1},
		{"barwidht", "barwidth", 2},
		{"gt", "gte", 1},
		{"", "abc", 3},
		{"id", "gte", 3},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, editDistance(tt.a, tt.b), "%q -> %q", tt.a, tt.b)
	}
}

func TestNearest(t *testing.T) {
	keys := []string{"type", "color", "backgroundColor", "bold", "thresholds", "barWidth", "gte"}
	assert.Equal(t, "color", nearest("coler", keys))
	assert.Equal(t, "color", nearest("Colr", keys))
	assert.Equal(t, "thresholds", nearest("tresholds", keys))
	assert.Equal(t, "gte", nearest("gt", keys))
	assert.Empty(t, nearest("id", keys), "a short word is not stretched into another")
	assert.Empty(t, nearest("metadata", keys))
}
