package widget

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
)

func TestRateLimitPercentIsClamped(t *testing.T) {
	settings := config.DefaultSettings()
	ctx := RenderContext{Data: &status.Session{RateLimits: &status.RateLimits{
		FiveHour: &status.RateLimitWindow{UsedPercentage: floatPtr(104.2)},
		SevenDay: &status.RateLimitWindow{UsedPercentage: floatPtr(-3)},
	}}}

	tests := []struct {
		widget string
		item   config.WidgetItem
		want   string
	}{
		{"rate-limit-5h", config.WidgetItem{}, "100%"},
		{"rate-limit-5h", config.WidgetItem{RawValue: true}, "100.0"},
		{"rate-limit-5h", config.WidgetItem{Metadata: map[string]string{"display": "bar", "barWidth": "4"}}, "▓▓▓▓ 100%"},
		{"rate-limit-7d", config.WidgetItem{}, "0%"},
		{"rate-limits", config.WidgetItem{}, "5h: 100% / 7d: 0%"},
	}
	for _, tt := range tests {
		t.Run(tt.widget, func(t *testing.T) {
			assert.Equal(t, tt.want, Get(tt.widget).Render(&tt.item, ctx, &settings))
		})
	}
}

func TestDisplayModes(t *testing.T) {
	tests := []struct {
		widget string
		want   []string
		wantOK bool
	}{
		{"rate-limit-5h", []string{"percent", "bar", "reset", "full"}, true},
		{"rate-limit-7d", []string{"percent", "bar", "reset", "full"}, true},
		{"block-timer", []string{"time", "progress", "percentage"}, true},
		{"added-dirs", []string{"count", "list"}, true},
		{"model", nil, false},
		{"no-such-widget", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.widget, func(t *testing.T) {
			modes, ok := DisplayModes(tt.widget)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, modes)
		})
	}
}
