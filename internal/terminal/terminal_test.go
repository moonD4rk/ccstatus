package terminal

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	t.Run("COLUMNS is honored before fd/tty fallbacks", func(t *testing.T) {
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

func TestWidthFromTTYName(t *testing.T) {
	for _, name := range []string{"", " ", "?", "??", "-", "nonexistent-tty-xyz"} {
		assert.Zero(t, widthFromTTYName(name), "name %q should not resolve to a width", name)
	}
}

func TestPpidAndTTY(t *testing.T) {
	// Looking up an obviously invalid pid must not panic and must report no tty.
	ppid, tty := ppidAndTTY(-1)
	assert.Zero(t, ppid)
	assert.Empty(t, tty)
}
