// Package main is the entry point for the ccstatus CLI.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/render"
	"github.com/moond4rk/ccstatus/internal/status"
	"github.com/moond4rk/ccstatus/internal/terminal"
	"github.com/moond4rk/ccstatus/internal/widget"
)

var version = "dev"

// versionString returns a human-readable version string.
// For tagged releases (set via ldflags), it returns the tag version.
// For dev builds (go install), it appends VCS commit and date from build info.
func versionString() string {
	if version != "dev" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	var revision, timeVal string
	var modified bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.time":
			timeVal = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if revision == "" {
		return version
	}
	// Shorten commit hash to 7 characters.
	if len(revision) > 7 {
		revision = revision[:7]
	}
	// Extract date portion from RFC3339 timestamp.
	if idx := strings.IndexByte(timeVal, 'T'); idx > 0 {
		timeVal = timeVal[:idx]
	}
	v := fmt.Sprintf("dev (%s %s)", revision, timeVal)
	if modified {
		v += " dirty"
	}
	return v
}

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
	cmd.Version = versionString()
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
		TerminalWidth: terminal.Width(),
	}

	// Buffer all lines and write atomically to avoid partial reads
	// when Claude Code reads from the pipe during rapid re-invocations.
	var buf strings.Builder
	for _, line := range settings.Lines {
		rendered := render.RenderLine(line, &settings, ctx)
		output := render.PostProcess(rendered)
		if output != "" {
			buf.WriteString(output)
			buf.WriteByte('\n')
		}
	}
	if buf.Len() > 0 {
		fmt.Print(buf.String())
	}
	return nil
}
