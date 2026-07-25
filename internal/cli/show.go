package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/analysis"
	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/envelope"
	"github.com/richhaase/c2/internal/jsonx"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/storage"
)

type showPayload struct {
	Workout               display.WorkoutOutput         `json:"workout"`
	Raw                   json.RawMessage               `json:"raw"`
	TargetPace500mSeconds *float64                      `json:"target_pace_500m_seconds"`
	Splits                []analysis.SplitRow           `json:"splits"`
	SplitShape            analysis.Shape                `json:"split_shape"`
	StrokeSummary         *analysis.StrokeSummaryResult `json:"stroke_summary"`
	Notes                 []notes.Record                `json:"notes"`
}

func targetPaceSeconds(w models.Workout) *float64 {
	if w.Workout == nil || w.Workout.Targets == nil || w.Workout.Targets.Pace == nil {
		return nil
	}
	if *w.Workout.Targets.Pace == 0 {
		return nil
	}
	return new(*w.Workout.Targets.Pace / models.TenthsPerSecond)
}

func newShowCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show full detail for one workout (use a workout id or 'last')",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref := args[0]
			cfg, p, workouts, err := loadWorkouts(cmd)
			if err != nil {
				return err
			}
			w := models.ResolveWorkout(workouts, ref)
			if w == nil {
				if ref == "last" {
					return reportf(cmd, "No workouts found. Run `c2 sync` first.")
				}
				return reportf(cmd, "No workout with id %s. Use `c2 log --json` to list ids.", ref)
			}

			splits := analysis.SplitTable(*w)
			shape := analysis.SplitShape(splits)
			strokes, err := storage.ReadStrokeData(p, w.ID)
			if err != nil {
				return err
			}
			var strokeSummary *analysis.StrokeSummaryResult
			if len(strokes) > 0 {
				summary := analysis.StrokeSummary(strokes)
				strokeSummary = &summary
			}
			allNotes, err := notes.ReadAll(p)
			if err != nil {
				return err
			}
			linked := notes.Apply(allNotes, notes.Filter{WorkoutID: &w.ID})

			out := cmd.OutOrStdout()
			if asJSON {
				raw := w.Raw
				if len(raw) == 0 {
					raw, err = jsonRaw(*w)
					if err != nil {
						return err
					}
				}
				if splits == nil {
					splits = []analysis.SplitRow{}
				}
				if linked == nil {
					linked = []notes.Record{}
				}
				return envelope.Print(out, "c2.show.v1", showPayload{
					Workout:               display.WorkoutOutputOf(*w),
					Raw:                   raw,
					TargetPace500mSeconds: targetPaceSeconds(*w),
					Splits:                splits,
					SplitShape:            shape,
					StrokeSummary:         strokeSummary,
					Notes:                 linked,
				})
			}

			fmt.Fprintln(out, display.FormatWorkoutLine(*w, cfg.Display.DateFormat))
			fmt.Fprintln(out)
			fmt.Fprintf(out, "Id: %d\n", w.ID)
			if w.Timezone != "" {
				fmt.Fprintf(out, "Date: %s (%s)\n", w.Date, w.Timezone)
			} else {
				fmt.Fprintf(out, "Date: %s\n", w.Date)
			}
			if w.WorkoutType != "" {
				fmt.Fprintf(out, "Type: %s\n", w.WorkoutType)
			}
			if w.Source != "" {
				fmt.Fprintf(out, "Source: %s\n", w.Source)
			}
			if w.StrokeCount != nil && *w.StrokeCount != 0 {
				fmt.Fprintf(out, "Strokes: %s\n", display.FormatMeters(*w.StrokeCount))
			}
			if w.CaloriesTotal != nil && *w.CaloriesTotal != 0 {
				fmt.Fprintf(out, "Calories: %d\n", *w.CaloriesTotal)
			}
			if hr := w.HeartRate; hr != nil && hr.Average != nil && *hr.Average != 0 {
				parts := []string{}
				if hr.Min != nil {
					parts = append(parts, fmt.Sprintf("min %d", *hr.Min))
				}
				parts = append(parts, fmt.Sprintf("avg %d", *hr.Average))
				if hr.Max != nil {
					parts = append(parts, fmt.Sprintf("max %d", *hr.Max))
				}
				if hr.Ending != nil {
					parts = append(parts, fmt.Sprintf("ending %d", *hr.Ending))
				}
				fmt.Fprintf(out, "Heart rate: %s\n", strings.Join(parts, ", "))
			}
			if w.RestTime != nil && *w.RestTime > 0 {
				fmt.Fprintf(out, "Interval rest: %s\n", models.FormatSeconds(float64(*w.RestTime)/10))
			}
			if w.RestDistance != nil && *w.RestDistance > 0 {
				fmt.Fprintf(out, "Interval rest distance: %sm\n", display.FormatMeters(*w.RestDistance))
			}
			if pace := targetPaceSeconds(*w); pace != nil {
				fmt.Fprintf(out, "Target pace: %s/500m\n", models.FormatSeconds(*pace))
			}
			if w.Comments != "" {
				fmt.Fprintf(out, "Comments: %s\n", w.Comments)
			}

			if len(splits) > 0 {
				fmt.Fprintln(out)
				fmt.Fprintf(out, "Splits (%s):\n", shape)
				fmt.Fprintln(out, "    #      dist      time     pace/500m   spm    hr")
				for _, s := range splits {
					dist := "-"
					if s.Distance != nil {
						dist = display.FormatMeters(*s.Distance) + "m"
					}
					hr := "-"
					if s.HRAvg != nil {
						hr = strconv.Itoa(*s.HRAvg)
						if s.HRMax != nil {
							hr += "/" + strconv.Itoa(*s.HRMax)
						}
					}
					fmt.Fprintf(out, "  %3d  %8s  %8s  %10s  %4s  %7s\n",
						s.Index,
						dist,
						models.FormatSeconds(s.TimeSeconds),
						orDash(s.Pace500m, func(v string) string { return v }),
						orDash(s.StrokeRate, strconv.Itoa),
						hr)
				}
			}

			if strokeSummary != nil {
				fmt.Fprintln(out)
				fmt.Fprintf(out, "Stroke data: %d samples, avg %s/500m, %sspm, HR avg %s max %s\n",
					strokeSummary.Samples,
					orDash(strokeSummary.AvgPace500m, func(v string) string { return v }),
					orDash(strokeSummary.AvgSPM, formatNumber),
					orDash(strokeSummary.AvgHR, strconv.Itoa),
					orDash(strokeSummary.MaxHR, formatNumber))
			}

			if len(linked) > 0 {
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Notes:")
				for _, n := range linked {
					fmt.Fprintf(out, "  %s [%s/%s] %s\n", n.Date[:10], n.Type, n.Author, n.Body)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func jsonRaw(w models.Workout) (json.RawMessage, error) {
	data, err := jsonx.Compact(w)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}
