package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd(b build) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "c2 %s\n", b.version)
			fmt.Fprintf(out, "  commit: %s\n", b.commit)
			fmt.Fprintf(out, "  built:  %s\n", b.date)
		},
	}
}
