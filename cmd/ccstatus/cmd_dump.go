package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
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
	stderr := cmd.ErrOrStderr()

	data, err := io.ReadAll(cmd.InOrStdin())
	if err == nil {
		if writeErr := writeDump(data, output); writeErr != nil {
			fmt.Fprintf(stderr, "Warning: failed to write dump: %v\n", writeErr)
		} else {
			fmt.Fprintf(stderr, "Dumped JSON to %s\n", output)
		}
	}

	// Still render the status line so it works as a drop-in replacement.
	renderStatusLine(cmd.OutOrStdout(), stderr, data, err)
	return nil
}

func writeDump(data []byte, path string) error {
	path = filepath.Clean(path)
	// Try to pretty-print; fall back to raw data if JSON is invalid.
	var raw json.RawMessage
	if json.Unmarshal(data, &raw) == nil {
		pretty, err := json.MarshalIndent(raw, "", "  ")
		if err == nil {
			pretty = append(pretty, '\n')
			return os.WriteFile(path, pretty, 0o600) //nolint:gosec // path comes from CLI flag, not untrusted input
		}
	}
	return os.WriteFile(path, data, 0o600) //nolint:gosec // path comes from CLI flag, not untrusted input
}
