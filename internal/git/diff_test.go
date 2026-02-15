package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseShortStat(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected DiffStat
	}{
		{
			name:     "both insertions and deletions",
			input:    " 3 files changed, 10 insertions(+), 5 deletions(-)",
			expected: DiffStat{Added: 10, Removed: 5},
		},
		{
			name:     "insertions only",
			input:    " 1 file changed, 42 insertions(+)",
			expected: DiffStat{Added: 42, Removed: 0},
		},
		{
			name:     "deletions only",
			input:    " 2 files changed, 7 deletions(-)",
			expected: DiffStat{Added: 0, Removed: 7},
		},
		{
			name:     "single insertion",
			input:    " 1 file changed, 1 insertion(+)",
			expected: DiffStat{Added: 1, Removed: 0},
		},
		{
			name:     "single deletion",
			input:    " 1 file changed, 1 deletion(-)",
			expected: DiffStat{Added: 0, Removed: 1},
		},
		{
			name:     "empty string",
			input:    "",
			expected: DiffStat{},
		},
		{
			name:     "large numbers",
			input:    " 50 files changed, 5430 insertions(+), 423 deletions(-)",
			expected: DiffStat{Added: 5430, Removed: 423},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseShortStat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
