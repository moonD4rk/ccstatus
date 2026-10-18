package status

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moond4rk/ccstatus/internal/sample"
)

// docsFixture is the full sample payload from the official status line docs.
func docsFixture(t *testing.T) []byte {
	t.Helper()
	return sample.StatusLineDocs
}

func TestParseDocsFixture(t *testing.T) {
	s, dropped, err := Parse(docsFixture(t))
	require.NoError(t, err)
	assert.Empty(t, dropped)

	assert.Equal(t, "Opus", s.Model.DisplayName)
	assert.Equal(t, "anthropics", s.Workspace.Repo.Owner)

	pc := s.PromptCache
	require.NotNil(t, pc)
	assert.True(t, *pc.Warm)
	assert.True(t, *pc.CachingObserved)
	assert.Equal(t, "1h", pc.TTL)
	assert.Equal(t, int64(1738429200), *pc.ExpiresAt)
	assert.Equal(t, 2, *pc.Misses)
	assert.InDelta(t, 0.91, *pc.HitRatio, 1e-9)
	assert.Equal(t, []string{"tools_changed"}, pc.LastMissCause.Causes)
	assert.Equal(t, 2, *pc.LastMissCause.ToolsAdded)
	assert.Equal(t, map[string]int{"tools_changed": 2}, pc.MissCauses)
	assert.Equal(t, 45000, *pc.RecacheTokensIfCold)

	sl := s.RateLimits.SpendLimit
	require.NotNil(t, sl)
	assert.InDelta(t, 62.8, *sl.UsedPercentage, 1e-9)
	assert.Equal(t, int64(1740787200), *sl.ResetsAt)
	assert.InDelta(t, 314.12, *sl.UsedUSD, 1e-9)
	assert.InDelta(t, 500, *sl.LimitUSD, 1e-9)
	assert.Equal(t, "monthly", sl.Period)

	assert.Equal(t, "pending", s.PR.ReviewState)
	assert.False(t, s.PR.IsMergeRequest())
}

// TestSessionModelsEveryDocumentedField fails when the docs fixture carries a
// field that Session drops, so schema additions are noticed when the fixture is
// refreshed from the docs.
func TestSessionModelsEveryDocumentedField(t *testing.T) {
	data := docsFixture(t)
	s, _, err := Parse(data)
	require.NoError(t, err)
	roundTrip, err := json.Marshal(s)
	require.NoError(t, err)

	var want, got any
	require.NoError(t, json.Unmarshal(data, &want))
	require.NoError(t, json.Unmarshal(roundTrip, &got))
	wantPaths, gotPaths := map[string]bool{}, map[string]bool{}
	leafPaths("", want, wantPaths)
	leafPaths("", got, gotPaths)
	for path := range wantPaths {
		assert.Truef(t, gotPaths[path], "documented field %q is not modeled by Session", path)
	}
}

// leafPaths records the dotted path of every scalar and non-empty array in v.
func leafPaths(prefix string, v any, out map[string]bool) {
	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			leafPaths(joinPath(prefix, k), val, out)
		}
	case []any:
		if len(x) > 0 {
			out[prefix] = true
		}
	default:
		out[prefix] = true
	}
}

func ptr[T any](v T) *T { return &v }

func TestIsMergeRequest(t *testing.T) {
	var none *PRInfo
	assert.False(t, none.IsMergeRequest())
	assert.False(t, (&PRInfo{}).IsMergeRequest())
	assert.True(t, (&PRInfo{Kind: "mr"}).IsMergeRequest())
}
