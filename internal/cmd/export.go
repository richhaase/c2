// Copyright (c) 2026 Rich Haase. All rights reserved.
// Use of this source code is governed by the MIT license.

package cmd

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

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
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
	cmd.Flags().StringP("format", "f", "csv", "output format: csv, json, or jsonl")
	cmd.Flags().String("from", "", "filter workouts from date (YYYY-MM-DD)")
	cmd.Flags().String("to", "", "filter workouts to date (YYYY-MM-DD)")
	return cmd
}

func runExport(format, from, to string) error {
	workouts, err := storage.ReadWorkouts()
	if err != nil {
		return err
	}
	if len(workouts) == 0 {
		return fmt.Errorf("no workouts found — run `c2cli sync` first")
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
		return fmt.Errorf("unsupported format %q — use csv, json, or jsonl", format)
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
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	header := []string{
		"id", "date", "distance", "time_tenths", "time_formatted",
		"pace_500m", "stroke_rate", "stroke_count", "calories",
		"drag_factor", "hr_avg", "hr_min", "hr_max",
		"workout_type", "machine_type", "comments",
	}
	if err := w.Write(header); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}

	for _, wo := range workouts {
		hrAvg, hrMin, hrMax := "", "", ""
		if wo.HeartRate != nil {
			if wo.HeartRate.Average > 0 {
				hrAvg = strconv.Itoa(wo.HeartRate.Average)
			}
			if wo.HeartRate.Min > 0 {
				hrMin = strconv.Itoa(wo.HeartRate.Min)
			}
			if wo.HeartRate.Max > 0 {
				hrMax = strconv.Itoa(wo.HeartRate.Max)
			}
		}

		record := []string{
			strconv.FormatInt(wo.ID, 10),
			wo.Date,
			strconv.FormatInt(wo.Distance, 10),
			strconv.FormatInt(wo.Time, 10),
			wo.TimeFormatted,
			wo.Pace500m(),
			strconv.Itoa(wo.StrokeRate),
			strconv.Itoa(wo.StrokeCount),
			strconv.Itoa(wo.CaloriesTotal),
			strconv.Itoa(wo.DragFactor),
			hrAvg, hrMin, hrMax,
			wo.WorkoutType,
			wo.MachineType,
			wo.Comments,
		}
		if err := w.Write(record); err != nil {
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
