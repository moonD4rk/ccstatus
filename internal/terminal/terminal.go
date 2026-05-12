// Package terminal provides terminal width detection.
package terminal

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	// defaultWidth is the fallback terminal width when detection fails.
	defaultWidth = 80
	// maxParentWalk bounds how far up the process tree Width climbs looking
	// for an ancestor that owns a controlling terminal.
	maxParentWalk = 8
	// psTimeout bounds the `ps` call used to inspect an ancestor process.
	psTimeout = time.Second
)

// Width returns the terminal width in columns.
//
// If override > 0 it wins (the `terminalWidth` setting; use it when running
// under a host like Claude Code that pipes stdio and may run the status line
// command detached from the terminal). Otherwise the detection order is:
// stdout fd, stderr fd, /dev/tty, the controlling terminal of an ancestor
// process, the COLUMNS env var, then the default 80.
func Width(override int) int {
	if override > 0 {
		return override
	}
	// 1. A standard fd that happens to be a real terminal.
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 { //nolint:gosec // fd values are small, no overflow risk
			return w
		}
	}
	// 2. Our own controlling terminal (works unless we were detached from it).
	if w := widthFromTTYPath("/dev/tty"); w > 0 {
		return w
	}
	// 3. The controlling terminal of an ancestor process. This recovers the
	//    real width when the host ran the status line command detached from
	//    the terminal (so /dev/tty fails) but an ancestor still owns the tty.
	if w := widthFromAncestorTTY(); w > 0 {
		return w
	}
	// 4. The COLUMNS env var (captured at spawn time, so it can be stale).
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if n, err := strconv.Atoi(cols); err == nil && n > 0 {
			return n
		}
	}
	return defaultWidth
}

// widthFromTTYPath opens a tty device path and returns its width, or 0.
func widthFromTTYPath(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	w, _, err := term.GetSize(int(f.Fd())) //nolint:gosec // fd values are small, no overflow risk
	if err != nil {
		return 0
	}
	return w
}

// widthFromAncestorTTY walks up the process tree and returns the width of the
// first ancestor that has a controlling terminal, or 0 if none is found.
func widthFromAncestorTTY() int {
	pid := os.Getppid()
	for range maxParentWalk {
		if pid <= 1 {
			break
		}
		ppid, tty := ppidAndTTY(pid)
		if w := widthFromTTYName(tty); w > 0 {
			return w
		}
		if ppid <= 1 || ppid == pid {
			break
		}
		pid = ppid
	}
	return 0
}

// ppidAndTTY reports a process's parent pid and controlling tty name via `ps`.
// Returns (0, "") if the lookup fails.
func ppidAndTTY(pid int) (ppid int, tty string) {
	ctx, cancel := context.WithTimeout(context.Background(), psTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ps", "-o", "ppid=,tty=", "-p", strconv.Itoa(pid)) //nolint:gosec // sanitized pid, not user input
	out, err := cmd.Output()
	if err != nil {
		return 0, ""
	}
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return 0, ""
	}
	ppid, _ = strconv.Atoi(fields[0])
	if len(fields) > 1 {
		tty = fields[1]
	}
	return ppid, tty
}

// widthFromTTYName resolves a `ps`-style tty name (e.g. "ttys003" on macOS,
// "pts/3" on Linux) to a device path and returns its width, or 0 if the name is
// not a real terminal.
func widthFromTTYName(name string) int {
	name = strings.TrimSpace(name)
	switch name {
	case "", "?", "??", "-":
		return 0
	}
	path := name
	if !strings.HasPrefix(path, "/") {
		path = "/dev/" + path
	}
	return widthFromTTYPath(path)
}
