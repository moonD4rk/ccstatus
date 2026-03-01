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
	Type    string `json:"type"`
	Command string `json:"command"`
	Padding int    `json:"padding"`
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

// Install registers ccstatus in Claude Code's settings.json.
// It preserves all existing fields and creates the file if it does not exist.
// The path to the written settings file is returned.
func Install() (string, error) {
	path := SettingsPath()
	settings, err := readSettings(path)
	if err != nil {
		return "", err
	}

	settings["statusLine"] = StatusLine{
		Type:    "command",
		Command: "ccstatus",
		Padding: 0,
	}

	if err := writeSettings(path, settings); err != nil {
		return "", err
	}
	return path, nil
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
