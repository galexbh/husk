package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/galexbh/husk/internal/version"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Muestra la versión de husk",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "husk %s\n", version.Version)
			fmt.Fprintf(out, "  commit:     %s\n", version.Commit)
			fmt.Fprintf(out, "  build date: %s\n", version.BuildDate)
			fmt.Fprintf(out, "  go version: %s\n", version.GoVersion())
			return nil
		},
	}
}
