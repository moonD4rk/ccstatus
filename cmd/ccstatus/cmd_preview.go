package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/sample"
)

func newPreviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "preview",
		Short: "Render your settings against sample data",
		Long: "Render the current settings.json against a sample Claude Code payload (the official docs'\n" +
			"full example, timestamps moved to now) or against --input FILE, so a layout can be tried\n" +
			"without starting Claude Code. --subagent previews subagentLine rows instead.",
		RunE: runPreview,
	}
	cmd.Flags().String("input", "", "Payload file to render instead of the built-in sample")
	cmd.Flags().Int("width", 0, "Render width in columns (default: the terminal width)")
	cmd.Flags().Bool("subagent", false, "Preview subagentLine rows with a sample task list")
	return cmd
}

func runPreview(cmd *cobra.Command, _ []string) error {
	input, _ := cmd.Flags().GetString("input")
	width, _ := cmd.Flags().GetInt("width")
	subagent, _ := cmd.Flags().GetBool("subagent")

	var data []byte
	switch {
	case input != "":
		var err error
		data, err = os.ReadFile(filepath.Clean(input))
		if err != nil {
			return err
		}
	case subagent:
		data = sample.Subagent(time.Now())
	default:
		data = sample.StatusLine(time.Now())
	}

	if !subagent {
		renderStatusLine(cmd.OutOrStdout(), cmd.ErrOrStderr(), data, nil, width)
		return nil
	}
	rows, err := subagentRows(data, width, cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	for _, row := range rows {
		fmt.Fprintln(cmd.OutOrStdout(), row.Content)
	}
	return nil
}
