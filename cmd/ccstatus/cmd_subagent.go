package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
			"and print one {\"id\",\"content\"} JSON line per row. Rows that render empty, and every row on any\n" +
			"error, are left out so Claude Code keeps its default rendering for them.",
		RunE: runSubagent,
	}
}

func runSubagent(cmd *cobra.Command, _ []string) error {
	data, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "ccstatus: reading stdin: %v\n", err)
		return nil
	}
	rows, err := subagentRows(data, 0, cmd.ErrOrStderr())
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "ccstatus: %v\n", err)
		return nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, row := range rows {
		if err := enc.Encode(row); err != nil {
			return nil //nolint:nilerr // print nothing so Claude Code keeps its default rows
		}
	}
	_, _ = cmd.OutOrStdout().Write(buf.Bytes())
	return nil
}

// subagentRow overrides one row of Claude Code's agent panel.
type subagentRow struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

// subagentRows renders settings.subagentLine for every task in the payload, at
// width columns or, when width is 0, the payload's "columns". Tasks whose row
// renders empty are left out.
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
		content := render.PostProcess(render.RenderRow(settings.SubagentLine, &settings, widget.RenderContext{Task: task}, width))
		if content != "" {
			rows = append(rows, subagentRow{ID: task.ID, Content: content})
		}
	}
	return rows, nil
}
