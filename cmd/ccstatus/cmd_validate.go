package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/widget"
)

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate settings.json",
		Long:  "Validate the ccstatus settings.json configuration file for correctness.",
		RunE:  runValidate,
	}
}

// runValidate checks the loaded settings for unknown widget types, missing or
// duplicate widget IDs, and invalid color names. It reports every problem and
// exits non-zero when any are found, so it is usable as a CI gate.
func runValidate(cmd *cobra.Command, _ []string) error {
	settings, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid settings: %w", err)
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

	checkColor := func(loc, kind, val string) {
		if val != "" && !color.IsNamed(val) {
			problems = append(problems, fmt.Sprintf("%s: invalid %s %q", loc, kind, val))
		}
	}

	for li, line := range settings.Lines {
		for i := range line {
			item := &line[i]
			loc := fmt.Sprintf("line %d widget %d", li+1, i+1)

			switch {
			case item.Type == "":
				problems = append(problems, loc+": missing widget type")
			case widget.Get(item.Type) == nil:
				problems = append(problems, fmt.Sprintf("%s: unknown widget type %q", loc, item.Type))
			}

			switch {
			case item.ID == "":
				problems = append(problems, loc+": missing id")
			case seen[item.ID]:
				problems = append(problems, fmt.Sprintf("%s: duplicate id %q", loc, item.ID))
			default:
				seen[item.ID] = true
			}

			checkColor(loc, "color", item.Color)
			checkColor(loc, "backgroundColor", item.BackgroundColor)
		}
	}

	checkColor("settings", "overrideForegroundColor", settings.OverrideForegroundColor)
	checkColor("settings", "overrideBackgroundColor", settings.OverrideBackgroundColor)

	return problems
}
