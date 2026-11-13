package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/claude"
	"github.com/moond4rk/ccstatus/internal/config"
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Register ccstatus in Claude Code settings",
		Long: "Register ccstatus as the status line command in Claude Code's settings.json, and as the\n" +
			"subagent row renderer with --subagent. Other settings keep their order and values; a symlinked\n" +
			"settings.json stays linked. A fresh install sets refreshInterval to 30 unless --refresh is given.\n" +
			"The command is \"ccstatus\", found through PATH; a command that already names the binary by\n" +
			"absolute path keeps that path.",
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

	res, err := claude.Install(opts)
	if err != nil {
		return err
	}
	stderr := cmd.ErrOrStderr()
	fmt.Fprintf(stderr, "ccstatus installed successfully: %s\n", res.Path)
	if res.Command == claude.Command {
		warnIfNotOnPath(stderr, res.Path)
	}
	if opts.HideVimMode != nil && *opts.HideVimMode && !showsVimMode() {
		fmt.Fprintf(stderr, "Note: Claude Code no longer shows the vim mode; add a vim-mode widget to a line in %s to show it\n",
			config.Path())
	}
	return nil
}

// warnIfNotOnPath warns when the bare command install wrote cannot be found
// through PATH: Claude Code then fails to start it and shows no status line,
// without saying why.
func warnIfNotOnPath(stderr io.Writer, settingsPath string) {
	if _, err := exec.LookPath(claude.Command); err == nil {
		return
	}
	fmt.Fprintf(stderr, "Warning: %s is not on PATH, so Claude Code cannot start it and will show no status line.\n", claude.Command)
	self, err := os.Executable()
	if err != nil {
		return
	}
	fmt.Fprintf(stderr, "Add %s to PATH, or set \"command\" to %q in %s: install keeps an absolute path.\n",
		filepath.Dir(self), self, settingsPath)
}

// showsVimMode reports whether a main status line has a vim-mode widget.
func showsVimMode() bool {
	settings, _ := config.Load()
	for _, line := range settings.Lines {
		for i := range line {
			if line[i].Type == "vim-mode" {
				return true
			}
		}
	}
	return false
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
