package proc

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireShell(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not installed")
	}
}

func TestCommandReturnsOutput(t *testing.T) {
	requireShell(t)
	out, err := Command(context.Background(), "sh", "-c", "echo hello").Output()
	require.NoError(t, err)
	assert.Equal(t, "hello\n", string(out))
}

func TestCommandDeadlineCoversChildren(t *testing.T) {
	requireShell(t)
	tests := []struct {
		name   string
		script string
	}{
		{"direct child", "sleep 5"},
		{"compound command", "sleep 5; echo late"},
		{"pipeline", "sleep 5 | cat"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			start := time.Now()
			out, err := Command(ctx, "sh", "-c", tt.script).Output()
			require.Error(t, err)
			assert.Empty(t, out)
			assert.Less(t, time.Since(start), 2*time.Second, "the deadline must not wait for the shell's children")
		})
	}
}

func TestCommandDoesNotWaitForBackgroundChild(t *testing.T) {
	requireShell(t)
	start := time.Now()
	out, err := Command(context.Background(), "sh", "-c", "sleep 5 & echo ready").Output()
	require.ErrorIs(t, err, exec.ErrWaitDelay)
	assert.Equal(t, "ready\n", string(out))
	assert.Less(t, time.Since(start), 2*time.Second)
}
