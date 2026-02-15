package git

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree returns the worktree name if the current directory is a git worktree.
// Returns empty string if in the main working tree or not in a git repository.
func Worktree() string {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	// Check if we are in a linked worktree.
	// For linked worktrees, --git-dir returns something like:
	//   /path/to/main/.git/worktrees/<name>
	// For the main worktree, it returns:
	//   /path/to/repo/.git
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--git-dir")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	gitDir := strings.TrimSpace(string(out))

	// Check if this is a linked worktree by looking for /worktrees/ in the path.
	dir, name := filepath.Split(gitDir)
	dir = filepath.Clean(dir)
	if filepath.Base(dir) == "worktrees" && name != "" {
		return name
	}

	return ""
}
