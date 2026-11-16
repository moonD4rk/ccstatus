// Package git provides git repository information extraction.
package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/moond4rk/ccstatus/internal/proc"
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
	out, err := r.output(args...)
	return out, err == nil
}

// output is run with the command's error, for callers that tell exit codes apart.
func (r Repo) output(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := proc.Command(ctx, "git", args...)
	if r.Dir != "" {
		cmd.Dir = r.Dir
	}
	// Never take index.lock: the status line runs while Claude runs its own git commands.
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// Branch returns the current branch name, also on a branch with no commits yet.
// On a detached HEAD it returns the abbreviated commit hash, and outside a git
// repository the empty string.
func (r Repo) Branch() string {
	out, err := r.output("symbolic-ref", "--short", "-q", "HEAD")
	if err == nil {
		return out
	}
	// With -q, exit status 1 means HEAD is not a symbolic ref, i.e. detached.
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		commit, _ := r.run("rev-parse", "--short", "HEAD")
		return commit
	}
	return ""
}
