package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/claude"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Register ccstatus in Claude Code settings",
		Long:  "Register ccstatus as the status line command in Claude Code's settings.json.",
		RunE:  runInstall,
	}
	cmd.Flags().Int("refresh", 0, "Re-run the status line every N seconds (writes refreshInterval; 0 = unset)")
	cmd.Flags().Bool("hide-vim-indicator", false, "Suppress Claude Code's built-in vim mode indicator (writes hideVimModeIndicator)")
	return cmd
}

func newUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove ccstatus from Claude Code settings",
		Long:  "Remove the ccstatus status line configuration from Claude Code's settings.json.",
		RunE:  runUninstall,
	}
}

func runInstall(cmd *cobra.Command, _ []string) error {
	refresh, _ := cmd.Flags().GetInt("refresh")
	hideVim, _ := cmd.Flags().GetBool("hide-vim-indicator")
	path, err := claude.Install(refresh, hideVim)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "ccstatus installed successfully: %s\n", path)
	return nil
}

func runUninstall(_ *cobra.Command, _ []string) error {
	path, err := claude.Uninstall()
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "ccstatus uninstalled successfully: %s\n", path)
	return nil
}
