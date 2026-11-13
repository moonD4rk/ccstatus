package main

import (
	"encoding/json"
	"os"
	"path/filepath"
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
		"Opus · high | Ctx: 8% | worktree-my-feature feature-xyz | +42 -7 | $0.01\n"+
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
		{Type: "prompt-cache"},
		{Type: "separator"},
		{Type: "rate-limit-5h", Display: "reset"},
		{Type: "separator"},
		{Type: "spend-limit", Display: "reset"},
		{Type: "separator"},
		{Type: "block-timer"},
	}}
	require.NoError(t, config.Save(&s))

	for range 3 {
		out, _, err := execRoot(t, "", "preview", "--width", "200")
		require.NoError(t, err)
		assert.Equal(t, "Cache: warm 42m | 5h limit: 2h13m | Spend: 12d4h | Block: 2h47m/5h\n", plain(out))
	}
}

func TestPreviewGitData(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := config.DefaultSettings()
	s.ColorLevel = 0
	s.Lines = [][]config.WidgetItem{{
		{Type: "git-branch"}, {Type: "git-changes"}, {Type: "lines-changed"}, {Type: "git-worktree"},
	}}
	require.NoError(t, config.Save(&s))

	t.Run("the built-in sample comes with a sample repository", func(t *testing.T) {
		out, _, err := execRoot(t, "", "preview", "--width", "200")
		require.NoError(t, err)
		assert.Equal(t, "worktree-my-feature 3 +42/-7 feature-xyz\n", plain(out))
	})

	t.Run("an input file reads the repository it names", func(t *testing.T) {
		dir := t.TempDir() // not a repository
		path := filepath.Join(dir, "payload.json")
		payload, err := json.Marshal(map[string]any{"cwd": dir, "version": "2.1.300"})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, payload, 0o600))

		out, _, err := execRoot(t, "", "preview", "--input", path, "--width", "200")
		require.NoError(t, err)
		assert.Empty(t, plain(out))
	})
}
