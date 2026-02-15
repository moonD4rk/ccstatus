package widget

import (
	"fmt"
	"strings"
	"time"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/jsonl"
)

const (
	blockDuration = 5 * time.Hour
	barWidth      = 10
)

// BlockTimerWidget displays elapsed time within a 5-hour Claude Code session block.
// Supports three display modes via metadata["display"]:
//   - "time" (default): shows elapsed/total like "2h15m/5h"
//   - "progress": shows progress bar "[=====>    ] 45%"
//   - "percentage": shows just "45%"
type BlockTimerWidget struct{}

// Render returns the block timer display.
// Uses cost.total_duration_ms when available, falls back to JSONL transcript parsing.
func (w *BlockTimerWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	elapsed := w.getElapsed(ctx)
	if elapsed <= 0 {
		return ""
	}

	// Clamp to block duration.
	if elapsed > blockDuration {
		elapsed = blockDuration
	}

	mode := "time"
	if item.Metadata != nil {
		if m, ok := item.Metadata["display"]; ok {
			mode = m
		}
	}

	pct := float64(elapsed) / float64(blockDuration) * 100

	switch mode {
	case "progress":
		return formatProgressBar(pct)
	case "percentage":
		return fmt.Sprintf("%.0f%%", pct)
	default:
		return formatBlockTime(elapsed)
	}
}

// getElapsed determines session elapsed time.
// Prefers cost.total_duration_ms; falls back to JSONL transcript timestamp.
func (w *BlockTimerWidget) getElapsed(ctx RenderContext) time.Duration {
	if ctx.Data == nil {
		return 0
	}

	// Primary: use cost.total_duration_ms if available.
	if ctx.Data.Cost != nil && ctx.Data.Cost.TotalDurationMS != nil {
		ms := *ctx.Data.Cost.TotalDurationMS
		if ms > 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}

	// Fallback: parse JSONL transcript for session start time.
	if ctx.Data.TranscriptPath != "" {
		start := jsonl.SessionStart(ctx.Data.TranscriptPath)
		if !start.IsZero() {
			return time.Since(start)
		}
	}

	return 0
}

func formatBlockTime(elapsed time.Duration) string {
	hours := int(elapsed.Hours())
	mins := int(elapsed.Minutes()) % 60
	if hours == 0 && mins == 0 {
		return "<1m/5h"
	}
	if hours == 0 {
		return fmt.Sprintf("%dm/5h", mins)
	}
	if mins == 0 {
		return fmt.Sprintf("%dh/5h", hours)
	}
	return fmt.Sprintf("%dh%dm/5h", hours, mins)
}

func formatProgressBar(pct float64) string {
	filled := min(int(pct*barWidth/100), barWidth)
	empty := barWidth - filled
	var b strings.Builder
	b.WriteByte('[')
	for range filled {
		b.WriteByte('=')
	}
	if empty > 0 {
		b.WriteByte('>')
		for range empty - 1 {
			b.WriteByte(' ')
		}
	}
	b.WriteByte(']')
	fmt.Fprintf(&b, " %.0f%%", pct)
	return b.String()
}

// DefaultColor returns the default foreground color.
func (w *BlockTimerWidget) DefaultColor() string { return defaultDimColor }

// DisplayName returns the human-readable name.
func (w *BlockTimerWidget) DisplayName() string { return "Block Timer" }

// Description returns what this widget shows.
func (w *BlockTimerWidget) Description() string { return "5-hour session block timer" }

// SupportsRawValue returns false since display mode is controlled by metadata.
func (w *BlockTimerWidget) SupportsRawValue() bool { return false }
