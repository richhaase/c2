package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/api"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/storage"
	"github.com/richhaase/c2/internal/store"
)

const strokeFailureLimit = 3

func syncStrokes(cmd *cobra.Command, client *api.Client, p paths.DataPaths, workouts []models.Workout) (int, error) {
	count := 0
	failures := 0
	errOut := cmd.ErrOrStderr()
	for _, w := range workouts {
		if !w.StrokeData {
			continue
		}
		has, err := storage.HasStrokeData(p, w.ID)
		if err != nil {
			return count, err
		}
		if has {
			continue
		}
		strokes, err := client.GetStrokes(cmd.Context(), w.ID)
		if err != nil {
			failures++
			fmt.Fprintf(errOut, "Warning: failed to fetch strokes for workout %d: %v\n", w.ID, err)
			if failures >= strokeFailureLimit {
				fmt.Fprintln(errOut, "Too many failures, skipping remaining stroke data.")
				break
			}
			continue
		}
		if len(strokes) > 0 {
			if err := storage.WriteStrokeData(p, w.ID, strokes); err != nil {
				return count, err
			}
			count++
		}
	}
	return count, nil
}

func syncTimestamp(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

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

			client := api.FromConfig(cfg, b.version)
			meta := storage.ReadMeta(p, warn)
			from := ""
			if meta != nil {
				from = meta.LastSync
			}

			out := cmd.OutOrStdout()
			if from != "" {
				fmt.Fprintf(out, "Syncing workouts since %s...\n", from)
			} else {
				fmt.Fprintln(out, "First sync — pulling all workouts...")
			}

			workouts, err := client.GetAllResults(cmd.Context(), from, "")
			if err != nil {
				return err
			}
			written, err := storage.AppendWorkouts(p, workouts)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "Fetched %d workouts, %d new.\n", len(workouts), written)

			strokeCount, err := syncStrokes(cmd, client, p, workouts)
			if err != nil {
				return err
			}
			if strokeCount > 0 {
				fmt.Fprintf(out, "Fetched stroke data for %d workouts.\n", strokeCount)
			}

			newMeta := storage.StoreMeta{
				SchemaVersion: new(storage.SchemaVersion),
				Created:       now.UTC().Format("2006-01-02T15:04:05.000Z07:00"),
				LastSync:      syncTimestamp(time.Now()),
			}
			if meta != nil {
				if meta.SchemaVersion != nil {
					newMeta.SchemaVersion = meta.SchemaVersion
				}
				if meta.Created != "" {
					newMeta.Created = meta.Created
				}
			}
			if err := storage.WriteMeta(p, newMeta); err != nil {
				return err
			}

			compacted, err := notes.Compact(p, now)
			if err != nil {
				return err
			}
			errOut := cmd.ErrOrStderr()
			for _, year := range compacted.SkippedYears {
				fmt.Fprintf(errOut, "Warning: notes/archive/%d.jsonl has corrupt lines; left untouched (run `c2 data doctor`).\n", year)
			}
			if compacted.Archived > 0 {
				fmt.Fprintf(out, "Compacted %d note%s into %s.\n",
					compacted.Archived, plural(compacted.Archived), archiveNames(compacted.Years))
			}

			total, err := storage.WorkoutCount(p)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "Total workouts: %d\n", total)
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
