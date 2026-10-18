// Package git provides git repository information extraction.
package git

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

const gitTimeout = 2 * time.Second

// Repo runs git commands against a working directory. The zero value runs git
// in the process's own current working directory; set Dir to the session
// directory (e.g. workspace.current_dir) so widgets read the right repository
// even when Claude Code launches the status line from elsewhere.
type Repo struct {
	Dir string
}

// run executes `git args...` in the repo directory and returns trimmed stdout.
// The bool is false on any failure (not a repo, git missing, timeout).
func (r Repo) run(args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	// Never take index.lock: the status line runs while Claude runs its own git commands.
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

// Branch returns the current git branch name, or empty string if not in a git repository.
func (r Repo) Branch() string {
	out, _ := r.run("rev-parse", "--abbrev-ref", "HEAD")
	return out
}
