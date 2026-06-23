package git

import "strings"

// Changes returns the number of uncommitted changes (staged + unstaged + untracked).
// Returns 0 if not in a git repository or on error.
func (r Repo) Changes() int {
	out, ok := r.run("status", "--porcelain")
	if !ok || out == "" {
		return 0
	}
	return len(strings.Split(out, "\n"))
}
