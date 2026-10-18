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
				// model · effort | Fast | context % | git branch worktree | +added -removed | $cost
				{ID: "1", Type: "model", Color: "cyan"},
				{ID: "2", Type: "effort", Color: "cyan", Prefix: "· "},
				{ID: "3", Type: "separator"},
				{ID: "4", Type: "fast-mode"},
				{ID: "5", Type: "separator"},
				{ID: "6", Type: "context-percentage", Color: "brightBlack", Thresholds: usageThresholds("white")},
				{ID: "7", Type: "separator"},
				{ID: "8", Type: "git-branch", Color: "magenta"},
				{ID: "9", Type: "git-worktree", Color: "brightBlack"},
				{ID: "10", Type: "separator"},
				{ID: "11", Type: "lines-added", Color: "green"},
				{ID: "12", Type: "lines-removed", Color: "red"},
				{ID: "13", Type: "separator"},
				{ID: "14", Type: "session-cost", Color: "green"},
			},
			{
				// current dir | rate limits (5h/7d) | session clock | session name  (plain separator, left-aligned)
				{ID: "15", Type: "current-working-dir", Color: "blue", RawValue: true},
				{ID: "16", Type: "separator"},
				{ID: "17", Type: "rate-limits", Color: "white", Thresholds: usageThresholds("brightWhite")},
				{ID: "18", Type: "separator"},
				{ID: "19", Type: "session-clock", Color: "white"},
				{ID: "20", Type: "separator"},
				{ID: "21", Type: "session-name", MaxWidth: 30},
			},
		},
		// name status elapsed | model · effort tokens context% sparkline | label
		SubagentLine: []WidgetItem{
			{ID: "s1", Type: "task-name", Color: "cyan"},
			{ID: "s2", Type: "task-status"},
			{ID: "s3", Type: "task-elapsed"},
			{ID: "s4", Type: "separator"},
			{ID: "s5", Type: "task-model"},
			{ID: "s6", Type: "task-effort", Prefix: "· "},
			{ID: "s7", Type: "task-tokens"},
			{ID: "s8", Type: "task-context", Color: "brightBlack", Thresholds: usageThresholds("white")},
			{ID: "s9", Type: "task-sparkline"},
			{ID: "s10", Type: "separator"},
			{ID: "s11", Type: "task-label"},
		},
	}
}

// usageThresholds marks 30% with hint, a step brighter than the base color, then escalates
// to yellow at 60% and red at 90%.
func usageThresholds(hint string) []Threshold {
	return []Threshold{
		{GTE: 30, Color: hint},
		{GTE: 60, Color: "yellow"},
		{GTE: 90, Color: "red"},
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
