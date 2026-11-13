package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/status"
	"github.com/moond4rk/ccstatus/internal/widget"
)

const (
	maxColorLevel       = 3
	maxCompactThreshold = 100
)

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate settings.json",
		Long: "Check the ccstatus settings.json strictly. Rendering skips what it does not understand;\n" +
			"validate reports it: unknown or misspelled keys, removed keys with their replacement, unknown\n" +
			"widget types and colors, options that have no effect on their widget, and out-of-range values.",
		RunE: runValidate,
	}
}

// runValidate reports every problem in the settings file and exits non-zero
// when there is one, so it is usable as a CI gate. A decode error is reported
// alone, as path:line:column, since nothing after it can be trusted.
func runValidate(cmd *cobra.Command, _ []string) error {
	path := config.Path()
	if path == "" {
		return config.ErrNoConfigDir
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(cmd.OutOrStdout(), "No settings file at %s: the default settings are in use\n", path)
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading settings: %w", err)
	}
	settings, err := config.Parse(data)
	if err != nil {
		var posErr *config.PositionError
		if errors.As(err, &posErr) {
			return fmt.Errorf("%s:%w", path, err)
		}
		return fmt.Errorf("%s: %w", path, err)
	}

	problems := append(documentProblems(data), collectProblems(&settings)...)
	if len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintln(cmd.ErrOrStderr(), "Error:", p)
		}
		return fmt.Errorf("%d problem(s) found in %s", len(problems), path)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Settings are valid: %s\n", path)
	return nil
}

// collectProblems returns a human-readable list of problems in the decoded
// settings, in document order so the output is stable.
func collectProblems(settings *config.Settings) []string {
	var problems []string
	for li, line := range settings.Lines {
		for i := range line {
			loc := fmt.Sprintf("line %d widget %d", li+1, i+1)
			problems = append(problems, itemProblems(loc, &line[i])...)
			if widget.IsTaskWidget(line[i].Type) {
				problems = append(problems, fmt.Sprintf("%s: %s only works in subagentLine", loc, line[i].Type))
			}
		}
	}
	for i := range settings.SubagentLine {
		item := &settings.SubagentLine[i]
		loc := fmt.Sprintf("subagentLine widget %d", i+1)
		problems = append(problems, itemProblems(loc, item)...)
		if widget.Get(item.Type) != nil && !widget.IsTaskWidget(item.Type) && !widget.IsSharedWidget(item.Type) {
			problems = append(problems, fmt.Sprintf("%s: %s only works on the main status line", loc, item.Type))
		}
	}
	return append(problems, globalProblems(settings)...)
}

// itemProblems checks one widget item.
func itemProblems(loc string, item *config.WidgetItem) []string {
	problems := colorProblem(loc, "color", item.Color)
	problems = append(problems, colorProblem(loc, "backgroundColor", item.BackgroundColor)...)
	if item.MaxWidth < 0 {
		problems = append(problems, loc+": maxWidth must not be negative")
	}
	if item.Timeout < 0 {
		problems = append(problems, loc+": timeout must not be negative")
	}

	switch {
	case item.Type == "":
		return append(problems, loc+": missing widget type")
	case widget.Get(item.Type) == nil:
		hint := ""
		if match := nearest(item.Type, widget.Types()); match != "" {
			hint = fmt.Sprintf(" (did you mean %q?)", match)
		}
		return append(problems, fmt.Sprintf("%s: unknown widget type %q%s", loc, item.Type, hint))
	}

	problems = append(problems, optionProblems(loc, item)...)
	problems = append(problems, displayProblems(loc, item)...)
	return append(problems, thresholdProblems(loc, item)...)
}

// optionProblems reports options that do nothing on the item's widget type, and
// widgets missing the option they need.
func optionProblems(loc string, item *config.WidgetItem) []string {
	var problems []string
	if item.RawValue && !widget.SupportsRawValue(item.Type) {
		problems = append(problems, fmt.Sprintf("%s: rawValue has no effect on %s", loc, item.Type))
	}
	options := []struct {
		name string
		set  bool
	}{
		{"character", item.Character != ""},
		{"customText", item.CustomText != ""},
		{"commandPath", item.CommandPath != ""},
		{"timeout", item.Timeout != 0},
		{"preserveColors", item.PreserveColors},
	}
	for _, opt := range options {
		users := widget.OptionUsers(opt.name)
		switch {
		case opt.set && !slices.Contains(users, item.Type):
			problems = append(problems, fmt.Sprintf("%s: %s has no effect on %s (it applies to %s)",
				loc, opt.name, item.Type, strings.Join(users, ", ")))
		case !opt.set && requiredOption(item.Type) == opt.name:
			problems = append(problems, fmt.Sprintf("%s: %s needs %s", loc, item.Type, opt.name))
		}
	}
	return problems
}

// requiredOption returns the option a widget type renders nothing without.
func requiredOption(widgetType string) string {
	switch widgetType {
	case widget.TypeCustomText:
		return "customText"
	case widget.TypeCustomCommand:
		return "commandPath"
	default:
		return ""
	}
}

// displayProblems checks display against the widget's modes and barWidth
// against the bar display it sizes.
func displayProblems(loc string, item *config.WidgetItem) []string {
	var problems []string
	modes, hasModes := widget.DisplayModes(item.Type)
	switch {
	case item.Display == "":
	case !hasModes:
		problems = append(problems, fmt.Sprintf("%s: %s has no display modes", loc, item.Type))
	case !slices.Contains(modes, item.Display):
		problems = append(problems, fmt.Sprintf("%s: invalid display %q (want one of %s)",
			loc, item.Display, strings.Join(modes, ", ")))
	}

	switch {
	case item.BarWidth == 0:
	case item.BarWidth < 0 || item.BarWidth > widget.MaxBarWidth:
		problems = append(problems, fmt.Sprintf("%s: barWidth %d out of range 1-%d", loc, item.BarWidth, widget.MaxBarWidth))
	case !widget.HasBarDisplay(item.Type):
		problems = append(problems, fmt.Sprintf("%s: barWidth has no effect on %s", loc, item.Type))
	case item.Display != widget.BarDisplay:
		problems = append(problems, fmt.Sprintf("%s: barWidth has no effect unless display is %q", loc, widget.BarDisplay))
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
	if modes := config.FlexModes(); !slices.Contains(modes, settings.FlexMode) {
		problems = append(problems, fmt.Sprintf("settings: invalid flexMode %q (want one of %s)",
			settings.FlexMode, strings.Join(modes, ", ")))
	}
	if settings.ColorLevel < 0 || settings.ColorLevel > maxColorLevel {
		problems = append(problems, fmt.Sprintf("settings: colorLevel %d out of range 0-%d", settings.ColorLevel, maxColorLevel))
	}
	if settings.CompactThreshold < 0 || settings.CompactThreshold > maxCompactThreshold {
		problems = append(problems, fmt.Sprintf("settings: compactThreshold %d out of range 0-%d",
			settings.CompactThreshold, maxCompactThreshold))
	}
	if settings.TerminalWidth < 0 {
		problems = append(problems, "settings: terminalWidth must not be negative")
	}
	known := status.TaskStatuses()
	for _, s := range settings.SubagentHideStatuses {
		if !slices.Contains(known, s) {
			problems = append(problems, fmt.Sprintf("settings: unknown subagentHideStatuses entry %q (want one of %s)",
				s, strings.Join(known, ", ")))
		}
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
