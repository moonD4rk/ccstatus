package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// CurrentVersion is the current settings schema version.
	CurrentVersion = 4

	configDirName  = "ccstatus"
	configFileName = "settings.json"
)

// Settings represents the ccstatus configuration.
type Settings struct {
	Version                 int            `json:"version"`
	Lines                   [][]WidgetItem `json:"lines"`
	FlexMode                string         `json:"flexMode"`
	CompactThreshold        int            `json:"compactThreshold"`
	ColorLevel              int            `json:"colorLevel"`
	TerminalWidth           int            `json:"terminalWidth,omitempty"`
	DefaultSeparator        string         `json:"defaultSeparator,omitempty"`
	DefaultPadding          string         `json:"defaultPadding,omitempty"`
	InheritSeparatorColors  bool           `json:"inheritSeparatorColors"`
	OverrideBackgroundColor string         `json:"overrideBackgroundColor,omitempty"`
	OverrideForegroundColor string         `json:"overrideForegroundColor,omitempty"`
	GlobalBold              bool           `json:"globalBold"`
	// SubagentLine is the row `ccstatus subagent` renders for each subagent in
	// Claude Code's agent panel.
	SubagentLine []WidgetItem `json:"subagentLine,omitempty"`
}

// DefaultSettings returns the default ccstatus configuration.
func DefaultSettings() Settings {
	return Settings{
		Version:          CurrentVersion,
		ColorLevel:       2,
		FlexMode:         "full-until-compact",
		CompactThreshold: 90,
		DefaultSeparator: "|",
		DefaultPadding:   " ",
		Lines: [][]WidgetItem{
			{
				// model · effort | context % | git branch worktree | +added -removed | $cost
				{ID: "1", Type: "model", Color: "cyan"},
				{ID: "2", Type: "effort", Color: "cyan", Prefix: "· "},
				{ID: "3", Type: "separator"},
				{ID: "4", Type: "context-percentage", Color: "green", Thresholds: []Threshold{
					{GTE: 70, Color: "yellow"},
					{GTE: 90, Color: "red"},
				}},
				{ID: "5", Type: "separator"},
				{ID: "6", Type: "git-branch", Color: "magenta"},
				{ID: "7", Type: "git-worktree", Color: "brightBlack"},
				{ID: "8", Type: "separator"},
				{ID: "9", Type: "lines-added", Color: "green"},
				{ID: "10", Type: "lines-removed", Color: "red"},
				{ID: "11", Type: "separator"},
				{ID: "12", Type: "session-cost", Color: "green"},
			},
			{
				// current dir | rate limits (5h/7d) | session clock  (plain separator, left-aligned)
				{ID: "13", Type: "current-working-dir", Color: "blue", RawValue: true},
				{ID: "14", Type: "separator"},
				{ID: "15", Type: "rate-limits", Color: "yellow"},
				{ID: "16", Type: "separator"},
				{ID: "17", Type: "session-clock", Color: "white"},
			},
		},
		// name status elapsed | model tokens context% sparkline | label
		SubagentLine: []WidgetItem{
			{ID: "s1", Type: "task-name", Color: "cyan"},
			{ID: "s2", Type: "task-status"},
			{ID: "s3", Type: "task-elapsed"},
			{ID: "s4", Type: "separator"},
			{ID: "s5", Type: "task-model"},
			{ID: "s6", Type: "task-tokens"},
			{ID: "s7", Type: "task-context", Thresholds: []Threshold{
				{GTE: 70, Color: "yellow"},
				{GTE: 90, Color: "red"},
			}},
			{ID: "s8", Type: "task-sparkline"},
			{ID: "s9", Type: "separator"},
			{ID: "s10", Type: "task-label"},
		},
	}
}

// Dir returns the ccstatus configuration directory path.
func Dir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, configDirName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", configDirName)
}

// Path returns the full path to the settings.json file.
func Path() string {
	return filepath.Join(Dir(), configFileName)
}

// Load reads the settings file over the defaults, so fields it omits keep their
// default values; a missing file yields the defaults. On a read or decode error
// it returns the defaults along with the error, so callers can still render.
func Load() (Settings, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultSettings(), nil
		}
		return DefaultSettings(), err
	}
	s, err := Parse(data)
	if err != nil {
		return DefaultSettings(), err
	}
	return s, nil
}

// Parse decodes settings JSON over the defaults. An absent or null "lines" keeps
// the default layout, while an explicit empty list renders nothing; the same
// holds for "subagentLine". Decode errors are returned as a *PositionError.
func Parse(data []byte) (Settings, error) {
	s := DefaultSettings()
	s.Lines = nil
	s.SubagentLine = nil
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, positionError(data, err)
	}
	def := DefaultSettings()
	if s.Lines == nil {
		s.Lines = def.Lines
	}
	if s.SubagentLine == nil {
		s.SubagentLine = def.SubagentLine
	}
	return s, nil
}

// PositionError is a settings decode error with the 1-based line and column at
// which it was detected.
type PositionError struct {
	Line   int
	Column int
	Err    error
}

// Error formats the error as "line:column: message".
func (e *PositionError) Error() string {
	return fmt.Sprintf("%d:%d: %v", e.Line, e.Column, e.Err)
}

// Unwrap returns the underlying decode error.
func (e *PositionError) Unwrap() error { return e.Err }

// positionError attaches a line and column to JSON syntax and type errors.
func positionError(data []byte, err error) error {
	var offset int64
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &syntaxErr):
		offset = syntaxErr.Offset
	case errors.As(err, &typeErr):
		offset = typeErr.Offset
	default:
		return err
	}
	prefix := data[:min(max(offset, 0), int64(len(data)))]
	return &PositionError{
		Line:   bytes.Count(prefix, []byte("\n")) + 1,
		Column: max(len(prefix)-bytes.LastIndexByte(prefix, '\n')-1, 1),
		Err:    err,
	}
}

// Save writes the settings to the configuration file.
func Save(s *Settings) error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(Path(), data, 0o600)
}
