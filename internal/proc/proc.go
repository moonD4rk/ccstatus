// Package proc runs helper commands that must not outlive their deadline.
package proc

import (
	"context"
	"os/exec"
	"time"
)

// waitDelay bounds how long Wait lingers on an output pipe that a descendant of
// a finished or killed command still holds open.
const waitDelay = 100 * time.Millisecond

// Command is exec.CommandContext for a command the status line cannot wait on.
// When ctx ends the command's whole process group is killed, so the children of
// an `sh -c` die with the shell, and a descendant that escaped the group delays
// Wait by at most waitDelay: Wait then fails with exec.ErrWaitDelay, with the
// output read so far still returned.
func Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.WaitDelay = waitDelay
	killGroupOnCancel(cmd)
	return cmd
}
