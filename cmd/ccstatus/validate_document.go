package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/moond4rk/ccstatus/internal/config"
)

// Keys earlier versions read, with what to do about them, so validate explains
// them rather than calling them unknown.
var (
	removedSettings = map[string]string{
		"version": "no longer used: remove it",
	}
	removedItemFields = map[string]string{
		"id":       "no longer used: remove it",
		"metadata": `replaced by "display" and "barWidth" on the item itself`,
	}
)

// documentProblems reports what only the raw settings document shows, because
// decoding ignores it: keys the schema does not define (a removed key comes with
// its replacement, a likely typo with the key it resembles), keys given twice,
// and thresholds without "gte". Problems lead with their location, in document
// order; a removed key that many items still carry is reported once. data must
// already decode as settings.
func documentProblems(data []byte) []string {
	d := documentLinter{removed: make(map[string]*removedKey)}
	fields := d.object("settings", data, reflect.TypeFor[config.Settings](), removedSettings)
	for i, line := range elements(fields["lines"]) {
		for j, item := range elements(line) {
			d.item(fmt.Sprintf("line %d widget %d", i+1, j+1), item)
		}
	}
	for j, item := range elements(fields["subagentLine"]) {
		d.item(fmt.Sprintf("subagentLine widget %d", j+1), item)
	}
	for _, r := range d.removed {
		if r.more > 0 {
			d.problems[r.index] = fmt.Sprintf("%s and %d more: %s", r.loc, r.more, r.problem)
		}
	}
	return d.problems
}

// documentLinter collects the problems found while walking a settings document.
type documentLinter struct {
	problems []string
	removed  map[string]*removedKey // by problem text
}

// removedKey tracks one removed key, reported where it first appears along with
// how many more places carry it.
type removedKey struct {
	index   int // of its entry in problems
	loc     string
	problem string
	more    int
}

func (d *documentLinter) addf(loc, format string, args ...any) {
	d.problems = append(d.problems, loc+": "+fmt.Sprintf(format, args...))
}

// item checks one widget item and its thresholds.
func (d *documentLinter) item(loc string, raw json.RawMessage) {
	fields := d.object(loc, raw, reflect.TypeFor[config.WidgetItem](), removedItemFields)
	for k, threshold := range elements(fields["thresholds"]) {
		tloc := fmt.Sprintf("%s threshold %d", loc, k+1)
		if tfields := d.object(tloc, threshold, reflect.TypeFor[config.Threshold](), nil); tfields != nil {
			if _, ok := tfields["gte"]; !ok {
				d.addf(tloc, `missing "gte"`)
			}
		}
	}
}

// object checks the keys of the JSON object in raw against the json tags of
// struct type t, and returns the object's values under those tags. It returns
// nil when raw is not an object. Keys match as encoding/json matches them,
// ignoring case.
func (d *documentLinter) object(loc string, raw json.RawMessage, t reflect.Type, removed map[string]string) map[string]json.RawMessage {
	dec := json.NewDecoder(bytes.NewReader(raw))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil
	}
	known := jsonKeys(t)
	fields := make(map[string]json.RawMessage)
	for dec.More() {
		tok, err := dec.Token()
		key, isKey := tok.(string)
		var value json.RawMessage
		if err != nil || !isKey || dec.Decode(&value) != nil {
			break
		}
		name, ok := matchKey(key, known)
		switch {
		case !ok && removed[key] != "":
			d.addRemoved(loc, fmt.Sprintf("unknown field %q (%s)", key, removed[key]))
		case !ok:
			d.addf(loc, "unknown field %q%s", key, typoHint(key, known))
		case fields[name] != nil:
			d.addf(loc, "field %q is set twice", name)
		default:
			fields[name] = value
		}
	}
	return fields
}

// addRemoved reports a removed key once, however many places still set it.
func (d *documentLinter) addRemoved(loc, problem string) {
	if r, seen := d.removed[problem]; seen {
		r.more++
		return
	}
	d.removed[problem] = &removedKey{index: len(d.problems), loc: loc, problem: problem}
	d.problems = append(d.problems, loc+": "+problem)
}

// typoHint names the known key an unknown one most resembles, if any.
func typoHint(key string, known []string) string {
	if match := nearest(key, known); match != "" {
		return fmt.Sprintf(" (did you mean %q?)", match)
	}
	return ""
}

// elements returns the elements of the JSON array in raw, or nil when raw is
// not an array.
func elements(raw json.RawMessage) []json.RawMessage {
	var out []json.RawMessage
	if json.Unmarshal(raw, &out) != nil {
		return nil
	}
	return out
}

// jsonKeys returns the JSON object keys struct type t decodes.
func jsonKeys(t reflect.Type) []string {
	keys := make([]string, 0, t.NumField())
	for field := range t.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name != "" && name != "-" {
			keys = append(keys, name)
		}
	}
	return keys
}

// matchKey returns the known key that key names, ignoring case.
func matchKey(key string, known []string) (string, bool) {
	for _, k := range known {
		if strings.EqualFold(k, key) {
			return k, true
		}
	}
	return "", false
}

// nearest returns the candidate within a typo or two of word, or "" when none
// is that close. Short words get one edit, so "id" does not turn into "gte".
func nearest(word string, candidates []string) string {
	const shortWord, maxEdits = 4, 2
	limit := maxEdits
	if len(word) <= shortWord {
		limit = 1
	}
	best, bestDist := "", limit+1
	for _, c := range candidates {
		if dist := editDistance(strings.ToLower(word), strings.ToLower(c)); dist < bestDist {
			best, bestDist = c, dist
		}
	}
	return best
}

// editDistance is the Levenshtein distance between a and b.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
