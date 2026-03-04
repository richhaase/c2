// Copyright (c) 2026 Rich Haase. All rights reserved.
// Use of this source code is governed by the MIT license.

package commands

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2cli/internal/config"
	"github.com/richhaase/c2cli/internal/display"
	"github.com/richhaase/c2cli/internal/storage"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show recent workouts",
	Long:  "Show the last N workouts in compact format (default: 10).",
	RunE: func(cmd *cobra.Command, args []string) error {
		n, _ := cmd.Flags().GetInt("count")
		return runLog(n)
	},
}

func init() {
	logCmd.Flags().IntP("count", "n", 10, "number of workouts to display")
	rootCmd.AddCommand(logCmd)
}

func runLog(n int) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	workouts, err := storage.ReadWorkouts()
	if err != nil {
		return err
	}
	if len(workouts) == 0 {
		fmt.Println("No workouts found. Run `c2cli sync` first.")
		return nil
	}

	sort.Slice(workouts, func(i, j int) bool {
		return workouts[i].Date > workouts[j].Date
	})

	if n > len(workouts) {
		n = len(workouts)
	}

	for _, w := range workouts[:n] {
		fmt.Println(display.FormatWorkoutLine(&w, cfg.Display.DateFormat))
	}
	return nil
}
