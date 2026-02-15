package main

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/widget"
)

func newWidgetsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "widgets",
		Short: "List all available widget types",
		Long:  "List all registered widget types with their descriptions and default colors.",
		RunE:  runWidgets,
	}
}

func runWidgets(_ *cobra.Command, _ []string) error {
	types := widget.Types()
	sort.Strings(types)

	fmt.Println("Available widgets:")
	fmt.Println()
	for _, t := range types {
		w := widget.Get(t)
		if w == nil {
			continue
		}
		colorInfo := ""
		if c := w.DefaultColor(); c != "" {
			colorInfo = fmt.Sprintf(" (%s)", c)
		}
		fmt.Printf("  %-28s %s%s\n", t, w.Description(), colorInfo)
	}
	return nil
}
