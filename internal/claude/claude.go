// Package claude provides integration with Claude Code's settings.json.
package claude

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	settingsFileName      = "settings.json"
	statusLineKey         = "statusLine"
	subagentStatusLineKey = "subagentStatusLine"
	// Command is the name Claude Code runs ccstatus by, resolved through PATH.
	Command          = "ccstatus"
	subagentArgument = "subagent"

	// DefaultRefreshInterval is the refreshInterval a fresh install writes, so
	// time-based widgets such as session-clock keep moving while Claude Code is
	// idle.
	DefaultRefreshInterval = 30
)

// StatusLine is the configuration block written to Claude Code's settings.json.
type StatusLine struct {
	Type                 string `json:"type"`
	Command              string `json:"command"`
	Padding              int    `json:"padding"`
	RefreshInterval      *int   `json:"refreshInterval,omitempty"`
	HideVimModeIndicator *bool  `json:"hideVimModeIndicator,omitempty"`
}

// SubagentStatusLine is the subagentStatusLine block written to Claude Code's
// settings.json.
type SubagentStatusLine struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// ErrNoConfigDir reports that neither $CLAUDE_CONFIG_DIR nor the home directory
// is known, so Claude Code's settings.json cannot be located.
var ErrNoConfigDir = errors.New("cannot locate Claude Code's config directory: set $HOME or $CLAUDE_CONFIG_DIR")

// Dir returns the Claude Code configuration directory, or "" when it cannot be
// located. It respects the CLAUDE_CONFIG_DIR environment variable, falling back
// to ~/.claude.
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

// SettingsPath returns the full path to Claude Code's settings.json, or "" when
// its directory cannot be located.
func SettingsPath() string {
	dir := Dir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, settingsFileName)
}

// InstallOptions configures the blocks written by Install. A nil field leaves
// that setting at its existing value (or its default on a fresh install) rather
// than overwriting it, so re-installing never silently drops a setting.
type InstallOptions struct {
	RefreshInterval *int  // > 0 sets refreshInterval; <= 0 clears it
	Padding         *int  // sets padding
	HideVimMode     *bool // true sets hideVimModeIndicator; false clears it
	Subagent        *bool // true writes subagentStatusLine; false removes ccstatus's
}

// InstallResult reports what Install wrote.
type InstallResult struct {
	Path    string // the settings.json written
	Command string // the statusLine command: Command, or the absolute path that was kept
}

// Install registers ccstatus as the statusLine command in Claude Code's
// settings.json, and as the subagentStatusLine command when opts.Subagent is
// true. Every other key keeps its position and value. The command is the bare
// name, which Claude Code resolves through PATH, unless the block already runs
// ccstatus by absolute path: that path is kept, without its arguments. A fresh
// install, where no statusLine ran ccstatus before, also sets refreshInterval
// to DefaultRefreshInterval unless one is already set or opts overrides it. The
// file is created if it does not exist.
func Install(opts InstallOptions) (InstallResult, error) {
	res := InstallResult{Path: SettingsPath()}
	if res.Path == "" {
		return res, ErrNoConfigDir
	}
	doc, err := readDocument(res.Path)
	if err != nil {
		return res, err
	}

	raw, had := doc.get(statusLineKey)
	res.Command = binaryOf(raw)
	sl := statusLineFor(decodeStatusLine(raw), res.Command, opts, !had || !runsCCStatus(raw))
	if err := doc.set(statusLineKey, sl); err != nil {
		return res, err
	}
	if err := applySubagent(doc, opts.Subagent, res.Command); err != nil {
		return res, err
	}
	return res, writeDocument(res.Path, doc)
}

// binaryOf returns the way a statusLine-style block names the ccstatus binary
// when that is an absolute path, so a path set by hand survives a re-install,
// and the bare command name otherwise.
func binaryOf(raw json.RawMessage) string {
	var block struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(raw, &block) == nil {
		if fields := strings.Fields(block.Command); len(fields) > 0 &&
			filepath.IsAbs(fields[0]) && filepath.Base(fields[0]) == Command {
			return fields[0]
		}
	}
	return Command
}

// statusLineFor points sl at the ccstatus binary and applies opts; fresh marks
// a statusLine that did not run ccstatus before.
func statusLineFor(sl StatusLine, binary string, opts InstallOptions, fresh bool) StatusLine {
	sl.Type = "command"
	sl.Command = binary
	if opts.Padding != nil {
		sl.Padding = *opts.Padding
	}
	switch {
	case opts.RefreshInterval != nil && *opts.RefreshInterval > 0:
		v := *opts.RefreshInterval
		sl.RefreshInterval = &v
	case opts.RefreshInterval != nil:
		sl.RefreshInterval = nil
	case fresh && sl.RefreshInterval == nil:
		v := DefaultRefreshInterval
		sl.RefreshInterval = &v
	}
	if opts.HideVimMode != nil {
		if *opts.HideVimMode {
			enabled := true
			sl.HideVimModeIndicator = &enabled
		} else {
			sl.HideVimModeIndicator = nil
		}
	}
	return sl
}

// applySubagent writes the subagentStatusLine block when enable is true and
// removes ccstatus's block when it is false; nil leaves the key alone. The
// block runs the binary it already names by absolute path, else the one the
// statusLine block runs.
func applySubagent(doc *document, enable *bool, binary string) error {
	switch {
	case enable == nil:
		return nil
	case *enable:
		if raw, ok := doc.get(subagentStatusLineKey); ok && binaryOf(raw) != Command {
			binary = binaryOf(raw)
		}
		return doc.set(subagentStatusLineKey, SubagentStatusLine{Type: "command", Command: binary + " " + subagentArgument})
	default:
		if raw, ok := doc.get(subagentStatusLineKey); ok && runsCCStatus(raw) {
			doc.remove(subagentStatusLineKey)
		}
		return nil
	}
}

// decodeStatusLine reads an existing statusLine block so fields Install does
// not override survive a re-install. Returns the zero StatusLine when absent or
// unparsable.
func decodeStatusLine(raw json.RawMessage) StatusLine {
	var sl StatusLine
	if len(raw) == 0 || json.Unmarshal(raw, &sl) != nil {
		return StatusLine{}
	}
	return sl
}

// UninstallResult reports what Uninstall changed.
type UninstallResult struct {
	Path    string
	Removed []string // keys removed because their command runs ccstatus
	Kept    []string // keys left alone because their command runs something else
}

// Uninstall removes the statusLine and subagentStatusLine blocks whose command
// runs ccstatus from Claude Code's settings.json, leaving blocks that run another
// command and every other key untouched. It is a no-op if the file does not
// exist or nothing runs ccstatus.
func Uninstall() (UninstallResult, error) {
	res := UninstallResult{Path: SettingsPath()}
	if res.Path == "" {
		return res, ErrNoConfigDir
	}
	if _, err := os.Stat(res.Path); os.IsNotExist(err) {
		return res, nil
	}
	doc, err := readDocument(res.Path)
	if err != nil {
		return res, err
	}
	for _, key := range []string{statusLineKey, subagentStatusLineKey} {
		raw, ok := doc.get(key)
		switch {
		case !ok:
		case runsCCStatus(raw):
			doc.remove(key)
			res.Removed = append(res.Removed, key)
		default:
			res.Kept = append(res.Kept, key)
		}
	}
	if len(res.Removed) == 0 {
		return res, nil
	}
	return res, writeDocument(res.Path, doc)
}

// runsCCStatus reports whether a statusLine-style block's command runs the
// ccstatus binary (by name or path, with any arguments).
func runsCCStatus(raw json.RawMessage) bool {
	var block struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(raw, &block) != nil {
		return false
	}
	fields := strings.Fields(block.Command)
	return len(fields) > 0 && filepath.Base(fields[0]) == Command
}
