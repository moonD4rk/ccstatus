package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/claude"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Register ccstatus in Claude Code settings",
		Long: "Register ccstatus as the status line command in Claude Code's settings.json, and as the\n" +
			"subagent row renderer with --subagent. Other settings keep their order and values; a symlinked\n" +
			"settings.json stays linked. A fresh install sets refreshInterval to 30 unless --refresh is given.",
		RunE: runInstall,
	}
	cmd.Flags().Int("refresh", 0, "Re-run the status line every N seconds (writes refreshInterval; 0 = unset)")
	cmd.Flags().Int("padding", 0, "Horizontal padding added to the status line (writes padding)")
	cmd.Flags().Bool("hide-vim-indicator", false, "Suppress Claude Code's built-in vim mode indicator (writes hideVimModeIndicator)")
	cmd.Flags().Bool("subagent", false, "Also render subagent rows in the agent panel (writes subagentStatusLine; =false removes it)")
	return cmd
}

func newUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove ccstatus from Claude Code settings",
		Long: "Remove the statusLine and subagentStatusLine blocks that run ccstatus from Claude Code's\n" +
			"settings.json. Blocks that run another command are left untouched.",
		RunE: runUninstall,
	}
}

func runInstall(cmd *cobra.Command, _ []string) error {
	// Only flags the user explicitly set become overrides; unset flags preserve
	// the current statusLine values on re-install.
	var opts claude.InstallOptions
	if cmd.Flags().Changed("refresh") {
		v, _ := cmd.Flags().GetInt("refresh")
		opts.RefreshInterval = &v
	}
	if cmd.Flags().Changed("padding") {
		v, _ := cmd.Flags().GetInt("padding")
		opts.Padding = &v
	}
	if cmd.Flags().Changed("hide-vim-indicator") {
		v, _ := cmd.Flags().GetBool("hide-vim-indicator")
		opts.HideVimMode = &v
	}
	if cmd.Flags().Changed("subagent") {
		v, _ := cmd.Flags().GetBool("subagent")
		opts.Subagent = &v
	}

	path, err := claude.Install(opts)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "ccstatus installed successfully: %s\n", path)
	return nil
}

func runUninstall(cmd *cobra.Command, _ []string) error {
	res, err := claude.Uninstall()
	if err != nil {
		return err
	}
	stderr := cmd.ErrOrStderr()
	if len(res.Removed) > 0 {
		fmt.Fprintf(stderr, "ccstatus uninstalled (%s): %s\n", strings.Join(res.Removed, ", "), res.Path)
	} else {
		fmt.Fprintf(stderr, "ccstatus is not installed: %s\n", res.Path)
	}
	for _, key := range res.Kept {
		fmt.Fprintf(stderr, "Kept %s: it runs another command\n", key)
	}
	return nil
}
