//go:build unix

package proc

import (
	"os/exec"
	"syscall"
)

// killGroupOnCancel starts cmd in its own process group and kills that group,
// rather than only the direct child, when the command's context ends.
func killGroupOnCancel(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
			return cmd.Process.Kill()
		}
		return nil
	}
}
