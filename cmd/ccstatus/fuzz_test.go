package main

import (
	"strings"
	"testing"

	"github.com/moond4rk/ccstatus/internal/config"
)

// FuzzValidate checks that neither half of validate panics on any document
// that decodes as settings, and that every problem it reports is located.
func FuzzValidate(f *testing.F) {
	f.Add([]byte(`{"version":4,"theme":"dark","lines":[[{"id":"1","type":"model","colr":"red","color":"a","color":"b"}]]}`))
	f.Add([]byte(`{"lines":[[{"type":"context-percentage","thresholds":[{"gt":70},{"gte":90,"color":"#f00"}]}]],"subagentLine":[{"type":"git-branch"}]}`))
	f.Add([]byte(`{"lines":[[{"type":"rate-limit-5h","display":"bars","barWidth":500,"rawValue":true,"character":"x"}]]}`))
	f.Add([]byte(`{"Lines":[[{"TYPE":"custom-text"}]],"flexMode":"none","colorLevel":9}`))
	f.Add([]byte(`{"lines":null,"subagentLine":null}`))
	f.Add([]byte(`[]`))

	f.Fuzz(func(t *testing.T, data []byte) {
		settings, err := config.Parse(data)
		if err != nil {
			return
		}
		for _, problem := range append(documentProblems(data), collectProblems(&settings)...) {
			if loc, _, ok := strings.Cut(problem, ": "); !ok || loc == "" {
				t.Fatalf("problem without a location: %q", problem)
			}
		}
	})
}
