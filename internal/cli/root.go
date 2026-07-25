package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/paths"
)

var errReported = errors.New("c2: already reported")

type build struct {
	version string
	commit  string
	date    string
}

func NewRoot(b build) *cobra.Command {
	root := &cobra.Command{
		Use:           "c2",
		Short:         "Concept2 Logbook CLI",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.Flags().BoolP("version", "v", false, "output the version number")
	root.RunE = func(cmd *cobra.Command, args []string) error {
		if show, _ := cmd.Flags().GetBool("version"); show {
			fmt.Fprintln(cmd.OutOrStdout(), b.version)
			return nil
		}
		return cmd.Help()
	}

	root.AddCommand(
		newSetupCmd(b),
		newSyncCmd(b),
		newLogCmd(),
		newStatusCmd(),
		newTrendCmd(),
		newStatsCmd(),
		newShowCmd(),
		newExportCmd(),
		newReportCmd(),
		newDataCmd(),
		newNoteCmd(),
		newDocCmd("plan", "Training plan (managed document)", func(p paths.DataPaths) string { return p.Plan }),
		newDocCmd("playbook", "Coaching knowledge playbook (managed document)", func(p paths.DataPaths) string { return p.Playbook }),
		newNarrativeCmd(),
		newVersionCmd(b),
	)

	return root
}

func Execute(ctx context.Context, version, commit, date string) int {
	root := NewRoot(build{version: version, commit: commit, date: date})
	if err := root.ExecuteContext(ctx); err != nil {
		if !errors.Is(err, errReported) {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return 1
	}
	return 0
}
