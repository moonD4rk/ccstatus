package git

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
)

// DiffStat holds line-level diff statistics from git.
type DiffStat struct {
	Added   int
	Removed int
}

// GetDiffStat returns the number of lines added and removed in the working tree
// (staged + unstaged) compared to HEAD. Returns zero values if not in a git
// repository or on error.
func GetDiffStat() DiffStat {
	// Staged changes (index vs HEAD)
	staged := diffShortStat("--cached")
	// Unstaged changes (working tree vs index)
	unstaged := diffShortStat()
	return DiffStat{
		Added:   staged.Added + unstaged.Added,
		Removed: staged.Removed + unstaged.Removed,
	}
}

// diffShortStat runs git diff --shortstat with optional extra args and parses the output.
// Output format: " 3 files changed, 10 insertions(+), 5 deletions(-)"
func diffShortStat(extraArgs ...string) DiffStat {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()

	args := append([]string{"diff", "--shortstat"}, extraArgs...)
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.Output()
	if err != nil {
		return DiffStat{}
	}
	return parseShortStat(strings.TrimSpace(string(out)))
}

// parseShortStat parses git diff --shortstat output.
// Examples:
//
//	" 3 files changed, 10 insertions(+), 5 deletions(-)"
//	" 1 file changed, 2 insertions(+)"
//	" 1 file changed, 3 deletions(-)"
//	""
func parseShortStat(line string) DiffStat {
	if line == "" {
		return DiffStat{}
	}
	var stat DiffStat
	for part := range strings.SplitSeq(line, ",") {
		part = strings.TrimSpace(part)
		fields := strings.Fields(part)
		if len(fields) < 2 {
			continue
		}
		n, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		switch {
		case strings.Contains(fields[1], "insertion"):
			stat.Added = n
		case strings.Contains(fields[1], "deletion"):
			stat.Removed = n
		}
	}
	return stat
}
