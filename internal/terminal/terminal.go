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
// If override > 0 it wins (the `terminalWidth` setting). Otherwise the order is:
//  1. COLUMNS env var — authoritative under the v2.1.153+ status line spec,
//     which captures the command's output (so the fds are not a terminal) and
//     sets COLUMNS/LINES to the live dimensions before each run. Costs no
//     subprocess and is re-set every invocation, so it is not stale.
//  2. stdout/stderr fd — direct CLI use, and Claude Code versions before the
//     COLUMNS contract.
//  3. /dev/tty — our own controlling terminal, if not detached from it.
//  4. an ancestor process's controlling terminal — last resort that spawns
//     `ps`, reached only when everything above failed.
//  5. the default 80.
func Width(override int) int {
	if override > 0 {
		return override
	}
	if w := widthFromColumns(); w > 0 {
		return w
	}
	for _, f := range []*os.File{os.Stdout, os.Stderr} {
		if w, _, err := term.GetSize(int(f.Fd())); err == nil && w > 0 { //nolint:gosec // fd values are small, no overflow risk
			return w
		}
	}
	if w := widthFromTTYPath("/dev/tty"); w > 0 {
		return w
	}
	if w := widthFromAncestorTTY(); w > 0 {
		return w
	}
	return defaultWidth
}

// widthFromColumns parses the COLUMNS environment variable, returning 0 when it
// is unset or not a positive integer.
func widthFromColumns() int {
	cols := os.Getenv("COLUMNS")
	if cols == "" {
		return 0
	}
	if n, err := strconv.Atoi(cols); err == nil && n > 0 {
		return n
	}
	return 0
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
