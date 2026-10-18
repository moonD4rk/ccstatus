package sample

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusLineShiftsTimestamps(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	var doc struct {
		PromptCache struct {
			ExpiresAt int64 `json:"expires_at"`
			TTL       string
		} `json:"prompt_cache"`
		RateLimits struct {
			FiveHour struct {
				ResetsAt int64 `json:"resets_at"`
			} `json:"five_hour"`
			SpendLimit struct {
				ResetsAt int64 `json:"resets_at"`
			} `json:"spend_limit"`
		} `json:"rate_limits"`
		Model struct {
			DisplayName string `json:"display_name"`
		} `json:"model"`
	}
	require.NoError(t, json.Unmarshal(StatusLine(now), &doc))
	assert.Equal(t, now.Add(42*time.Minute+30*time.Second).Unix(), doc.PromptCache.ExpiresAt)
	assert.Equal(t, now.Add(2*time.Hour+13*time.Minute).Unix(), doc.RateLimits.FiveHour.ResetsAt)
	assert.Equal(t, now.Add(12*24*time.Hour+4*time.Hour).Unix(), doc.RateLimits.SpendLimit.ResetsAt)
	assert.Equal(t, "Opus", doc.Model.DisplayName, "everything else is kept")
}

func TestSubagentShiftsStartTimes(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	var doc struct {
		Columns int `json:"columns"`
		Tasks   []struct {
			ID        string `json:"id"`
			StartTime int64  `json:"startTime"`
		} `json:"tasks"`
	}
	require.NoError(t, json.Unmarshal(Subagent(now), &doc))
	assert.Equal(t, 100, doc.Columns)
	require.Len(t, doc.Tasks, 2)
	assert.Equal(t, now.Add(-83*time.Second).UnixMilli(), doc.Tasks[0].StartTime)
	assert.Equal(t, "task-2", doc.Tasks[1].ID)
}

func TestShiftKeepsUnparsablePayload(t *testing.T) {
	assert.Equal(t, []byte("not json"), shift([]byte("not json"), func(map[string]any) {}))
}
