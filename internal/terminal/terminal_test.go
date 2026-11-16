package terminal

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWidth(t *testing.T) {
	t.Run("override wins", func(t *testing.T) {
		assert.Equal(t, 150, Width(150))
		assert.Equal(t, 200, Width(200))
	})

	t.Run("auto-detect returns a positive width", func(t *testing.T) {
		// With no override and (under CI) no usable TTY, Width still returns
		// a sane positive value (the default fallback at minimum).
		assert.Positive(t, Width(0))
	})

	t.Run("non-positive override is ignored", func(t *testing.T) {
		assert.Positive(t, Width(0))
		assert.Positive(t, Width(-5))
	})

	t.Run("COLUMNS is honored before the attached terminal", func(t *testing.T) {
		t.Setenv("COLUMNS", "123")
		assert.Equal(t, 123, Width(0))
	})

	t.Run("override still beats COLUMNS", func(t *testing.T) {
		t.Setenv("COLUMNS", "123")
		assert.Equal(t, 200, Width(200))
	})

	t.Run("invalid COLUMNS falls through to a positive width", func(t *testing.T) {
		t.Setenv("COLUMNS", "not-a-number")
		assert.Positive(t, Width(0))
	})
}

func TestWidthFromColumns(t *testing.T) {
	t.Run("positive integer", func(t *testing.T) {
		t.Setenv("COLUMNS", "80")
		assert.Equal(t, 80, widthFromColumns())
	})
	for _, v := range []string{"", "0", "-3", "abc"} {
		t.Run("ignores "+v, func(t *testing.T) {
			t.Setenv("COLUMNS", v)
			assert.Zero(t, widthFromColumns())
		})
	}
}

func TestIsTerminal(t *testing.T) {
	assert.False(t, IsTerminal(strings.NewReader("{}")), "a reader that is not a file")

	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()
	defer w.Close()
	assert.False(t, IsTerminal(r), "a pipe")
}
