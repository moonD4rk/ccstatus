package main

import (
	"github.com/spf13/cobra"

	"github.com/moond4rk/ccstatus/internal/schema"
)

func newSchemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Print the JSON Schema of settings.json",
		Long: "Print the JSON Schema of settings.json, generated from the widget registry of this build.\n" +
			"A settings file that references it in \"$schema\" gets completion and checks in an editor;\n" +
			"`ccstatus init` writes that reference.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out, err := schema.Generate()
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(out)
			return err
		},
	}
}
