package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/api"
	"github.com/richhaase/c2/internal/store"
	"github.com/richhaase/c2/internal/syncer"
)

const strokeFailureLimit = 3

func newSyncCmd(b build) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Pull new workouts from the API",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, p, err := loadStore()
			if err != nil {
				return err
			}
			if cfg.API.Token == "" {
				return reportf(cmd, "No API token configured. Run `c2 setup` first.")
			}
			warn := warner(cmd)
			inspection, err := store.Inspect(p, warn)
			if err != nil {
				return err
			}
			if inspection.State == store.StateForeign {
				return reportf(cmd, "%s exists but is not a c2 data store. Fix data_dir via `c2 setup`.", p.Root)
			}
			if !inspection.Writable {
				return reportf(cmd, "Cannot write to %s.", p.Root)
			}
			now := time.Now()
			if err := store.Init(p, now, warn); err != nil {
				return err
			}

			client, err := api.FromConfig(cfg, b.version)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			result, err := syncer.Run(cmd.Context(), p, client, now, warn)
			if err != nil {
				return err
			}
			if result.Since != "" {
				fmt.Fprintf(out, "Synced workouts updated since %s.\n", result.Since)
			} else {
				fmt.Fprintln(out, "Completed first full workout sync.")
			}
			fmt.Fprintf(out, "Fetched %d workouts, %d new, %d updated.\n",
				result.Fetched, result.Workouts.Added, result.Workouts.Updated)
			if result.Strokes > 0 {
				fmt.Fprintf(out, "Fetched stroke data for %d workouts.\n", result.Strokes)
			}
			errOut := cmd.ErrOrStderr()
			for i, failure := range result.StrokeFailures {
				if i >= strokeFailureLimit {
					break
				}
				fmt.Fprintf(errOut, "Warning: failed to fetch strokes for workout %d: %v\n",
					failure.WorkoutID, failure.Err)
			}
			if extra := len(result.StrokeFailures) - strokeFailureLimit; extra > 0 {
				fmt.Fprintf(errOut, "Warning: %d additional stroke fetch failure%s suppressed.\n", extra, plural(extra))
			}
			if len(result.StrokeFailures) > 0 {
				fmt.Fprintln(errOut, "Missing stroke data will be retried on the next sync.")
			}

			for _, year := range result.Compacted.SkippedYears {
				fmt.Fprintf(errOut, "Warning: notes/archive/%d.jsonl has corrupt lines; left untouched (run `c2 data doctor`).\n", year)
			}
			if result.Compacted.Archived > 0 {
				fmt.Fprintf(out, "Compacted %d note%s into %s.\n",
					result.Compacted.Archived, plural(result.Compacted.Archived), archiveNames(result.Compacted.Years))
			}
			fmt.Fprintf(out, "Total workouts: %d\n", result.TotalWorkouts)
			return nil
		},
	}
}

func archiveNames(years []int) string {
	out := ""
	for i, y := range years {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%d.jsonl", y)
	}
	return out
}
