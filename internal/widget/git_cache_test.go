package widget

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/git"
	"github.com/moond4rk/ccstatus/internal/status"
)

// countingGit wraps fakeGit and counts calls, safely under concurrent prefetch.
type countingGit struct {
	fakeGit
	branchCalls, changesCalls, diffCalls, worktreeCalls atomic.Int32
}

func (c *countingGit) Branch() string     { c.branchCalls.Add(1); return c.fakeGit.Branch() }
func (c *countingGit) Changes() int       { c.changesCalls.Add(1); return c.fakeGit.Changes() }
func (c *countingGit) Diff() git.DiffStat { c.diffCalls.Add(1); return c.fakeGit.Diff() }
func (c *countingGit) Worktree() string   { c.worktreeCalls.Add(1); return c.fakeGit.Worktree() }

func TestPrefetchGit(t *testing.T) {
	settings := config.DefaultSettings()
	lines := [][]config.WidgetItem{
		{{ID: "1", Type: "git-branch"}, {ID: "2", Type: "lines-added"}, {ID: "3", Type: "lines-removed"}},
		{{ID: "4", Type: "git-worktree"}, {ID: "5", Type: "model"}},
	}

	t.Run("each needed lookup runs once, unused ones never", func(t *testing.T) {
		p := &countingGit{fakeGit: fakeGit{branch: "main", diff: git.DiffStat{Added: 3, Removed: 1}, worktree: "wt"}}
		ctx := RenderContext{Data: &status.Session{}, Git: NewGitCacheWithProvider(p)}

		PrefetchGit(lines, ctx)
		var out []string
		for _, line := range lines {
			for i := range line {
				out = append(out, Get(line[i].Type).Render(&line[i], ctx, &settings))
			}
		}

		assert.Equal(t, []string{"main", "+3", "-1", "wt", ""}, out)
		assert.Equal(t, int32(1), p.branchCalls.Load())
		assert.Equal(t, int32(1), p.diffCalls.Load())
		assert.Equal(t, int32(1), p.worktreeCalls.Load())
		assert.Equal(t, int32(0), p.changesCalls.Load())
	})

	t.Run("worktree lookup skipped when the payload is authoritative", func(t *testing.T) {
		p := &countingGit{fakeGit: fakeGit{worktree: "stale"}}
		ctx := RenderContext{Data: &status.Session{Version: "2.1.302"}, Git: NewGitCacheWithProvider(p)}

		PrefetchGit(lines, ctx)
		item := config.WidgetItem{Type: "git-worktree"}
		assert.Empty(t, Get("git-worktree").Render(&item, ctx, &settings))
		assert.Equal(t, int32(0), p.worktreeCalls.Load())
	})

	t.Run("nil cache is a no-op", func(t *testing.T) {
		assert.NotPanics(t, func() { PrefetchGit(lines, RenderContext{}) })
	})
}

func TestWorktreeFromJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     *status.Session
		wantName string
		wantOK   bool
	}{
		{"nil payload", nil, "", false},
		{"old payload without a worktree defers to git", &status.Session{Version: "2.1.90"}, "", false},
		{"modern payload without a worktree means none", &status.Session{Version: "2.1.98"}, "", true},
		{"workspace.git_worktree", &status.Session{Workspace: &status.Workspace{GitWorktree: "feat"}}, "feat", true},
		{"worktree.name", &status.Session{Worktree: &status.WorktreeInfo{Name: "wt"}}, "wt", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, ok := worktreeFromJSON(tt.data)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}
