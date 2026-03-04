package cmd

import (
	"fmt"
	"sort"

	"github.com/richhaase/c2cli/internal/config"
	"github.com/richhaase/c2cli/internal/display"
	"github.com/richhaase/c2cli/internal/storage"
)

func RunLog(n int) error {
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

	// Sort by date descending
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
