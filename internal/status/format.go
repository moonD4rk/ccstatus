package status

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	defaultMaxTokens   = 200_000
	defaultUsableRatio = 80 // 80% of max
	longMaxTokens      = 1_000_000
)

// ContextConfig holds resolved context window size information.
type ContextConfig struct {
	MaxTokens    int
	UsableTokens int
}

// FormatTokens formats a token count into a human-readable string.
// Examples: 500 -> "500", 1500 -> "1.5k", 1200000 -> "1.2M"
func FormatTokens(count int) string {
	if count >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(count)/1_000_000)
	}
	if count >= 1000 {
		return fmt.Sprintf("%.1fk", float64(count)/1000)
	}
	return strconv.Itoa(count)
}

// GetContextConfig resolves context window size.
// Primary: use context_window.context_window_size from JSON input.
// Fallback: heuristic based on model ID (for older Claude Code versions).
func GetContextConfig(data *StatusJSON) ContextConfig {
	if data.ContextWindow != nil && data.ContextWindow.ContextWindowSize != nil {
		size := *data.ContextWindow.ContextWindowSize
		return ContextConfig{
			MaxTokens:    size,
			UsableTokens: size * defaultUsableRatio / 100,
		}
	}

	lower := strings.ToLower(data.Model.ID)
	if strings.Contains(lower, "claude-sonnet-4-5") && strings.Contains(lower, "[1m]") {
		return ContextConfig{
			MaxTokens:    longMaxTokens,
			UsableTokens: longMaxTokens * defaultUsableRatio / 100,
		}
	}
	return ContextConfig{
		MaxTokens:    defaultMaxTokens,
		UsableTokens: defaultMaxTokens * defaultUsableRatio / 100,
	}
}

// GetContextPercentage returns the context usage percentage.
// Primary: use pre-calculated used_percentage from JSON input.
// Fallback: calculate from current_usage tokens and context_window_size.
func GetContextPercentage(data *StatusJSON) float64 {
	if data.ContextWindow != nil && data.ContextWindow.UsedPercentage != nil {
		return *data.ContextWindow.UsedPercentage
	}
	if data.ContextWindow != nil && data.ContextWindow.CurrentUsage != nil {
		cu := data.ContextWindow.CurrentUsage
		contextLength := cu.InputTokens + cu.CacheCreationInputTokens + cu.CacheReadInputTokens
		cfg := GetContextConfig(data)
		if cfg.MaxTokens == 0 {
			return 0
		}
		pct := float64(contextLength) / float64(cfg.MaxTokens) * 100
		if pct > 100 {
			return 100
		}
		return pct
	}
	return 0
}

// GetRemainingPercentage returns the remaining context window percentage.
// Primary: use pre-calculated remaining_percentage from JSON input.
// Fallback: calculate as 100 - used_percentage.
func GetRemainingPercentage(data *StatusJSON) float64 {
	if data.ContextWindow != nil && data.ContextWindow.RemainingPercentage != nil {
		return *data.ContextWindow.RemainingPercentage
	}
	used := GetContextPercentage(data)
	if used == 0 {
		return 0
	}
	remaining := 100 - used
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetContextLength returns the total input token count (context length).
// This is the sum of input_tokens + cache_creation_input_tokens + cache_read_input_tokens.
func GetContextLength(data *StatusJSON) int {
	if data.ContextWindow == nil || data.ContextWindow.CurrentUsage == nil {
		return 0
	}
	cu := data.ContextWindow.CurrentUsage
	return cu.InputTokens + cu.CacheCreationInputTokens + cu.CacheReadInputTokens
}
