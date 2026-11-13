package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moond4rk/ccstatus/internal/config"
)

// TestPreviewDefaultLayout pins what the default settings render for the docs
// samples. The sample timestamps and the render share one instant, so the
// countdowns and elapsed times are exact on every run.
func TestPreviewDefaultLayout(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // no settings file: the defaults

	out, _, err := execRoot(t, "", "preview", "--width", "200")
	require.NoError(t, err)
	assert.Equal(t,
		"Opus · high | Ctx: 8% | feature-xyz | $0.01\n"+
			"/current/working/directory | Limit 5h: 24% / 7d: 41% / spend: $314.12/$500.00 | Session: <1m | my-session\n",
		plain(out))

	out, _, err = execRoot(t, "", "preview", "--subagent")
	require.NoError(t, err)
	assert.Equal(t,
		"reviewer running 1m23s | Sonnet 5.5 · high 45.2k 23% ▅▄▆▇██▇ | Reading internal/auth/session.go\n"+
			"Explore completed | Haiku 5.5 · 16k 18.4k 9% | Find every caller of Session.Refresh\n",
		plain(out))
}

func TestPreviewTimeWidgetsAreExact(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := config.DefaultSettings()
	s.ColorLevel = 0
	s.Lines = [][]config.WidgetItem{{
		{ID: "1", Type: "prompt-cache"},
		{ID: "2", Type: "separator"},
		{ID: "3", Type: "rate-limit-5h", Metadata: map[string]string{"display": "reset"}},
		{ID: "4", Type: "separator"},
		{ID: "5", Type: "spend-limit", Metadata: map[string]string{"display": "reset"}},
		{ID: "6", Type: "separator"},
		{ID: "7", Type: "block-timer"},
	}}
	require.NoError(t, config.Save(&s))

	for range 3 {
		out, _, err := execRoot(t, "", "preview", "--width", "200")
		require.NoError(t, err)
		assert.Equal(t, "Cache: warm 42m | 5h limit: 2h13m | Spend: 12d4h | Block: 2h47m/5h\n", plain(out))
	}
}
