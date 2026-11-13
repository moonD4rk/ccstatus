package schema

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/widget"
)

// decoded returns the generated schema as a generic JSON value.
func decoded(t *testing.T) map[string]any {
	t.Helper()
	out, err := Generate()
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(out, &doc))
	return doc
}

// at walks nested objects by key.
func at(t *testing.T, doc map[string]any, path ...string) map[string]any {
	t.Helper()
	for _, key := range path {
		next, ok := doc[key].(map[string]any)
		require.Truef(t, ok, "no object at %q in %v", key, path)
		doc = next
	}
	return doc
}

func stringsOf(t *testing.T, v any) []string {
	t.Helper()
	list, ok := v.([]any)
	require.True(t, ok, "not a list: %v", v)
	out := make([]string, len(list))
	for i, e := range list {
		out[i], ok = e.(string)
		require.True(t, ok, "not a string: %v", e)
	}
	return out
}

func TestCommittedSchemaIsCurrent(t *testing.T) {
	want, err := Generate()
	require.NoError(t, err)
	got, err := os.ReadFile("../../schema.json")
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got),
		"schema.json is stale: regenerate it with `go run ./cmd/ccstatus schema > schema.json`")
}

func TestGenerateIsDeterministic(t *testing.T) {
	first, err := Generate()
	require.NoError(t, err)
	for range 5 {
		again, err := Generate()
		require.NoError(t, err)
		require.Equal(t, string(first), string(again))
	}
	assert.Equal(t, byte('\n'), first[len(first)-1])
	assert.NotContains(t, string(first), `<`, "HTML escaping is off")
}

func TestEverySettingIsDocumented(t *testing.T) {
	tests := []struct {
		name   string
		typ    reflect.Type
		fields map[string]field
	}{
		{"settings", reflect.TypeFor[config.Settings](), settingsFields},
		{"widget item", reflect.TypeFor[config.WidgetItem](), itemFields},
		{"threshold", reflect.TypeFor[config.Threshold](), thresholdFields},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var keys []string
			for f := range tt.typ.Fields() {
				keys = append(keys, jsonName(f))
			}
			for _, key := range keys {
				assert.NotEmptyf(t, tt.fields[key].description, "%q has no description", key)
			}
			for key := range tt.fields {
				assert.Containsf(t, keys, key, "%q is documented but is not a field", key)
			}
		})
	}
}

func TestTopLevelShape(t *testing.T) {
	doc := decoded(t)
	assert.Equal(t, draft, doc["$schema"])
	assert.Equal(t, URL, doc["$id"])
	assert.Equal(t, false, doc["additionalProperties"])

	props := at(t, doc, "properties")
	assert.Equal(t, "full-until-compact", at(t, props, "flexMode")["default"])
	assert.Equal(t, config.FlexModes(), stringsOf(t, at(t, props, "flexMode")["enum"]))
	assert.InDelta(t, 2, at(t, props, "colorLevel")["default"], 0)
	assert.Equal(t, true, at(t, props, "hyperlinks")["default"])
	assert.NotContains(t, at(t, props, "terminalWidth"), "default", "0 means detect, not a default width")
	assert.Equal(t, colorRef, at(t, props, "overrideForegroundColor")["$ref"])
	assert.NotContains(t, at(t, props, "overrideForegroundColor"), "type", "a reference replaces the plain type")

	item := at(t, doc, "definitions", "widget")
	assert.Equal(t, []string{"type"}, stringsOf(t, item["required"]))
	assert.Equal(t, false, item["additionalProperties"])
	assert.Equal(t, "string", at(t, item, "properties", "prefix")["type"])
	assert.Equal(t, thresholdRef, at(t, item, "properties", "thresholds", "items")["$ref"])
	assert.Equal(t, []string{"gte"}, stringsOf(t, at(t, doc, "definitions", "threshold")["required"]))
}

// rowSchema returns the type enum and the per-type rules of a row definition.
func rowSchema(t *testing.T, doc map[string]any, name string) (types []string, rules []map[string]any) {
	t.Helper()
	all, ok := at(t, doc, "definitions", name)["allOf"].([]any)
	require.True(t, ok)
	require.GreaterOrEqual(t, len(all), 3)
	enum, ok := all[1].(map[string]any)
	require.True(t, ok)
	types = stringsOf(t, at(t, enum, "properties", "type")["enum"])
	assert.Len(t, stringsOf(t, at(t, enum, "properties", "type")["enumDescriptions"]), len(types))
	for _, r := range all[2:] {
		rule, ok := r.(map[string]any)
		require.True(t, ok)
		rules = append(rules, rule)
	}
	return types, rules
}

func TestWidgetTypesAreSplitByRow(t *testing.T) {
	doc := decoded(t)
	lineTypes, lineRules := rowSchema(t, doc, "lineWidget")
	subagentTypes, subagentRules := rowSchema(t, doc, "subagentWidget")

	for _, typ := range widget.Types() {
		onLine := slices.Contains(lineTypes, typ)
		onSubagent := slices.Contains(subagentTypes, typ)
		assert.Equalf(t, !widget.IsTaskWidget(typ), onLine, "%s on a main line", typ)
		assert.Equalf(t, widget.IsTaskWidget(typ) || widget.IsSharedWidget(typ), onSubagent, "%s in subagentLine", typ)
	}
	assert.True(t, slices.IsSorted(lineTypes))
	assert.True(t, slices.IsSorted(subagentTypes))

	for name, row := range map[string]struct {
		types []string
		rules []map[string]any
	}{"lineWidget": {lineTypes, lineRules}, "subagentWidget": {subagentTypes, subagentRules}} {
		var ruled []string
		for _, rule := range row.rules {
			ruled = append(ruled, stringsOf(t, at(t, rule, "if", "properties", "type")["enum"])...)
		}
		slices.Sort(ruled)
		assert.Equalf(t, row.types, ruled, "%s: every type is in exactly one rule", name)
	}
}

func TestRestrictionsFollowTheRegistry(t *testing.T) {
	props := func(typ string) map[string]any {
		p, ok := restrictions(typ)["properties"].(object)
		require.True(t, ok)
		return p
	}

	model := props("model")
	assert.Equal(t, false, model["display"])
	assert.Equal(t, false, model["barWidth"])
	assert.Equal(t, false, model["thresholds"])
	assert.Equal(t, false, model["character"])
	assert.NotContains(t, model, "rawValue", "model has a raw mode")

	limit := props("rate-limit-5h")
	assert.Equal(t, object{"enum": []string{"percent", "bar", "reset", "full"}, "default": "percent"}, limit["display"])
	assert.NotContains(t, limit, "barWidth")
	assert.NotContains(t, limit, "thresholds")

	assert.Equal(t, object{"const": false}, props("version")["rawValue"])
	assert.NotContains(t, props("separator"), "character")
	assert.NotContains(t, props("git-branch"), "character")

	assert.Equal(t, []string{"customText"}, restrictions("custom-text")["required"])
	assert.Equal(t, []string{"commandPath"}, restrictions("custom-command")["required"])
	assert.NotContains(t, restrictions("model"), "required")
	for _, option := range []string{"commandPath", "timeout", "preserveColors"} {
		assert.NotContains(t, props("custom-command"), option)
		assert.Equal(t, false, props("custom-text")[option])
	}
}
