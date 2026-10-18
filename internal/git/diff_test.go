package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNumStat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected DiffStat
	}{
		{
			name:     "several files",
			input:    "10\t5\tmain.go\n3\t0\tREADME.md\n0\t2\tgo.sum",
			expected: DiffStat{Added: 13, Removed: 7},
		},
		{
			name:     "single file",
			input:    "42\t0\tdocs/guide.md",
			expected: DiffStat{Added: 42},
		},
		{
			name:     "binary files count as zero",
			input:    "-\t-\tdocs/terminal-screenshot.png\n1\t1\tmain.go",
			expected: DiffStat{Added: 1, Removed: 1},
		},
		{
			name:     "paths with tabs and renames",
			input:    "2\t1\told name.go => new\tname.go",
			expected: DiffStat{Added: 2, Removed: 1},
		},
		{
			name:     "empty output",
			input:    "",
			expected: DiffStat{},
		},
		{
			name:     "malformed lines are skipped",
			input:    "garbage\n5\t1\tok.go\n",
			expected: DiffStat{Added: 5, Removed: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, parseNumStat(tt.input))
		})
	}
}
