package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/config"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate default settings.json",
		Long:  "Generate a default settings.json configuration file at the ccstatus config directory.",
		RunE:  runInit,
	}
	cmd.Flags().Bool("force", false, "Overwrite existing settings.json")
	return cmd
}

func runInit(cmd *cobra.Command, _ []string) error {
	force, _ := cmd.Flags().GetBool("force")
	path := config.Path()

	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("settings already exist at %s (use --force to overwrite)", path)
		}
	}

	settings := config.DefaultSettings()
	if err := config.Save(&settings); err != nil {
		return fmt.Errorf("saving settings: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Created %s\n", path)
	return nil
}
