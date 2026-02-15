package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Register ccstatus in Claude Code settings",
		Long:  "Register ccstatus as the status line command in Claude Code's settings.json.",
		RunE:  runInstall,
	}
}

func newUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove ccstatus from Claude Code settings",
		Long:  "Remove the ccstatus status line configuration from Claude Code's settings.json.",
		RunE:  runUninstall,
	}
}

func runInstall(_ *cobra.Command, _ []string) error {
	fmt.Fprintln(os.Stderr, "Install not yet implemented")
	return nil
}

func runUninstall(_ *cobra.Command, _ []string) error {
	fmt.Fprintln(os.Stderr, "Uninstall not yet implemented")
	return nil
}
