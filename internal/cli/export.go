package cli

import (
	"encoding/csv"
	"fmt"
	"slices"
	"sort"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/envelope"
	"github.com/richhaase/c2/internal/jsonx"
	"github.com/richhaase/c2/internal/models"
)

var csvHeader = []string{
	"id",
	"date",
	"distance",
	"time_tenths",
	"time_formatted",
	"pace_500m",
	"stroke_rate",
	"stroke_count",
	"calories",
	"drag_factor",
	"hr_avg",
	"hr_min",
	"hr_max",
	"workout_type",
	"rest_time_tenths",
	"rest_distance",
	"machine_type",
	"comments",
}

type exportPayload struct {
	Count    int              `json:"count"`
	Workouts []models.Workout `json:"workouts"`
}

func intOrEmpty(v *int) string {
	if v == nil {
		return ""
	}
	return strconv.Itoa(*v)
}

func truthyIntOrEmpty(v *int) string {
	if v == nil || *v == 0 {
		return ""
	}
	return strconv.Itoa(*v)
}

func buildCSVRow(w models.Workout) []string {
	var hrAvg, hrMin, hrMax string
	if w.HeartRate != nil {
		hrAvg = truthyIntOrEmpty(w.HeartRate.Average)
		hrMin = truthyIntOrEmpty(w.HeartRate.Min)
		hrMax = truthyIntOrEmpty(w.HeartRate.Max)
	}
	return []string{
		strconv.FormatInt(w.ID, 10),
		w.Date,
		strconv.Itoa(w.Distance),
		strconv.Itoa(w.Time),
		w.TimeFormatted,
		models.Pace500m(w),
		intOrEmpty(w.StrokeRate),
		intOrEmpty(w.StrokeCount),
		intOrEmpty(w.CaloriesTotal),
		intOrEmpty(w.DragFactor),
		hrAvg,
		hrMin,
		hrMax,
		w.WorkoutType,
		intOrEmpty(w.RestTime),
		intOrEmpty(w.RestDistance),
		w.Type,
		w.Comments,
	}
}

func writeCSV(out *csv.Writer, workouts []models.Workout) error {
	if err := out.Write(csvHeader); err != nil {
		return err
	}
	for _, workout := range workouts {
		if err := out.Write(buildCSVRow(workout)); err != nil {
			return err
		}
	}
	out.Flush()
	return out.Error()
}

func newExportCmd() *cobra.Command {
	var (
		format string
		from   string
		to     string
	)

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export workouts to CSV or JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateDateFlag(cmd, "--from", from); err != nil {
				return err
			}
			if err := validateDateFlag(cmd, "--to", to); err != nil {
				return err
			}
			if !slices.Contains([]string{"csv", "json", "jsonl"}, format) {
				return reportf(cmd, "Unsupported format %q: must be csv, json, or jsonl", format)
			}

			_, _, workouts, err := loadWorkouts(cmd)
			if err != nil {
				return err
			}
			if len(workouts) == 0 {
				return reportf(cmd, "No workouts found. Run `c2 sync` first.")
			}

			workouts = models.FilterByDate(workouts, from, to)
			if len(workouts) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "No workouts match the specified date range.")
			}
			sort.SliceStable(workouts, func(i, j int) bool {
				return workouts[i].Date < workouts[j].Date
			})

			out := cmd.OutOrStdout()
			switch format {
			case "csv":
				return writeCSV(csv.NewWriter(out), workouts)
			case "json":
				if workouts == nil {
					workouts = []models.Workout{}
				}
				return envelope.Print(out, "c2.export.v1", exportPayload{
					Count:    len(workouts),
					Workouts: workouts,
				})
			case "jsonl":
				for _, w := range workouts {
					line, err := jsonx.Compact(w)
					if err != nil {
						return err
					}
					fmt.Fprintln(out, string(line))
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "csv", "output format: csv, json, or jsonl")
	cmd.Flags().StringVar(&from, "from", "", "filter workouts from date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&to, "to", "", "filter workouts to date (YYYY-MM-DD)")
	return cmd
}
