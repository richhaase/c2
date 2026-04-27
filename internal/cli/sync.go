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
				if _, err := fmt.Fprintf(out, "Syncing workouts since %s...\n", cfg.Sync.LastSync); err != nil {
					return err
				}
			} else {
				if _, err := fmt.Fprintln(out, "First sync - pulling all workouts..."); err != nil {
					return err
				}
			}

			result, err := deps.RunSync(cmd.Context(), cfg, version)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "Fetched %d workouts, %d new.\n", result.FetchedWorkouts, result.NewWorkouts); err != nil {
				return err
			}
			for _, warning := range result.Warnings {
				if _, err := fmt.Fprintln(cmd.ErrOrStderr(), warning); err != nil {
					return err
				}
			}
			if result.StrokeCount > 0 {
				if _, err := fmt.Fprintf(out, "Fetched stroke data for %d workouts.\n", result.StrokeCount); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintf(out, "Total workouts: %d\n", result.TotalWorkouts)
			return err
		},
	}
}
