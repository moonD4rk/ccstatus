package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/widget"
)

func newWidgetsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "widgets",
		Short: "List all available widget types",
		Long:  "List all registered widget types with their descriptions, default colors and default prefixes.",
		RunE:  runWidgets,
	}
}

func runWidgets(cmd *cobra.Command, _ []string) error {
	types := widget.Types()
	sort.Strings(types)

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Available widgets:")
	fmt.Fprintln(out)
	for _, t := range types {
		w := widget.Get(t)
		if w == nil {
			continue
		}
		var details []string
		if c := w.DefaultColor(); c != "" {
			details = append(details, c)
		}
		if p := w.DefaultPrefix(); p != "" {
			details = append(details, fmt.Sprintf("prefix %q", p))
		}
		info := ""
		if len(details) > 0 {
			info = " (" + strings.Join(details, ", ") + ")"
		}
		fmt.Fprintf(out, "  %-28s %s%s\n", t, w.Description(), info)
	}
	return nil
}
