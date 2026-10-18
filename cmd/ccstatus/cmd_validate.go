package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/widget"
)

const (
	maxColorLevel       = 3
	maxCompactThreshold = 100
)

var flexModes = []string{"full", "full-minus-40", "full-until-compact"}

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate settings.json",
		Long:  "Validate the ccstatus settings.json configuration file for correctness.",
		RunE:  runValidate,
	}
}

// runValidate checks the loaded settings for decode errors (reported as
// path:line:column), unknown widget types, missing or duplicate widget IDs,
// invalid colors, out-of-range globals, unsupported display modes and
// thresholds, and widgets used on the wrong kind of line. It
// reports every problem and exits non-zero when any are found, so it is usable
// as a CI gate.
func runValidate(cmd *cobra.Command, _ []string) error {
	settings, err := config.Load()
	if err != nil {
		var posErr *config.PositionError
		if errors.As(err, &posErr) {
			return fmt.Errorf("%s:%w", config.Path(), err)
		}
		return fmt.Errorf("reading settings: %w", err)
	}

	problems := collectProblems(&settings)

	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintln(cmd.ErrOrStderr(), "Error:", p)
		}
		return fmt.Errorf("%d problem(s) found in %s", len(problems), config.Path())
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Settings are valid")
	return nil
}

// collectProblems returns a human-readable list of configuration problems, in
// document order so the output is stable.
func collectProblems(settings *config.Settings) []string {
	var problems []string
	seen := make(map[string]bool)
	for li, line := range settings.Lines {
		for i := range line {
			loc := fmt.Sprintf("line %d widget %d", li+1, i+1)
			problems = append(problems, itemProblems(loc, &line[i], seen)...)
			if widget.IsTaskWidget(line[i].Type) {
				problems = append(problems, fmt.Sprintf("%s: %s only works in subagentLine", loc, line[i].Type))
			}
		}
	}
	for i := range settings.SubagentLine {
		item := &settings.SubagentLine[i]
		loc := fmt.Sprintf("subagentLine widget %d", i+1)
		problems = append(problems, itemProblems(loc, item, seen)...)
		if widget.Get(item.Type) != nil && !widget.IsTaskWidget(item.Type) && !widget.IsSharedWidget(item.Type) {
			problems = append(problems, fmt.Sprintf("%s: %s only works on the main status line", loc, item.Type))
		}
	}
	return append(problems, globalProblems(settings)...)
}

// itemProblems checks one widget item; seen tracks the IDs used so far.
func itemProblems(loc string, item *config.WidgetItem, seen map[string]bool) []string {
	var problems []string
	known := false
	switch {
	case item.Type == "":
		problems = append(problems, loc+": missing widget type")
	case widget.Get(item.Type) == nil:
		problems = append(problems, fmt.Sprintf("%s: unknown widget type %q", loc, item.Type))
	default:
		known = true
	}

	switch {
	case item.ID == "":
		problems = append(problems, loc+": missing id")
	case seen[item.ID]:
		problems = append(problems, fmt.Sprintf("%s: duplicate id %q", loc, item.ID))
	default:
		seen[item.ID] = true
	}

	problems = append(problems, colorProblem(loc, "color", item.Color)...)
	problems = append(problems, colorProblem(loc, "backgroundColor", item.BackgroundColor)...)
	if known {
		problems = append(problems, thresholdProblems(loc, item)...)
	}

	if display, ok := item.Metadata["display"]; ok && known {
		modes, hasModes := widget.DisplayModes(item.Type)
		switch {
		case !hasModes:
			problems = append(problems, fmt.Sprintf("%s: %s does not use metadata.display", loc, item.Type))
		case !slices.Contains(modes, display):
			problems = append(problems, fmt.Sprintf("%s: invalid metadata.display %q (want one of %s)",
				loc, display, strings.Join(modes, ", ")))
		}
	}
	return problems
}

// thresholdProblems checks that thresholds sit on a numeric widget and name
// valid colors.
func thresholdProblems(loc string, item *config.WidgetItem) []string {
	if len(item.Thresholds) == 0 {
		return nil
	}
	if _, ok := widget.Get(item.Type).(widget.Valuer); !ok {
		return []string{fmt.Sprintf("%s: %s does not support thresholds", loc, item.Type)}
	}
	var problems []string
	for i, t := range item.Thresholds {
		tloc := fmt.Sprintf("%s threshold %d", loc, i+1)
		problems = append(problems, colorProblem(tloc, "color", t.Color)...)
		problems = append(problems, colorProblem(tloc, "backgroundColor", t.BackgroundColor)...)
	}
	return problems
}

// globalProblems checks the top-level settings.
func globalProblems(settings *config.Settings) []string {
	problems := colorProblem("settings", "overrideForegroundColor", settings.OverrideForegroundColor)
	problems = append(problems, colorProblem("settings", "overrideBackgroundColor", settings.OverrideBackgroundColor)...)
	if !slices.Contains(flexModes, settings.FlexMode) {
		problems = append(problems, fmt.Sprintf("settings: invalid flexMode %q (want one of %s)",
			settings.FlexMode, strings.Join(flexModes, ", ")))
	}
	if settings.ColorLevel < 0 || settings.ColorLevel > maxColorLevel {
		problems = append(problems, fmt.Sprintf("settings: colorLevel %d out of range 0-%d", settings.ColorLevel, maxColorLevel))
	}
	if settings.CompactThreshold < 0 || settings.CompactThreshold > maxCompactThreshold {
		problems = append(problems, fmt.Sprintf("settings: compactThreshold %d out of range 0-%d",
			settings.CompactThreshold, maxCompactThreshold))
	}
	return problems
}

// colorProblem reports a non-empty value that is not a named color.
func colorProblem(loc, kind, val string) []string {
	if val != "" && !color.IsNamed(val) {
		return []string{fmt.Sprintf("%s: invalid %s %q", loc, kind, val)}
	}
	return nil
}
