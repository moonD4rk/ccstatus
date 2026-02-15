package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

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

func runValidate(_ *cobra.Command, _ []string) error {
	settings, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid settings: %w", err)
	}

	// Check for unknown widget types.
	var warnings []string
	for _, line := range settings.Lines {
		for i := range line {
			if widget.Get(line[i].Type) == nil {
				warnings = append(warnings, fmt.Sprintf("unknown widget type: %q", line[i].Type))
			}
		}
	}

	if len(warnings) > 0 {
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
		}
	}
	fmt.Fprintln(os.Stderr, "Settings are valid")
	return nil
}
