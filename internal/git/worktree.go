package git

import (
	"path/filepath"
)

// Worktree returns the worktree name if the current directory is a git worktree.
// Returns empty string if in the main working tree or not in a git repository.
func (r Repo) Worktree() string {
	// For linked worktrees, --absolute-git-dir returns something like:
	//   /path/to/main/.git/worktrees/<name>
	// For the main worktree, it returns the repo's /path/to/repo/.git.
	// --absolute-git-dir (vs --git-dir) guarantees an absolute path regardless
	// of the directory git runs in, so detection works when Repo.Dir is set.
	gitDir, ok := r.run("rev-parse", "--absolute-git-dir")
	if !ok {
		return ""
	}

	dir, name := filepath.Split(gitDir)
	dir = filepath.Clean(dir)
	if filepath.Base(dir) == "worktrees" && name != "" {
		return name
	}
	return ""
}
