package jsonl

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionStart(t *testing.T) {
	t.Run("valid RFC3339 timestamp", func(t *testing.T) {
		f, err := os.CreateTemp("", "test-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(f.Name())

		_, err = f.WriteString(`{"timestamp":"2025-01-15T10:30:00Z","type":"start"}` + "\n")
		require.NoError(t, err)
		f.Close()

		result := SessionStart(f.Name())
		assert.Equal(t, 2025, result.Year())
		assert.Equal(t, time.January, result.Month())
		assert.Equal(t, 15, result.Day())
		assert.Equal(t, 10, result.Hour())
		assert.Equal(t, 30, result.Minute())
	})

	t.Run("valid RFC3339Nano timestamp", func(t *testing.T) {
		f, err := os.CreateTemp("", "test-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(f.Name())

		_, err = f.WriteString(`{"timestamp":"2025-01-15T10:30:00.123456789Z","type":"start"}` + "\n")
		require.NoError(t, err)
		f.Close()

		result := SessionStart(f.Name())
		assert.False(t, result.IsZero())
		assert.Equal(t, 10, result.Hour())
	})

	t.Run("multiple lines returns first", func(t *testing.T) {
		f, err := os.CreateTemp("", "test-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(f.Name())

		_, err = f.WriteString(`{"timestamp":"2025-01-15T10:00:00Z"}` + "\n")
		require.NoError(t, err)
		_, err = f.WriteString(`{"timestamp":"2025-01-15T11:00:00Z"}` + "\n")
		require.NoError(t, err)
		f.Close()

		result := SessionStart(f.Name())
		assert.Equal(t, 10, result.Hour())
	})

	t.Run("empty file returns zero", func(t *testing.T) {
		f, err := os.CreateTemp("", "test-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(f.Name())
		f.Close()

		result := SessionStart(f.Name())
		assert.True(t, result.IsZero())
	})

	t.Run("invalid JSON returns zero", func(t *testing.T) {
		f, err := os.CreateTemp("", "test-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(f.Name())

		_, err = f.WriteString("{invalid}\n")
		require.NoError(t, err)
		f.Close()

		result := SessionStart(f.Name())
		assert.True(t, result.IsZero())
	})

	t.Run("missing timestamp field returns zero", func(t *testing.T) {
		f, err := os.CreateTemp("", "test-*.jsonl")
		require.NoError(t, err)
		defer os.Remove(f.Name())

		_, err = f.WriteString(`{"type":"start"}` + "\n")
		require.NoError(t, err)
		f.Close()

		result := SessionStart(f.Name())
		assert.True(t, result.IsZero())
	})

	t.Run("nonexistent file returns zero", func(t *testing.T) {
		result := SessionStart("/nonexistent/path/transcript.jsonl")
		assert.True(t, result.IsZero())
	})

	t.Run("empty path returns zero", func(t *testing.T) {
		result := SessionStart("")
		assert.True(t, result.IsZero())
	})
}
