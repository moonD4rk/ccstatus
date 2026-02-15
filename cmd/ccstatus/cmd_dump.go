package main

import (
	"encoding/json"
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

const defaultDumpPath = "/tmp/ccstatus-dump.json"

func newDumpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Dump raw JSON input from Claude Code for debugging",
		Long: "Read JSON from stdin, save to a file for inspection, and render the status line normally.\n" +
			"Useful for debugging what Claude Code sends to ccstatus.",
		RunE: runDump,
	}
	cmd.Flags().StringP("output", "o", defaultDumpPath, "Output file path for the JSON dump")
	return cmd
}

func runDump(cmd *cobra.Command, _ []string) error {
	output, _ := cmd.Flags().GetString("output")

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	// Pretty-print JSON to the dump file.
	if writeErr := writeDump(data, output); writeErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write dump: %v\n", writeErr)
	} else {
		fmt.Fprintf(os.Stderr, "Dumped JSON to %s\n", output)
	}

	// Still render the status line so it works as a drop-in replacement.
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

	for _, line := range settings.Lines {
		rendered := render.RenderLine(line, &settings, ctx)
		processed := render.PostProcess(rendered)
		if processed != "" {
			fmt.Println(processed)
		}
	}
	return nil
}

func writeDump(data []byte, path string) error {
	// Try to pretty-print; fall back to raw data if JSON is invalid.
	var raw json.RawMessage
	if json.Unmarshal(data, &raw) == nil {
		pretty, err := json.MarshalIndent(raw, "", "  ")
		if err == nil {
			pretty = append(pretty, '\n')
			return os.WriteFile(path, pretty, 0o600)
		}
	}
	return os.WriteFile(path, data, 0o600)
}
