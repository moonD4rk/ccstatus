// Package claude provides integration with Claude Code's settings.json.
package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const settingsFileName = "settings.json"

// StatusLine is the configuration block written to Claude Code's settings.json.
type StatusLine struct {
	Type                 string `json:"type"`
	Command              string `json:"command"`
	Padding              int    `json:"padding"`
	RefreshInterval      *int   `json:"refreshInterval,omitempty"`
	HideVimModeIndicator *bool  `json:"hideVimModeIndicator,omitempty"`
}

// Dir returns the Claude Code configuration directory.
// It respects the CLAUDE_CONFIG_DIR environment variable,
// falling back to ~/.claude.
func Dir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude")
}

// SettingsPath returns the full path to Claude Code's settings.json.
func SettingsPath() string {
	return filepath.Join(Dir(), settingsFileName)
}

// InstallOptions configures the statusLine block written by Install. A nil field
// leaves that setting at its existing value (or its default on a fresh install)
// rather than overwriting it, so re-installing never silently drops a setting.
type InstallOptions struct {
	RefreshInterval *int  // > 0 sets refreshInterval; <= 0 clears it
	Padding         *int  // sets padding
	HideVimMode     *bool // true sets hideVimModeIndicator; false clears it
}

// Install registers ccstatus as the statusLine command in Claude Code's
// settings.json. It preserves every other top-level field and every statusLine
// field not overridden by opts, creating the file if it does not exist. Returns
// the path to the written file.
func Install(opts InstallOptions) (string, error) {
	path := SettingsPath()
	settings, err := readSettings(path)
	if err != nil {
		return "", err
	}

	sl := existingStatusLine(settings)
	sl.Type = "command"
	sl.Command = "ccstatus"
	if opts.Padding != nil {
		sl.Padding = *opts.Padding
	}
	if opts.RefreshInterval != nil {
		if v := *opts.RefreshInterval; v > 0 {
			sl.RefreshInterval = &v
		} else {
			sl.RefreshInterval = nil
		}
	}
	if opts.HideVimMode != nil {
		if *opts.HideVimMode {
			enabled := true
			sl.HideVimModeIndicator = &enabled
		} else {
			sl.HideVimModeIndicator = nil
		}
	}
	settings["statusLine"] = sl

	if err := writeSettings(path, settings); err != nil {
		return "", err
	}
	return path, nil
}

// existingStatusLine extracts the current statusLine block so unspecified fields
// survive a re-install. Returns the zero StatusLine when absent or unparsable.
func existingStatusLine(settings map[string]any) StatusLine {
	raw, ok := settings["statusLine"]
	if !ok {
		return StatusLine{}
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return StatusLine{}
	}
	var sl StatusLine
	if err := json.Unmarshal(b, &sl); err != nil {
		return StatusLine{}
	}
	return sl
}

// Uninstall removes the ccstatus statusLine from Claude Code's settings.json.
// It is a no-op if the file does not exist or statusLine is not present.
// The path to the settings file is returned.
func Uninstall() (string, error) {
	path := SettingsPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return path, nil
		}
		return "", err
	}

	settings := make(map[string]any)
	if err := json.Unmarshal(data, &settings); err != nil {
		return "", err
	}

	if _, ok := settings["statusLine"]; !ok {
		return path, nil
	}

	delete(settings, "statusLine")

	if err := writeSettings(path, settings); err != nil {
		return "", err
	}
	return path, nil
}

// readSettings reads and parses the settings file. Returns an empty map if
// the file does not exist.
func readSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, err
	}

	settings := make(map[string]any)
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	return settings, nil
}

// writeSettings marshals the settings map to JSON and writes it to the given path.
func writeSettings(path string, settings map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}
