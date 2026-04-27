package cli

import (
	"fmt"
	"sort"

	"github.com/richhaase/c2/internal/display"
	"github.com/spf13/cobra"
)

func newLogCommand(deps Dependencies) *cobra.Command {
	var count int
	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show recent workouts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := positiveIntFlag(count, "count"); err != nil {
				return err
			}
			cfg, err := deps.LoadConfig()
			if err != nil {
				return err
			}
			workouts, err := deps.ReadWorkouts()
			if err != nil {
				return err
			}
			if len(workouts) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No workouts found. Run `c2 sync` first.")
				return nil
			}

			sort.Slice(workouts, func(i, j int) bool {
				return workouts[i].Date > workouts[j].Date
			})
			if count > len(workouts) {
				count = len(workouts)
			}
			for _, workout := range workouts[:count] {
				fmt.Fprintln(cmd.OutOrStdout(), display.FormatWorkoutLine(workout, cfg.Display.DateFormat))
			}
			return nil
		},
	}
	cmd.Flags().IntVarP(&count, "count", "n", 10, "number of workouts to display")
	return cmd
}
