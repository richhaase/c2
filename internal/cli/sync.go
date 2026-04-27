package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSyncCommand(version string, deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Pull new workouts from the API",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}
			if cfg.API.Token == "" {
				return fmt.Errorf("no API token configured. Run `c2 setup` first")
			}

			out := cmd.OutOrStdout()
			if cfg.Sync.LastSync != "" {
				fmt.Fprintf(out, "Syncing workouts since %s...\n", cfg.Sync.LastSync)
			} else {
				fmt.Fprintln(out, "First sync - pulling all workouts...")
			}

			result, err := deps.RunSync(cmd.Context(), cfg, version)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "Fetched %d workouts, %d new.\n", result.FetchedWorkouts, result.NewWorkouts)
			for _, warning := range result.Warnings {
				fmt.Fprintln(cmd.ErrOrStderr(), warning)
			}
			if result.StrokeCount > 0 {
				fmt.Fprintf(out, "Fetched stroke data for %d workouts.\n", result.StrokeCount)
			}
			fmt.Fprintf(out, "Total workouts: %d\n", result.TotalWorkouts)
			return nil
		},
	}
}
