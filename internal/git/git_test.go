package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// isolateGit points HOME/XDG and git's config vars at throwaway locations so
// neither the host's global config nor its global ignore file (which can hide
// the scratch repo's files) affects the test. It applies to both the gitExec
// setup commands and the production Repo.run calls, which inherit os.Environ.
func isolateGit(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, "nonexistent-gitconfig"))
	t.Setenv("GIT_CONFIG_SYSTEM", filepath.Join(home, "nonexistent-gitconfig"))
	t.Setenv("GIT_AUTHOR_NAME", "test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
}

func gitExec(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "git %v: %s", args, out)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func TestRepoAgainstScratchRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	isolateGit(t)

	dir := t.TempDir()
	gitExec(t, dir, "init", "-q")
	writeFile(t, filepath.Join(dir, "README.md"), "line1\n")
	gitExec(t, dir, "add", "README.md")
	gitExec(t, dir, "commit", "-q", "-m", "init")
	gitExec(t, dir, "branch", "-M", "main")

	repo := Repo{Dir: dir}

	t.Run("Branch", func(t *testing.T) {
		assert.Equal(t, "main", repo.Branch())
	})

	t.Run("clean tree", func(t *testing.T) {
		assert.Equal(t, 0, repo.Changes())
		assert.Equal(t, DiffStat{}, repo.Diff())
		assert.Empty(t, repo.Worktree())
	})

	t.Run("modified and untracked", func(t *testing.T) {
		writeFile(t, filepath.Join(dir, "README.md"), "line1\nline2\nline3\n")
		writeFile(t, filepath.Join(dir, "extra.txt"), "new\n")
		assert.Equal(t, 2, repo.Changes()) // one modified, one untracked
		assert.Equal(t, DiffStat{Added: 2}, repo.Diff())
	})

	t.Run("Dir, not process cwd, selects the repo", func(t *testing.T) {
		assert.Empty(t, Repo{Dir: t.TempDir()}.Branch())
	})

	t.Run("linked worktree", func(t *testing.T) {
		wtPath := filepath.Join(t.TempDir(), "wt")
		gitExec(t, dir, "worktree", "add", "-q", "-b", "feature", wtPath)
		assert.Equal(t, "wt", Repo{Dir: wtPath}.Worktree())
		assert.Equal(t, "feature", Repo{Dir: wtPath}.Branch())
	})
}
