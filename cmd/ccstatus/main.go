// Package main is the entry point for the ccstatus CLI.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/color"
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
		newSubagentCmd(),
		newPreviewCmd(),
	)
	return cmd
}

func runStatusLine(cmd *cobra.Command, _ []string) error {
	if awaitsTypedInput(cmd) {
		return cmd.Help()
	}
	data, err := io.ReadAll(cmd.InOrStdin())
	renderStatusLine(cmd.OutOrStdout(), cmd.ErrOrStderr(), data, err, 0)
	return nil
}

// awaitsTypedInput reports whether the command's stdin is a terminal. Claude
// Code always pipes a payload in, so a terminal means the command was run by
// hand, and reading it would hang until the user typed an end-of-file.
func awaitsTypedInput(cmd *cobra.Command) bool {
	return terminal.IsTerminal(cmd.InOrStdin())
}

// renderStatusLine writes the status line for the Claude Code payload in data,
// at width columns or, when width is 0, the detected terminal width. It never
// fails: Claude Code blanks the whole status line when the command exits
// non-zero, so problems, a panic included, are reported as a red line instead.
func renderStatusLine(stdout, stderr io.Writer, data []byte, readErr error, width int) {
	// Buffer all lines and write once to avoid partial reads when Claude Code
	// reads from the pipe during rapid re-invocations.
	var buf strings.Builder
	defer func() { _, _ = io.WriteString(stdout, buf.String()) }()
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(stderr, "ccstatus: panic: %v\n%s", r, debug.Stack())
			writeLine(&buf, panicLine(r, width))
		}
	}()

	settings, settingsErr := config.Load()
	if width <= 0 {
		width = terminal.Width(settings.TerminalWidth)
	}

	if settingsErr != nil {
		writeLine(&buf, errorLine(&settings, width, "settings.json:"+settingsErrorText(settingsErr)+" (run ccstatus validate)"))
	}
	if readErr != nil {
		writeLine(&buf, errorLine(&settings, width, "reading stdin: "+readErr.Error()))
		return
	}
	session, dropped, err := status.Parse(data)
	if err != nil {
		writeLine(&buf, errorLine(&settings, width, "invalid input: "+err.Error()))
		return
	}
	if len(dropped) > 0 {
		fmt.Fprintf(stderr, "ccstatus: ignored fields with unexpected types: %s\n", strings.Join(dropped, ", "))
	}

	ctx := widget.RenderContext{
		Data:          session,
		TerminalWidth: width,
		Git:           widget.NewGitCache(session.WorkingDir()),
		RawInput:      data,
	}
	widget.PrefetchGit(settings.Lines, ctx)
	for _, line := range settings.Lines {
		writeLine(&buf, render.PostProcess(render.RenderLine(line, &settings, ctx)))
	}
}

// settingsErrorText formats a settings load error after the "settings.json:"
// label: decode errors already lead with "line:column:".
func settingsErrorText(err error) string {
	var posErr *config.PositionError
	if errors.As(err, &posErr) {
		return posErr.Error()
	}
	return " " + err.Error()
}

// panicLine reports a recovered panic as a status line row. It is laid out with
// the default settings, since the loaded ones may be what caused the panic.
func panicLine(r any, width int) string {
	settings := config.DefaultSettings()
	if width <= 0 {
		width = terminal.Width(0)
	}
	return errorLine(&settings, width, fmt.Sprintf("internal error: %v (please report it)", r))
}

// errorLine renders msg as a red status line row, truncated to the width the
// configured flex mode leaves for the status line.
func errorLine(settings *config.Settings, width int, msg string) string {
	text := color.Apply("ccstatus: "+msg, "red", "", false, settings.ColorLevel)
	maxWidth := render.CalculateFlexWidth(width, settings.FlexMode, settings.CompactThreshold, 0)
	return render.PostProcess(render.Truncate(text, maxWidth))
}

func writeLine(buf *strings.Builder, line string) {
	if line == "" {
		return
	}
	buf.WriteString(line)
	buf.WriteByte('\n')
}
