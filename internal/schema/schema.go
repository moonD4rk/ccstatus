// Package schema generates the JSON Schema of settings.json from the settings
// types and the widget registry, so an editor can complete and check a
// settings file as it is written.
package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
	"github.com/moond4rk/ccstatus/internal/widget"
)

const (
	// URL is where the generated schema is published. `ccstatus init` writes it
	// as the "$schema" of a new settings file.
	URL = "https://raw.githubusercontent.com/moonD4rk/ccstatus/main/schema.json"

	// draft is the JSON Schema dialect: draft-07 is the newest one every common
	// editor supports.
	draft = "http://json-schema.org/draft-07/schema#"

	colorRef     = "#/definitions/color"
	thresholdRef = "#/definitions/threshold"
	itemRef      = "#/definitions/widget"
)

// object is a JSON object under construction.
type object = map[string]any

// Generate returns the JSON Schema of settings.json, indented, with a trailing
// newline.
func Generate() ([]byte, error) {
	doc := object{
		"$schema":              draft,
		"$id":                  URL,
		"title":                "ccstatus settings",
		"description":          "Settings of ccstatus, the status line formatter for Claude Code. Fields left out keep their defaults.",
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties(reflect.TypeFor[config.Settings](), settingsFields, defaults()),
		"definitions": object{
			"color":          colorSchema(),
			"threshold":      structSchema(reflect.TypeFor[config.Threshold](), thresholdFields, "gte"),
			"widget":         structSchema(reflect.TypeFor[config.WidgetItem](), itemFields, "type"),
			"lineWidget":     widgetSchema(lineTypes()),
			"subagentWidget": widgetSchema(subagentTypes()),
		},
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// field documents one settings key: its description and the constraints that
// its Go type does not carry.
type field struct {
	description string
	extra       object
}

var settingsFields = map[string]field{
	"$schema": {description: "The JSON Schema this file follows, which lets an editor complete and check it."},
	"lines": {
		description: "The rows of the status line. Each row is a list of widget items, rendered left to right.",
		extra:       object{"items": object{"type": "array", "items": object{"$ref": "#/definitions/lineWidget"}}},
	},
	"flexMode": {
		description: "The right margin each row leaves: where the row is cut with \"...\" and how far a flex-separator fills.",
		extra: object{
			"enum": config.FlexModes(),
			"enumDescriptions": []string{
				"The terminal width minus 10 columns.",
				"The terminal width minus 40 columns.",
				"Minus 10 columns, then minus 40 once the context percentage reaches compactThreshold.",
				"The terminal width, with no margin: for Claude Code's fullscreen rendering, where notifications get a row of their own.",
			},
		},
	},
	"compactThreshold": {
		description: "The context percentage at which full-until-compact switches to the 40-column margin.",
		extra:       object{"minimum": 0, "maximum": 100}, //nolint:mnd // a percentage
	},
	"colorLevel": {
		description: "0 disables color. 1 emits the 16 ANSI colors, 2 the 256-color palette, 3 true color; " +
			"a color the level cannot express is mapped to the nearest one it can.",
		extra: object{"enum": []int{color.LevelNone, color.Level16, color.Level256, color.LevelTrueColor}},
	},
	"hyperlinks": {description: "Make the pr number and the repo name clickable OSC 8 links."},
	"terminalWidth": {
		description: "Override the detected terminal width; the flexMode margin still applies. 0 detects it.",
		extra:       object{"minimum": 0},
	},
	"defaultSeparator":        {description: "The text a separator shows unless it sets character."},
	"defaultPadding":          {description: "Inserted between adjacent widgets, separators included, except after an item with merge."},
	"inheritSeparatorColors":  {description: "A separator without its own color takes the color of the widget before it."},
	"overrideBackgroundColor": {description: "Force one background color on every widget and separator.", extra: object{"$ref": colorRef}},
	"overrideForegroundColor": {description: "Force one foreground color on every widget and separator.", extra: object{"$ref": colorRef}},
	"globalBold":              {description: "Apply bold to all widgets."},
	"subagentLine": {
		description: "The row rendered for each subagent in Claude Code's agent panel. It takes the task-* widgets, " +
			"separator, flex-separator and custom-text.",
		extra: object{"items": object{"$ref": "#/definitions/subagentWidget"}},
	},
	"subagentHideStatuses": {
		description: "Task statuses whose subagent rows are hidden.",
		extra:       object{"items": object{"enum": status.TaskStatuses()}},
	},
}

var itemFields = map[string]field{
	"type":            {description: "The widget to render."},
	"color":           {description: "Foreground color, in place of the widget's default.", extra: object{"$ref": colorRef}},
	"backgroundColor": {description: "Background color.", extra: object{"$ref": colorRef}},
	"bold":            {description: "Bold text."},
	"prefix": {
		description: "Text before the widget's output, in place of the widget's default prefix such as \"Ctx: \". " +
			"An empty string removes the default.",
	},
	"suffix":    {description: "Text after the widget's output."},
	"character": {description: "The text of a separator, or an icon before the git-branch name."},
	"rawValue":  {description: "Use the widget's raw mode, on widgets that have one."},
	"display":   {description: "One of the widget's display modes."},
	"barWidth": {
		description: "The width of a bar display, in columns.",
		extra:       object{"minimum": 1, "maximum": widget.MaxBarWidth, "default": widget.DefaultBarWidth},
	},
	"maxWidth": {
		description: "Cap the widget's text at this many columns, ending it with an ellipsis. " +
			"The prefix and suffix are not counted.",
		extra: object{"minimum": 0},
	},
	"merge":      {description: "Join this widget to the next one without defaultPadding."},
	"customText": {description: "The text of a custom-text widget."},
	"commandPath": {
		description: "The shell command of a custom-command widget, run with sh -c and given Claude Code's JSON on stdin. " +
			"The first line of its output is shown.",
	},
	"timeout": {
		description: "The custom-command timeout in milliseconds. When it expires, the command and everything it started are killed.",
		extra:       object{"minimum": 0, "default": widget.DefaultCommandTimeout.Milliseconds()},
	},
	"preserveColors": {description: "Keep the ANSI colors in custom-command output."},
	"thresholds": {
		description: "Restyle a numeric widget by its value. The highest gte reached wins; below the first, the item's own colors apply.",
		extra:       object{"items": object{"$ref": thresholdRef}},
	},
}

var thresholdFields = map[string]field{
	"gte":             {description: "The value from which this threshold applies."},
	"color":           {description: "Foreground color from this value on.", extra: object{"$ref": colorRef}},
	"backgroundColor": {description: "Background color from this value on.", extra: object{"$ref": colorRef}},
	"bold":            {description: "Bold text from this value on."},
}

// defaults returns the default of every scalar top-level setting, by key.
func defaults() map[string]any {
	settings := config.DefaultSettings()
	out := make(map[string]any)
	value := reflect.ValueOf(settings)
	for f, v := range value.Fields() {
		switch v.Kind() {
		case reflect.String, reflect.Bool, reflect.Int:
			if !v.IsZero() || v.Kind() == reflect.Bool {
				out[jsonName(f)] = v.Interface()
			}
		default:
		}
	}
	return out
}

// structSchema returns the schema of the JSON object that struct type t
// decodes, with the given keys required.
func structSchema(t reflect.Type, fields map[string]field, required ...string) object {
	return object{
		"type":                 "object",
		"additionalProperties": false,
		"required":             required,
		"properties":           properties(t, fields, nil),
	}
}

// properties returns the schema of each JSON key struct type t decodes: its
// type from the Go field, then the documented constraints and default.
func properties(t reflect.Type, fields map[string]field, defaults map[string]any) object {
	props := make(object)
	for f := range t.Fields() {
		name := jsonName(f)
		if name == "" {
			continue
		}
		doc := fields[name]
		prop := typeSchema(f.Type)
		if _, isRef := doc.extra["$ref"]; isRef {
			prop = object{} // a reference replaces the plain type
		}
		for key, value := range doc.extra {
			prop[key] = value
		}
		prop["description"] = doc.description
		if def, ok := defaults[name]; ok {
			prop["default"] = def
		}
		props[name] = prop
	}
	return props
}

// typeSchema maps a Go type to the JSON type it decodes from.
func typeSchema(t reflect.Type) object {
	switch t.Kind() {
	case reflect.Pointer:
		return typeSchema(t.Elem())
	case reflect.String:
		return object{"type": "string"}
	case reflect.Bool:
		return object{"type": "boolean"}
	case reflect.Int:
		return object{"type": "integer"}
	case reflect.Float64:
		return object{"type": "number"}
	case reflect.Slice:
		return object{"type": "array", "items": typeSchema(t.Elem())}
	case reflect.Struct:
		return object{"$ref": itemRef}
	default:
		panic(fmt.Sprintf("schema: no JSON type for %s", t))
	}
}

// jsonName returns the JSON key a struct field decodes from.
func jsonName(f reflect.StructField) string { //nolint:gocritic // reflect.StructField is the iterator's element type
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	if name == "-" {
		return ""
	}
	return name
}

// colorSchema describes the three ways to write a color.
func colorSchema() object {
	return object{
		"description": "A color: one of the 16 ANSI color names, a hex triplet such as \"#ff8700\", " +
			"or a 256-color palette index \"0\" to \"255\".",
		"anyOf": []object{
			{"enum": color.Names()},
			{"type": "string", "pattern": color.HexPattern},
			{"type": "string", "pattern": color.PalettePattern},
		},
	}
}

// lineTypes returns, sorted, the widget types a main status line takes.
func lineTypes() []string {
	types := slices.DeleteFunc(widget.Types(), widget.IsTaskWidget)
	slices.Sort(types)
	return types
}

// subagentTypes returns, sorted, the widget types subagentLine takes.
func subagentTypes() []string {
	types := slices.DeleteFunc(widget.Types(), func(t string) bool {
		return !widget.IsTaskWidget(t) && !widget.IsSharedWidget(t)
	})
	slices.Sort(types)
	return types
}

// widgetSchema returns the schema of an item on a row that takes the given
// widget types: the shared item shape, the types with what each one shows, and
// per type the options that have no effect on it.
func widgetSchema(types []string) object {
	descriptions := make([]string, len(types))
	for i, t := range types {
		descriptions[i] = describe(t)
	}
	all := []object{
		{"$ref": itemRef},
		{"properties": object{"type": object{"enum": types, "enumDescriptions": descriptions}}},
	}
	return object{"allOf": append(all, typeRules(types)...)}
}

// describe returns what a widget type shows, with its default color and prefix.
func describe(widgetType string) string {
	w := widget.Get(widgetType)
	out := w.Description() + "."
	if c := w.DefaultColor(); c != "" {
		out += " Default color: " + c + "."
	}
	if p := widget.DefaultPrefix(w); p != "" {
		out += fmt.Sprintf(" Default prefix: %q.", p)
	}
	return out
}

// typeRules returns one if/then rule per group of widget types that share the
// same restrictions, ordered by each group's first type.
func typeRules(types []string) []object {
	type group struct {
		types []string
		then  object
	}
	var groups []*group
	byRule := make(map[string]*group)
	for _, t := range types {
		then := restrictions(t)
		key, err := json.Marshal(then)
		if err != nil {
			panic(err) // then holds only strings, booleans and slices of strings
		}
		g, ok := byRule[string(key)]
		if !ok {
			g = &group{then: then}
			byRule[string(key)] = g
			groups = append(groups, g)
		}
		g.types = append(g.types, t)
	}

	rules := make([]object, len(groups))
	for i, g := range groups {
		rules[i] = object{
			"if":   object{"required": []string{"type"}, "properties": object{"type": object{"enum": g.types}}},
			"then": g.then,
		}
	}
	return rules
}

// restrictions returns what a widget type narrows in the shared item shape:
// its display modes, the option it needs, and the options that do nothing on
// it, which are not allowed.
func restrictions(widgetType string) object {
	props := make(object)
	if modes, ok := widget.DisplayModes(widgetType); ok {
		props["display"] = object{"enum": modes, "default": modes[0]}
	} else {
		props["display"] = false
	}
	if !widget.SupportsRawValue(widgetType) {
		props["rawValue"] = object{"const": false}
	}
	if !widget.HasBarDisplay(widgetType) {
		props["barWidth"] = false
	}
	if !widget.TakesThresholds(widgetType) {
		props["thresholds"] = false
	}
	for _, option := range widget.SpecificOptions() {
		if !slices.Contains(widget.OptionUsers(option), widgetType) {
			props[option] = false
		}
	}
	then := object{"properties": props}
	if required := widget.RequiredOption(widgetType); required != "" {
		then["required"] = []string{required}
	}
	return then
}
