package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/render"
	"github.com/moond4rk/ccstatus/internal/status"
	"github.com/moond4rk/ccstatus/internal/terminal"
	"github.com/moond4rk/ccstatus/internal/widget"
)

func newSubagentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "subagent",
		Short: "Render subagent rows for Claude Code's subagentStatusLine",
		Long: "Read the subagentStatusLine payload from stdin, render the subagentLine widgets for each task,\n" +
			"and print one {\"id\",\"content\"} JSON line per row. Tasks whose status is listed in\n" +
			"subagentHideStatuses get an empty row, which hides them. Rows that render empty, and every row\n" +
			"on any error, are left out so Claude Code keeps its default rendering for them.",
		RunE: runSubagent,
	}
}

func runSubagent(cmd *cobra.Command, _ []string) error {
	if awaitsTypedInput(cmd) {
		return cmd.Help()
	}
	data, err := io.ReadAll(cmd.InOrStdin())
	writeSubagentRows(cmd.OutOrStdout(), cmd.ErrOrStderr(), data, err)
	return nil
}

// writeSubagentRows prints one {"id","content"} JSON line per subagent row of
// the payload in data. On any error, a panic included, it prints nothing, so
// Claude Code keeps its default rows.
func writeSubagentRows(stdout, stderr io.Writer, data []byte, readErr error) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(stderr, "ccstatus: panic: %v\n%s", r, debug.Stack())
		}
	}()
	if readErr != nil {
		fmt.Fprintf(stderr, "ccstatus: reading stdin: %v\n", readErr)
		return
	}
	rows, err := subagentRows(data, 0, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "ccstatus: %v\n", err)
		return
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, row := range rows {
		if enc.Encode(row) != nil {
			return
		}
	}
	_, _ = stdout.Write(buf.Bytes())
}

// subagentRow overrides one row of Claude Code's agent panel; an empty Content
// hides the row.
type subagentRow struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

// subagentRows renders settings.subagentLine for every task in the payload, at
// width columns or, when width is 0, the payload's "columns". Tasks whose status
// is in settings.subagentHideStatuses get an empty row; tasks whose row renders
// empty are left out.
func subagentRows(data []byte, width int, stderr io.Writer) ([]subagentRow, error) {
	settings, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("settings.json: %w", err)
	}
	payload, dropped, err := status.ParseSubagent(data)
	if err != nil {
		return nil, fmt.Errorf("invalid subagent input: %w", err)
	}
	if len(dropped) > 0 {
		fmt.Fprintf(stderr, "ccstatus: ignored fields with unexpected types: %s\n", strings.Join(dropped, ", "))
	}
	if width <= 0 {
		width = payload.Columns
	}
	if width <= 0 {
		width = terminal.Width(settings.TerminalWidth)
	}
	rows := make([]subagentRow, 0, len(payload.Tasks))
	for i := range payload.Tasks {
		task := &payload.Tasks[i]
		if task.ID == "" {
			continue
		}
		if slices.Contains(settings.SubagentHideStatuses, task.Status) {
			rows = append(rows, subagentRow{ID: task.ID})
			continue
		}
		content := render.PostProcess(render.RenderRow(settings.SubagentLine, &settings, widget.RenderContext{Task: task}, width))
		if content != "" {
			rows = append(rows, subagentRow{ID: task.ID, Content: content})
		}
	}
	return rows, nil
}
