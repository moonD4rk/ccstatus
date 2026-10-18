package widget

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

func displayItem(mode string) config.WidgetItem {
	return config.WidgetItem{Metadata: map[string]string{"display": mode}}
}

func TestPromptCacheWidget(t *testing.T) {
	settings := config.DefaultSettings()
	w := Get("prompt-cache")
	require.NotNil(t, w)

	warm := &status.PromptCache{
		Warm:            boolPtr(true),
		CachingObserved: boolPtr(true),
		ExpiresAt:       int64Ptr(time.Now().Add(42*time.Minute + 30*time.Second).Unix()),
		Misses:          intPtr(2),
		HitRatio:        floatPtr(0.91),
		LastMissCause:   &status.MissCause{Causes: []string{"tools_changed"}},
	}
	cold := &status.PromptCache{Warm: boolPtr(false), CachingObserved: boolPtr(true), Misses: intPtr(0)}
	expired := &status.PromptCache{Warm: boolPtr(true), ExpiresAt: int64Ptr(time.Now().Add(-time.Minute).Unix())}
	ctxFor := func(pc *status.PromptCache) RenderContext {
		return RenderContext{Data: &status.Session{PromptCache: pc}}
	}

	tests := []struct {
		name string
		pc   *status.PromptCache
		item config.WidgetItem
		want string
	}{
		{"status warm", warm, config.WidgetItem{}, "warm 42m"},
		{"status cold", cold, config.WidgetItem{}, "cold"},
		{"expired countdown reads cold", expired, config.WidgetItem{}, "cold"},
		{"warm without expiry", &status.PromptCache{Warm: boolPtr(true)}, config.WidgetItem{}, "warm"},
		{"ttl", warm, displayItem("ttl"), "42m"},
		{"ttl cold", cold, displayItem("ttl"), "cold"},
		{"hit", warm, displayItem("hit"), "91%"},
		{"hit raw", warm, config.WidgetItem{RawValue: true, Metadata: map[string]string{"display": "hit"}}, "91.0"},
		{"hit absent", cold, displayItem("hit"), ""},
		{"misses with cause", warm, displayItem("misses"), "2 misses (tools_changed)"},
		{"no misses", cold, displayItem("misses"), "0 misses"},
		{"one miss", &status.PromptCache{Misses: intPtr(1)}, displayItem("misses"), "1 miss"},
		{"full", warm, displayItem("full"), "warm 42m · 91% hit · 2 misses (tools_changed)"},
		{"full without misses", cold, displayItem("full"), "cold"},
		{"caching never observed hides it", &status.PromptCache{CachingObserved: boolPtr(false), Warm: boolPtr(false)}, config.WidgetItem{}, ""},
		{"absent", nil, config.WidgetItem{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, w.Render(&tt.item, ctxFor(tt.pc), &settings))
		})
	}

	t.Run("state-driven default color", func(t *testing.T) {
		dyn, ok := w.(DynamicDefaultColor)
		require.True(t, ok)
		assert.Equal(t, "green", dyn.DefaultColorFor(ctxFor(warm)))
		assert.Equal(t, "brightBlack", dyn.DefaultColorFor(ctxFor(cold)))
		assert.Empty(t, dyn.DefaultColorFor(ctxFor(nil)))
	})

	t.Run("value is minutes left, or hit percentage in hit mode", func(t *testing.T) {
		v := w.(Valuer)
		minutes, ok := v.Value(&config.WidgetItem{}, ctxFor(warm))
		require.True(t, ok)
		assert.InDelta(t, 42.5, minutes, 0.1)
		hitItem := displayItem("hit")
		hit, ok := v.Value(&hitItem, ctxFor(warm))
		require.True(t, ok)
		assert.InDelta(t, 91, hit, 0.001)
		_, ok = v.Value(&config.WidgetItem{}, ctxFor(nil))
		assert.False(t, ok)
	})
}

func TestSpendLimitWidget(t *testing.T) {
	settings := config.DefaultSettings()
	w := Get("spend-limit")
	require.NotNil(t, w)

	resets := time.Now().Add(12*24*time.Hour + 4*time.Hour + 30*time.Minute).Unix()
	full := &status.SpendLimit{
		RateLimitWindow: status.RateLimitWindow{UsedPercentage: floatPtr(62.8), ResetsAt: &resets},
		UsedUSD:         floatPtr(314.12),
		LimitUSD:        floatPtr(500),
		Period:          "monthly",
	}
	pctOnly := &status.SpendLimit{RateLimitWindow: status.RateLimitWindow{UsedPercentage: floatPtr(104.4)}}
	ctxFor := func(sl *status.SpendLimit) RenderContext {
		return RenderContext{Data: &status.Session{RateLimits: &status.RateLimits{SpendLimit: sl}}}
	}

	tests := []struct {
		name string
		sl   *status.SpendLimit
		item config.WidgetItem
		want string
	}{
		{"amount", full, config.WidgetItem{}, "$314.12 / $500.00"},
		{"amount falls back to percent", pctOnly, config.WidgetItem{}, "104%"},
		{"percent is not clamped", pctOnly, displayItem("percent"), "104%"},
		{"percent raw", full, config.WidgetItem{RawValue: true, Metadata: map[string]string{"display": "percent"}}, "62.8"},
		{"bar", full, config.WidgetItem{Metadata: map[string]string{"display": "bar", "barWidth": "10"}}, "▓▓▓▓▓▓░░░░ 63%"},
		{"bar over the limit stays full width", pctOnly, config.WidgetItem{Metadata: map[string]string{"display": "bar", "barWidth": "4"}}, "▓▓▓▓ 104%"},
		{"reset", full, displayItem("reset"), "12d4h"},
		{"full", full, displayItem("full"), "$314.12 / $500.00 monthly, resets 12d4h"},
		{"full with percent only", pctOnly, displayItem("full"), "104%"},
		{"absent", nil, config.WidgetItem{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, w.Render(&tt.item, ctxFor(tt.sl), &settings))
		})
	}

	v, ok := w.(Valuer).Value(&config.WidgetItem{}, ctxFor(pctOnly))
	require.True(t, ok)
	assert.InDelta(t, 104.4, v, 0.001)
}

func TestRateLimitsWidgetWithSpend(t *testing.T) {
	settings := config.DefaultSettings()
	w := Get("rate-limits")
	item := config.WidgetItem{}

	gatewayOnly := RenderContext{Data: &status.Session{RateLimits: &status.RateLimits{
		SpendLimit: &status.SpendLimit{RateLimitWindow: status.RateLimitWindow{UsedPercentage: floatPtr(63)}},
	}}}
	assert.Equal(t, "spend: 63%", w.Render(&item, gatewayOnly, &settings))

	all := RenderContext{Data: &status.Session{RateLimits: &status.RateLimits{
		FiveHour: &status.RateLimitWindow{UsedPercentage: floatPtr(24)},
		SevenDay: &status.RateLimitWindow{UsedPercentage: floatPtr(41)},
		SpendLimit: &status.SpendLimit{
			RateLimitWindow: status.RateLimitWindow{UsedPercentage: floatPtr(110)},
			UsedUSD:         floatPtr(550), LimitUSD: floatPtr(500),
		},
	}}}
	assert.Equal(t, "5h: 24% / 7d: 41% / spend: $550.00/$500.00", w.Render(&item, all, &settings))

	v, ok := w.(Valuer).Value(&item, all)
	require.True(t, ok)
	assert.InDelta(t, 110, v, 0.001, "the most constrained limit drives the color")
}

func TestSessionLinesWidget(t *testing.T) {
	settings := config.DefaultSettings()
	w := Get("session-lines")
	item := config.WidgetItem{}
	tests := []struct {
		name string
		cost *status.CostInfo
		want string
	}{
		{"added and removed", &status.CostInfo{TotalLinesAdded: intPtr(156), TotalLinesRemoved: intPtr(23)}, "+156/-23"},
		{"only added", &status.CostInfo{TotalLinesAdded: intPtr(4)}, "+4/-0"},
		{"zero hides", &status.CostInfo{TotalLinesAdded: intPtr(0), TotalLinesRemoved: intPtr(0)}, ""},
		{"no cost", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := RenderContext{Data: &status.Session{Cost: tt.cost}}
			assert.Equal(t, tt.want, w.Render(&item, ctx, &settings))
		})
	}
}

func TestPRWidgetMergeRequest(t *testing.T) {
	settings := config.DefaultSettings()
	w := Get("pr")
	ctx := RenderContext{Data: &status.Session{PR: &status.PRInfo{Number: intPtr(42), ReviewState: "approved", Kind: "mr"}}}
	assert.Equal(t, "!42 approved", w.Render(&config.WidgetItem{}, ctx, &settings))
	assert.Equal(t, "42", w.Render(&config.WidgetItem{RawValue: true}, ctx, &settings))
}

func TestPercentageBars(t *testing.T) {
	settings := config.DefaultSettings()
	ctx := RenderContext{Data: &status.Session{ContextWindow: &status.ContextWindow{
		UsedPercentage: floatPtr(32), RemainingPercentage: floatPtr(68),
	}}}
	bar := config.WidgetItem{Metadata: map[string]string{"display": "bar"}}
	assert.Equal(t, "▓▓▓░░░░░░░ 32%", Get("context-percentage").Render(&bar, ctx, &settings))
	assert.Equal(t, "▓▓▓▓▓▓▓░░░ 68%", Get("remaining-percentage").Render(&bar, ctx, &settings))

	usable := RenderContext{Data: &status.Session{ContextWindow: &status.ContextWindow{
		ContextWindowSize: intPtr(200_000), TotalInputTokens: intPtr(80_000),
	}}}
	assert.Equal(t, "▓▓▓▓▓░░░░░ 50%", Get("context-percentage-usable").Render(&bar, usable, &settings))
}

func TestFormatDurationDays(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "<1m"},
		{42 * time.Minute, "42m"},
		{2*time.Hour + 13*time.Minute, "2h13m"},
		{24 * time.Hour, "1d"},
		{4*24*time.Hour + 15*time.Hour + 12*time.Minute, "4d15h"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, formatDuration(float64(tt.d.Milliseconds())), tt.d.String())
	}
}

func TestNumericWidgetsExposeValues(t *testing.T) {
	ctx := RenderContext{Data: &status.Session{
		Cost: &status.CostInfo{TotalCostUSD: floatPtr(1.25)},
		ContextWindow: &status.ContextWindow{
			UsedPercentage: floatPtr(32), TotalInputTokens: intPtr(64_000), ContextWindowSize: intPtr(200_000),
		},
		RateLimits: &status.RateLimits{FiveHour: &status.RateLimitWindow{UsedPercentage: floatPtr(120)}},
	}}
	tests := []struct {
		widget string
		want   float64
	}{
		{"context-percentage", 32},
		{"session-cost", 1.25},
		{"context-length", 64_000},
		{"tokens-input", 64_000},
		{"context-percentage-usable", 40},
		{"rate-limit-5h", 100},
	}
	for _, tt := range tests {
		t.Run(tt.widget, func(t *testing.T) {
			v, ok := Get(tt.widget).(Valuer)
			require.True(t, ok, "%s should implement Valuer", tt.widget)
			got, ok := v.Value(&config.WidgetItem{}, ctx)
			require.True(t, ok)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}
