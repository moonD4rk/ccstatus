package git

import (
	"strconv"
	"strings"
)

// DiffStat holds line-level diff statistics from git.
type DiffStat struct {
	Added   int
	Removed int
}

// Diff returns the number of lines added and removed in the working tree
// (staged + unstaged) compared to HEAD. It uses plumbing that never rewrites the
// index and --numstat, whose output is not localized. Returns zero values if not
// in a git repository or on error.
func (r Repo) Diff() DiffStat {
	if out, ok := r.run("diff-index", "--numstat", "HEAD"); ok {
		return parseNumStat(out)
	}
	// No commit yet (unborn HEAD): sum the unstaged and staged changes instead.
	unstaged, ok := r.run("diff-files", "--numstat")
	if !ok {
		return DiffStat{}
	}
	staged, _ := r.run("diff", "--cached", "--numstat")
	u, s := parseNumStat(unstaged), parseNumStat(staged)
	return DiffStat{Added: u.Added + s.Added, Removed: u.Removed + s.Removed}
}

// parseNumStat sums `git diff --numstat` output, one "<added>\t<removed>\t<path>"
// line per file. Binary files report "-" and count as zero.
func parseNumStat(out string) DiffStat {
	var stat DiffStat
	for line := range strings.Lines(out) {
		added, rest, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		removed, _, _ := strings.Cut(rest, "\t")
		if n, err := strconv.Atoi(added); err == nil {
			stat.Added += n
		}
		if n, err := strconv.Atoi(removed); err == nil {
			stat.Removed += n
		}
	}
	return stat
}
