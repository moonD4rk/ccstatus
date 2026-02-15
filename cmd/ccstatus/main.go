// Package main is the entry point for the ccstatus CLI.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/render"
	"github.com/moond4rk/ccstatus/internal/status"
	"github.com/moond4rk/ccstatus/internal/terminal"
	"github.com/moond4rk/ccstatus/internal/widget"
)

var version = "dev"

func main() {
	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ccstatus",
		Short: "Customizable status line for Claude Code",
		Long: "A customizable status line formatter for Claude Code CLI.\n\n" +
			"When run without a subcommand, reads JSON from stdin and renders the status line.",
		RunE:          runStatusLine,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.Version = version
	cmd.AddCommand(
		newInitCmd(),
		newValidateCmd(),
		newInstallCmd(),
		newUninstallCmd(),
		newDumpCmd(),
		newWidgetsCmd(),
	)
	return cmd
}

func runStatusLine(_ *cobra.Command, _ []string) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	statusData, err := status.Parse(data)
	if err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	settings, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading settings: %w", err)
	}

	ctx := widget.RenderContext{
		Data:          statusData,
		TerminalWidth: terminal.GetWidth(),
	}

	for _, line := range settings.Lines {
		rendered := render.RenderLine(line, &settings, ctx)
		output := render.PostProcess(rendered)
		if output != "" {
			fmt.Println(output)
		}
	}
	return nil
}
