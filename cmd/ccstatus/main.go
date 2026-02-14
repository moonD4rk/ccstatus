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
		Use:           "ccstatus",
		Short:         "Customizable status line for Claude Code",
		Long:          "A Go implementation of a customizable status line formatter for Claude Code CLI.",
		RunE:          runRoot,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.Flags().Bool("init", false, "Generate default settings.json")
	cmd.Flags().Bool("validate", false, "Validate settings.json")
	cmd.Flags().Bool("install", false, "Register in Claude Code settings.json")
	cmd.Flags().Bool("uninstall", false, "Remove from Claude Code settings.json")
	cmd.Version = version
	return cmd
}

func runRoot(cmd *cobra.Command, _ []string) error {
	if v, _ := cmd.Flags().GetBool("init"); v {
		return runInit()
	}
	if v, _ := cmd.Flags().GetBool("validate"); v {
		return runValidate()
	}
	if v, _ := cmd.Flags().GetBool("install"); v {
		return runInstall()
	}
	if v, _ := cmd.Flags().GetBool("uninstall"); v {
		return runUninstall()
	}
	return runStatusLine()
}

func runStatusLine() error {
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

func runInit() error {
	settings := config.DefaultSettings()
	if err := config.Save(&settings); err != nil {
		return fmt.Errorf("saving settings: %w", err)
	}
	fmt.Fprintf(os.Stderr, "Created %s\n", config.ConfigPath())
	return nil
}

func runValidate() error {
	_, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid settings: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Settings are valid")
	return nil
}

func runInstall() error {
	fmt.Fprintln(os.Stderr, "Install not yet implemented")
	return nil
}

func runUninstall() error {
	fmt.Fprintln(os.Stderr, "Uninstall not yet implemented")
	return nil
}
