package widget

import (
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

const (
	// defaultDimColor is the default foreground color for secondary widgets.
	defaultDimColor = "white"
	// defaultGreenColor is the default foreground color for positive/cost widgets.
	defaultGreenColor = "green"
)

// tokenExtractor is a function that extracts a token count from status Data.
type tokenExtractor func(data *status.Session) (int, bool)

// tokenWidget is a generic widget that displays a formatted token count.
type tokenWidget struct {
	extract       tokenExtractor
	displayName   string
	description   string
	defaultPrefix string
	defaultSuffix string
}

func (w *tokenWidget) Render(_ *config.WidgetItem, ctx RenderContext, _ *config.Settings) string {
	if ctx.Data == nil {
		return ""
	}
	count, ok := w.extract(ctx.Data)
	if !ok || count == 0 {
		return ""
	}
	return status.FormatTokens(count)
}

func (w *tokenWidget) DefaultColor() string   { return defaultDimColor }
func (w *tokenWidget) DisplayName() string    { return w.displayName }
func (w *tokenWidget) Description() string    { return w.description }
func (w *tokenWidget) SupportsRawValue() bool { return false }
func (w *tokenWidget) DefaultPrefix() string  { return w.defaultPrefix }
func (w *tokenWidget) DefaultSuffix() string  { return w.defaultSuffix }

func extractInputTokens(data *status.Session) (int, bool) {
	if data.ContextWindow == nil || data.ContextWindow.TotalInputTokens == nil {
		return 0, false
	}
	return *data.ContextWindow.TotalInputTokens, true
}

func extractOutputTokens(data *status.Session) (int, bool) {
	if data.ContextWindow == nil || data.ContextWindow.TotalOutputTokens == nil {
		return 0, false
	}
	return *data.ContextWindow.TotalOutputTokens, true
}

func extractCachedTokens(data *status.Session) (int, bool) {
	if data.ContextWindow == nil || data.ContextWindow.CurrentUsage == nil {
		return 0, false
	}
	return data.ContextWindow.CurrentUsage.CacheReadInputTokens, true
}

func extractCurrentInputTokens(data *status.Session) (int, bool) {
	if data.ContextWindow == nil || data.ContextWindow.CurrentUsage == nil {
		return 0, false
	}
	return data.ContextWindow.CurrentUsage.InputTokens, true
}

func extractCurrentOutputTokens(data *status.Session) (int, bool) {
	if data.ContextWindow == nil || data.ContextWindow.CurrentUsage == nil {
		return 0, false
	}
	return data.ContextWindow.CurrentUsage.OutputTokens, true
}

func extractCacheCreationTokens(data *status.Session) (int, bool) {
	if data.ContextWindow == nil || data.ContextWindow.CurrentUsage == nil {
		return 0, false
	}
	return data.ContextWindow.CurrentUsage.CacheCreationInputTokens, true
}

func extractTotalTokens(data *status.Session) (int, bool) {
	if data.ContextWindow == nil {
		return 0, false
	}
	var total int
	if data.ContextWindow.TotalInputTokens != nil {
		total += *data.ContextWindow.TotalInputTokens
	}
	if data.ContextWindow.TotalOutputTokens != nil {
		total += *data.ContextWindow.TotalOutputTokens
	}
	return total, true
}
