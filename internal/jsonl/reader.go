// Package jsonl provides JSONL transcript file parsing for the block-timer widget.
package jsonl

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// maxLineBytes bounds the first transcript line the scanner will read. The
// default bufio.Scanner cap is 64KB, which a large first record (pasted/tool/
// image payload) can exceed, silently yielding no timestamp.
const maxLineBytes = 1 << 20 // 1 MiB

// entry represents the minimal fields we need from a JSONL transcript entry.
type entry struct {
	Timestamp string `json:"timestamp"`
}

// SessionStart reads the first entry from a JSONL transcript file and returns
// its timestamp. Returns zero time if the file cannot be read or parsed.
func SessionStart(path string) time.Time {
	if path == "" {
		return time.Time{}
	}
	f, err := os.Open(filepath.Clean(path)) //nolint:gosec // path comes from Claude Code JSON, not untrusted input
	if err != nil {
		return time.Time{}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	if !scanner.Scan() {
		return time.Time{}
	}
	line := scanner.Bytes()

	var e entry
	if unmarshalErr := json.Unmarshal(line, &e); unmarshalErr != nil {
		return time.Time{}
	}
	if e.Timestamp == "" {
		return time.Time{}
	}

	t, parseErr := time.Parse(time.RFC3339Nano, e.Timestamp)
	if parseErr != nil {
		// Try RFC3339 without nanoseconds.
		t, parseErr = time.Parse(time.RFC3339, e.Timestamp)
		if parseErr != nil {
			return time.Time{}
		}
	}
	return t
}
