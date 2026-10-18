// Package sample embeds sample Claude Code payloads for `ccstatus preview` and
// for tests.
package sample

import (
	_ "embed"
	"encoding/json"
	"time"
)

// StatusLineDocs is the full status line payload from the official docs, as
// published.
//
//go:embed statusline.json
var StatusLineDocs []byte

// SubagentDocs is a subagentStatusLine payload that exercises every task field.
//
//go:embed subagent.json
var SubagentDocs []byte

// StatusLine returns the docs payload with its timestamps moved relative to now,
// so countdowns render as they would in a live session.
func StatusLine(now time.Time) []byte {
	return shift(StatusLineDocs, func(doc map[string]any) {
		set(doc, now.Add(42*time.Minute+30*time.Second).Unix(), "prompt_cache", "expires_at")
		set(doc, now.Add(-10*time.Minute).Unix(), "prompt_cache", "last_miss_at")
		set(doc, now.Add(2*time.Hour+13*time.Minute).Unix(), "rate_limits", "five_hour", "resets_at")
		set(doc, now.Add(4*24*time.Hour+15*time.Hour).Unix(), "rate_limits", "seven_day", "resets_at")
		set(doc, now.Add(12*24*time.Hour+4*time.Hour).Unix(), "rate_limits", "spend_limit", "resets_at")
	})
}

// Subagent returns the subagent payload with task start times moved relative to
// now.
func Subagent(now time.Time) []byte {
	ages := []time.Duration{83 * time.Second, 4*time.Minute + 10*time.Second}
	return shift(SubagentDocs, func(doc map[string]any) {
		tasks, _ := doc["tasks"].([]any)
		for i, t := range tasks {
			if task, ok := t.(map[string]any); ok && i < len(ages) {
				task["startTime"] = now.Add(-ages[i]).UnixMilli()
			}
		}
	})
}

// shift applies edit to a decoded copy of payload, returning payload unchanged
// if it does not round-trip.
func shift(payload []byte, edit func(map[string]any)) []byte {
	var doc map[string]any
	if json.Unmarshal(payload, &doc) != nil {
		return payload
	}
	edit(doc)
	out, err := json.Marshal(doc)
	if err != nil {
		return payload
	}
	return out
}

// set assigns value at the nested key path when every parent object exists.
func set(doc map[string]any, value any, path ...string) {
	m := doc
	for _, key := range path[:len(path)-1] {
		next, ok := m[key].(map[string]any)
		if !ok {
			return
		}
		m = next
	}
	m[path[len(path)-1]] = value
}
