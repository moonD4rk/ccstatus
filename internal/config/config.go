package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	// CurrentVersion is the current settings schema version.
	CurrentVersion = 3

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
	DefaultSeparator        string         `json:"defaultSeparator,omitempty"`
	DefaultPadding          string         `json:"defaultPadding,omitempty"`
	InheritSeparatorColors  bool           `json:"inheritSeparatorColors"`
	OverrideBackgroundColor string         `json:"overrideBackgroundColor,omitempty"`
	OverrideForegroundColor string         `json:"overrideForegroundColor,omitempty"`
	GlobalBold              bool           `json:"globalBold"`
}

// DefaultSettings returns the default ccstatus configuration.
func DefaultSettings() Settings {
	return Settings{
		Version:          CurrentVersion,
		ColorLevel:       2,
		FlexMode:         "full-minus-40",
		CompactThreshold: 60,
		DefaultSeparator: "|",
		DefaultPadding:   " ",
		Lines: [][]WidgetItem{
			{
				{ID: "1", Type: "model", Color: "cyan"},
				{ID: "2", Type: "separator"},
				{ID: "3", Type: "context-length", Color: "brightBlack"},
				{ID: "4", Type: "separator"},
				{ID: "5", Type: "git-branch", Color: "magenta"},
				{ID: "6", Type: "separator"},
				{ID: "7", Type: "git-changes", Color: "yellow"},
			},
		},
	}
}

// ConfigDir returns the ccstatus configuration directory path.
func ConfigDir() string {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, configDirName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", configDirName)
}

// ConfigPath returns the full path to the settings.json file.
func ConfigPath() string {
	return filepath.Join(ConfigDir(), configFileName)
}

// Load reads and parses the settings file. Returns defaults if the file does not exist.
func Load() (Settings, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultSettings(), nil
		}
		return Settings{}, err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, err
	}
	return s, nil
}

// Save writes the settings to the configuration file.
func Save(s *Settings) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(ConfigPath(), data, 0o600)
}
