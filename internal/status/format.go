package status

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	defaultMaxTokens = 200_000
	// defaultUsableRatio is the share of the window treated as comfortably
	// usable before auto-compact pressure sets in near the limit. It is a
	// ccstatus heuristic for the opt-in context-percentage-usable widget, not a
	// value from the official spec (which exposes only the full-window
	// used_percentage and the fixed-200k exceeds_200k_tokens).
	defaultUsableRatio = 80
	longMaxTokens      = 1_000_000
)

// WindowLimits holds resolved context window size information.
type WindowLimits struct {
	MaxTokens    int
	UsableTokens int
}

// FormatTokens formats a token count into a human-readable string.
// Examples: 500 -> "500", 1500 -> "1.5k", 1200000 -> "1.2M".
// Rounding is applied before the unit is chosen, so values just under a boundary
// (e.g. 999_950) read as "1.0M" rather than a stale "1000.0k".
func FormatTokens(count int) string {
	switch {
	case count >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(count)/1_000_000)
	case count >= 1000:
		// Round to the 0.1k that %.1f will display; if it reaches 1000.0k, promote to M.
		if math.Round(float64(count)/100)/10 >= 1000 {
			return fmt.Sprintf("%.1fM", float64(count)/1_000_000)
		}
		return fmt.Sprintf("%.1fk", float64(count)/1000)
	default:
		return strconv.Itoa(count)
	}
}

// ContextConfig resolves context window size.
// Primary: use context_window.context_window_size from JSON input.
// Fallback: heuristic based on model ID (for older Claude Code versions).
func ContextConfig(data *Session) WindowLimits {
	if data.ContextWindow != nil && data.ContextWindow.ContextWindowSize != nil {
		size := *data.ContextWindow.ContextWindowSize
		return WindowLimits{
			MaxTokens:    size,
			UsableTokens: size * defaultUsableRatio / 100,
		}
	}

	// Fallback for older Claude Code that omits context_window_size: the "[1m]"
	// model-id suffix marks an extended (1M) window for any model family, e.g.
	// claude-sonnet-4-5[1m] or claude-opus-4-8[1m].
	if strings.Contains(strings.ToLower(data.Model.ID), "[1m]") {
		return WindowLimits{
			MaxTokens:    longMaxTokens,
			UsableTokens: longMaxTokens * defaultUsableRatio / 100,
		}
	}
	return WindowLimits{
		MaxTokens:    defaultMaxTokens,
		UsableTokens: defaultMaxTokens * defaultUsableRatio / 100,
	}
}

// ContextPercentage returns the context usage percentage.
// Primary: use pre-calculated used_percentage from JSON input.
// Fallback: calculate from current_usage tokens and context_window_size.
// Returns (value, ok) where ok=false means no data available.
func ContextPercentage(data *Session) (float64, bool) {
	if data.ContextWindow != nil && data.ContextWindow.UsedPercentage != nil {
		return *data.ContextWindow.UsedPercentage, true
	}
	if data.ContextWindow != nil && data.ContextWindow.CurrentUsage != nil {
		cu := data.ContextWindow.CurrentUsage
		contextLength := cu.InputTokens + cu.CacheCreationInputTokens + cu.CacheReadInputTokens
		cfg := ContextConfig(data)
		if cfg.MaxTokens == 0 {
			return 0, false
		}
		pct := float64(contextLength) / float64(cfg.MaxTokens) * 100
		if pct > 100 {
			return 100, true
		}
		return pct, true
	}
	return 0, false
}

// RemainingPercentage returns the remaining context window percentage.
// Primary: use pre-calculated remaining_percentage from JSON input.
// Fallback: calculate as 100 - used_percentage.
// Returns (value, ok) where ok=false means no data available.
func RemainingPercentage(data *Session) (float64, bool) {
	if data.ContextWindow != nil && data.ContextWindow.RemainingPercentage != nil {
		return *data.ContextWindow.RemainingPercentage, true
	}
	used, ok := ContextPercentage(data)
	if !ok {
		return 0, false
	}
	remaining := 100 - used
	if remaining < 0 {
		return 0, true
	}
	return remaining, true
}

// CacheHitRate returns the cache read ratio as a percentage.
// Formula: cache_read_input_tokens / (input_tokens + cache_creation_input_tokens + cache_read_input_tokens) * 100
// It needs the per-component current_usage breakdown, so it returns ok=false
// when current_usage is absent (e.g. just after /compact) or the total is 0.
func CacheHitRate(data *Session) (float64, bool) {
	if data.ContextWindow == nil || data.ContextWindow.CurrentUsage == nil {
		return 0, false
	}
	cu := data.ContextWindow.CurrentUsage
	total := cu.InputTokens + cu.CacheCreationInputTokens + cu.CacheReadInputTokens
	if total == 0 {
		return 0, false
	}
	return float64(cu.CacheReadInputTokens) / float64(total) * 100, true
}

// ContextLength returns the total input token count (context length): the sum of
// input_tokens + cache_creation_input_tokens + cache_read_input_tokens.
// When current_usage is null (e.g. just after /compact, until the next API call)
// it falls back to total_input_tokens, which the spec defines as the same sum,
// so the value does not blank out mid-session.
func ContextLength(data *Session) int {
	if data.ContextWindow == nil {
		return 0
	}
	if cu := data.ContextWindow.CurrentUsage; cu != nil {
		return cu.InputTokens + cu.CacheCreationInputTokens + cu.CacheReadInputTokens
	}
	if data.ContextWindow.TotalInputTokens != nil {
		return *data.ContextWindow.TotalInputTokens
	}
	return 0
}
