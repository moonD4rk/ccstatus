package widget

import (
	"fmt"
	"strings"
	"time"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

// PromptCacheWidget displays the main conversation's prompt cache state.
// Display modes:
//   - "status" (default): "warm 42m" (time until the cache goes cold) or "cold"
//   - "ttl": just the countdown, or "cold"
//   - "hit": session hit ratio, e.g. "91%"
//   - "misses": e.g. "2 misses (tools_changed)"
//   - "full": "warm 42m · 91% hit · 2 misses (tools_changed)"
//
// Hidden when prompt_cache is absent or no response has reported cache tokens.
// The countdown advances only with a refreshInterval; Claude Code re-runs the
// status line itself when the cache expires, so the flip to "cold" is immediate.
type PromptCacheWidget struct{}

// Render returns the prompt cache display for the configured mode.
func (w *PromptCacheWidget) Render(item *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	pc := promptCache(ctx)
	if pc == nil {
		return ""
	}
	now := ctx.now()
	switch displayMode(item, "status") {
	case "ttl":
		return cacheTTL(pc, now)
	case "hit":
		return cacheHit(pc, item.RawValue)
	case "misses":
		return cacheMisses(pc)
	case "full":
		parts := []string{cacheStatus(pc, now)}
		if hit := cacheHit(pc, false); hit != "" {
			parts = append(parts, hit+" hit")
		}
		if pc.Misses != nil && *pc.Misses > 0 {
			parts = append(parts, cacheMisses(pc))
		}
		return strings.Join(parts, " · ")
	default:
		return cacheStatus(pc, now)
	}
}

// Value returns the hit percentage in "hit" mode and the minutes until the cache
// goes cold otherwise, for threshold colors.
func (w *PromptCacheWidget) Value(item *config.WidgetItem, ctx RenderContext) (float64, bool) {
	pc := promptCache(ctx)
	if pc == nil {
		return 0, false
	}
	if displayMode(item, "status") == "hit" {
		if pc.HitRatio == nil {
			return 0, false
		}
		return *pc.HitRatio * 100, true
	}
	left, _ := cacheTimeLeft(pc, ctx.now())
	return left.Minutes(), true
}

// DefaultColorFor returns green while the cache is warm and brightBlack once cold.
func (w *PromptCacheWidget) DefaultColorFor(ctx RenderContext) string {
	pc := promptCache(ctx)
	if pc == nil {
		return ""
	}
	if _, warm := cacheTimeLeft(pc, ctx.now()); warm {
		return defaultGreenColor
	}
	return "brightBlack"
}

func promptCache(ctx RenderContext) *status.PromptCache {
	if ctx.Data == nil || ctx.Data.PromptCache == nil {
		return nil
	}
	pc := ctx.Data.PromptCache
	if pc.CachingObserved != nil && !*pc.CachingObserved {
		return nil
	}
	return pc
}

// cacheTimeLeft returns the time from now until the cached prefix goes cold
// (zero when the expiry is unknown) and whether it is still warm.
func cacheTimeLeft(pc *status.PromptCache, now time.Time) (time.Duration, bool) {
	if pc.Warm == nil || !*pc.Warm {
		return 0, false
	}
	if pc.ExpiresAt == nil {
		return 0, true
	}
	left := time.Unix(*pc.ExpiresAt, 0).Sub(now)
	if left <= 0 {
		return 0, false
	}
	return left, true
}

func cacheStatus(pc *status.PromptCache, now time.Time) string {
	left, warm := cacheTimeLeft(pc, now)
	switch {
	case !warm:
		return "cold"
	case left > 0:
		return "warm " + formatDuration(float64(left.Milliseconds()))
	default:
		return "warm"
	}
}

func cacheTTL(pc *status.PromptCache, now time.Time) string {
	left, warm := cacheTimeLeft(pc, now)
	switch {
	case !warm:
		return "cold"
	case left > 0:
		return formatDuration(float64(left.Milliseconds()))
	default:
		return "warm"
	}
}

func cacheHit(pc *status.PromptCache, raw bool) string {
	if pc.HitRatio == nil {
		return ""
	}
	if raw {
		return fmt.Sprintf("%.1f", *pc.HitRatio*100)
	}
	return fmt.Sprintf("%.0f%%", *pc.HitRatio*100)
}

func cacheMisses(pc *status.PromptCache) string {
	if pc.Misses == nil {
		return ""
	}
	n := *pc.Misses
	word := "misses"
	if n == 1 {
		word = "miss"
	}
	out := fmt.Sprintf("%d %s", n, word)
	if n > 0 && pc.LastMissCause != nil && len(pc.LastMissCause.Causes) > 0 {
		out += " (" + strings.Join(pc.LastMissCause.Causes, ", ") + ")"
	}
	return out
}

func (w *PromptCacheWidget) displayModes() []string {
	return []string{"status", "ttl", "hit", "misses", "full"}
}

// DefaultColor returns the default foreground color.
func (w *PromptCacheWidget) DefaultColor() string { return defaultGreenColor }

// Description returns what this widget shows.
func (w *PromptCacheWidget) Description() string {
	return "Prompt cache warm/cold with time left, hit ratio, or misses"
}

// SupportsRawValue returns true; raw mode drops the % in "hit" mode.
func (w *PromptCacheWidget) SupportsRawValue() bool { return true }

// DefaultPrefix returns the default prefix.
func (w *PromptCacheWidget) DefaultPrefix() string { return "Cache: " }
