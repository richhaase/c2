package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/envelope"
	"github.com/richhaase/c2/internal/models"
)

type logPayload struct {
	Count    int                     `json:"count"`
	Workouts []display.WorkoutOutput `json:"workouts"`
}

func newLogCmd() *cobra.Command {
	var (
		count  string
		from   string
		to     string
		asJSON bool
	)

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show recent workouts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateDateFlag(cmd, "--from", from); err != nil {
				return err
			}
			if err := validateDateFlag(cmd, "--to", to); err != nil {
				return err
			}
			n, err := positiveInt(cmd, count, "Error: --count must be a positive integer.")
			if err != nil {
				return err
			}

			cfg, _, all, err := loadWorkouts(cmd)
			if err != nil {
				return err
			}
			workouts := models.FilterByDate(all, from, to)
			sort.SliceStable(workouts, func(i, j int) bool {
				return workouts[i].Date > workouts[j].Date
			})
			if n > len(workouts) {
				n = len(workouts)
			}
			shown := workouts[:n]

			out := cmd.OutOrStdout()
			if asJSON {
				payload := make([]display.WorkoutOutput, 0, len(shown))
				for _, w := range shown {
					payload = append(payload, display.WorkoutOutputOf(w))
				}
				return envelope.Print(out, "c2.log.v1", logPayload{
					Count:    len(shown),
					Workouts: payload,
				})
			}

			if len(all) == 0 {
				fmt.Fprintln(out, "No workouts found. Run `c2 sync` first.")
				return nil
			}
			if len(shown) == 0 {
				fmt.Fprintln(out, "No workouts match the specified date range.")
				return nil
			}

			for _, w := range shown {
				line := display.FormatWorkoutLine(w, cfg.Display.DateFormat)
				if w.Comments != "" {
					fmt.Fprintf(out, "%s  — %s\n", line, w.Comments)
					continue
				}
				fmt.Fprintln(out, line)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&count, "count", "n", "10", "number of workouts to display")
	cmd.Flags().StringVar(&from, "from", "", "only workouts on or after date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&to, "to", "", "only workouts on or before date (YYYY-MM-DD)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}
