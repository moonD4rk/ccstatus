package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultDumpPath   = "/tmp/ccstatus-dump.json"
	defaultAppendPath = "/tmp/ccstatus-dump.jsonl"
)

func newDumpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Dump raw JSON input from Claude Code for debugging",
		Long: "Read JSON from stdin, save to a file for inspection, and render the status line normally.\n" +
			"Useful for debugging what Claude Code sends to ccstatus. With --append, every invocation adds\n" +
			"one JSON line (timestamp, COLUMNS, LINES, input) so a whole session can be captured.",
		RunE: runDump,
	}
	cmd.Flags().StringP("output", "o", defaultDumpPath,
		"Output file path for the JSON dump (default with --append: "+defaultAppendPath+")")
	cmd.Flags().Bool("append", false, "Append one JSON line per invocation instead of overwriting")
	return cmd
}

func runDump(cmd *cobra.Command, _ []string) error {
	output, _ := cmd.Flags().GetString("output")
	appendMode, _ := cmd.Flags().GetBool("append")
	if appendMode && !cmd.Flags().Changed("output") {
		output = defaultAppendPath
	}
	stderr := cmd.ErrOrStderr()

	data, err := io.ReadAll(cmd.InOrStdin())
	if err == nil {
		write := writeDump
		if appendMode {
			write = appendDump
		}
		if writeErr := write(data, output); writeErr != nil {
			fmt.Fprintf(stderr, "Warning: failed to write dump: %v\n", writeErr)
		} else {
			fmt.Fprintf(stderr, "Dumped JSON to %s\n", output)
		}
	}

	// Still render the status line so it works as a drop-in replacement.
	renderStatusLine(cmd.OutOrStdout(), stderr, data, err, 0)
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

// dumpRecord is one line of `dump --append` output.
type dumpRecord struct {
	TS      string          `json:"ts"`
	Columns *int            `json:"columns,omitempty"`
	Lines   *int            `json:"lines,omitempty"`
	Input   json.RawMessage `json:"input"`
}

// appendDump adds data to path as one JSON line, alongside the time and the
// terminal size Claude Code passed; input that is not JSON is kept as a string.
func appendDump(data []byte, path string) error {
	rec := dumpRecord{
		TS:      time.Now().Format(time.RFC3339Nano),
		Columns: envInt("COLUMNS"),
		Lines:   envInt("LINES"),
	}
	var compact bytes.Buffer
	if json.Compact(&compact, data) == nil {
		rec.Input = compact.Bytes()
	} else {
		quoted, err := json.Marshal(string(data))
		if err != nil {
			return err
		}
		rec.Input = quoted
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Clean(path), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // path comes from CLI flag
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// envInt returns the environment variable name as an integer, or nil when it is
// unset or not a number.
func envInt(name string) *int {
	n, err := strconv.Atoi(os.Getenv(name))
	if err != nil {
		return nil
	}
	return &n
}
