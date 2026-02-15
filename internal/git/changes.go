package git

import (
	"context"
	"os/exec"
	"strings"
)

// GetChanges returns the number of uncommitted changes (staged + unstaged + untracked).
// Returns 0 if not in a git repository or on error.
func GetChanges() int {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return 0
	}
	return len(strings.Split(trimmed, "\n"))
}
