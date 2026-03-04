// Copyright (c) 2026 Rich Haase. All rights reserved.
// Use of this source code is governed by the MIT license.

package commands

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2cli/internal/models"
	"github.com/richhaase/c2cli/internal/storage"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export workouts to CSV or JSON",
	Long:  "Export all stored workouts to stdout in CSV or JSON format.",
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		return runExport(format, from, to)
	},
}

func init() {
	exportCmd.Flags().StringP("format", "f", "csv", "output format: csv, json, or jsonl")
	exportCmd.Flags().String("from", "", "filter workouts from date (YYYY-MM-DD)")
	exportCmd.Flags().String("to", "", "filter workouts to date (YYYY-MM-DD)")
	rootCmd.AddCommand(exportCmd)
}

func runExport(format, from, to string) error {
	workouts, err := storage.ReadWorkouts()
	if err != nil {
		return err
	}
	if len(workouts) == 0 {
		return fmt.Errorf("no workouts found: run c2 sync first")
	}

	// Apply date filters
	workouts = filterByDate(workouts, from, to)
	if len(workouts) == 0 {
		return fmt.Errorf("no workouts match the specified date range")
	}

	// Sort oldest first for export
	sort.Slice(workouts, func(i, j int) bool {
		return workouts[i].Date < workouts[j].Date
	})

	switch format {
	case "csv":
		return exportCSV(workouts)
	case "json":
		return exportJSON(workouts)
	case "jsonl":
		return exportJSONL(workouts)
	default:
		return fmt.Errorf("unsupported format %q: must be csv, json, or jsonl", format)
	}
}

func filterByDate(workouts []models.Workout, from, to string) []models.Workout {
	if from == "" && to == "" {
		return workouts
	}
	var filtered []models.Workout
	for _, w := range workouts {
		date := w.Date[:10] // "2006-01-02" portion
		if from != "" && date < from {
			continue
		}
		if to != "" && date > to {
			continue
		}
		filtered = append(filtered, w)
	}
	return filtered
}

func exportCSV(workouts []models.Workout) error {
	cw := csv.NewWriter(os.Stdout)
	defer cw.Flush()

	header := []string{
		"id", "date", "distance", "time_tenths", "time_formatted",
		"pace_500m", "stroke_rate", "stroke_count", "calories",
		"drag_factor", "hr_avg", "hr_min", "hr_max",
		"workout_type", "machine_type", "comments",
	}
	if err := cw.Write(header); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}

	for _, w := range workouts {
		hrAvg, hrMin, hrMax := "", "", ""
		if w.HeartRate != nil {
			if w.HeartRate.Average > 0 {
				hrAvg = strconv.Itoa(w.HeartRate.Average)
			}
			if w.HeartRate.Min > 0 {
				hrMin = strconv.Itoa(w.HeartRate.Min)
			}
			if w.HeartRate.Max > 0 {
				hrMax = strconv.Itoa(w.HeartRate.Max)
			}
		}

		record := []string{
			strconv.FormatInt(w.ID, 10),
			w.Date,
			strconv.FormatInt(w.Distance, 10),
			strconv.FormatInt(w.Time, 10),
			w.TimeFormatted,
			w.Pace500m(),
			strconv.Itoa(w.StrokeRate),
			strconv.Itoa(w.StrokeCount),
			strconv.Itoa(w.CaloriesTotal),
			strconv.Itoa(w.DragFactor),
			hrAvg, hrMin, hrMax,
			w.WorkoutType,
			w.MachineType,
			w.Comments,
		}
		if err := cw.Write(record); err != nil {
			return fmt.Errorf("write CSV row: %w", err)
		}
	}
	return nil
}

func exportJSON(workouts []models.Workout) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(workouts)
}

func exportJSONL(workouts []models.Workout) error {
	enc := json.NewEncoder(os.Stdout)
	for _, w := range workouts {
		if err := enc.Encode(w); err != nil {
			return fmt.Errorf("encode workout: %w", err)
		}
	}
	return nil
}
