package cmd

import (
	"fmt"
	"time"

	"github.com/richhaase/c2cli/internal/api"
	"github.com/richhaase/c2cli/internal/config"
	"github.com/richhaase/c2cli/internal/storage"
)

func RunSync() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := config.EnsureDirs(); err != nil {
		return err
	}

	client := api.FromConfig(cfg)
	from := cfg.Sync.LastSync

	if from != "" {
		fmt.Printf("Syncing workouts since %s...\n", from)
	} else {
		fmt.Println("First sync — pulling all workouts...")
	}

	workouts, err := client.GetAllResults(from, "")
	if err != nil {
		return err
	}

	written, err := storage.AppendWorkouts(workouts)
	if err != nil {
		return err
	}

	fmt.Printf("Fetched %d workouts, %d new.\n", len(workouts), written)

	// Fetch stroke data
	strokeCount := 0
	for _, w := range workouts {
		if !storage.HasStrokeData(w.ID) {
			strokes, err := client.GetStrokes(w.ID)
			if err != nil {
				continue
			}
			if len(strokes) > 0 {
				if err := storage.WriteStrokeData(w.ID, strokes); err != nil {
					continue
				}
				strokeCount++
			}
		}
	}
	if strokeCount > 0 {
		fmt.Printf("Fetched stroke data for %d workouts.\n", strokeCount)
	}

	// Update last_sync
	cfg.Sync.LastSync = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	total, _ := storage.WorkoutCount()
	fmt.Printf("Total workouts: %d\n", total)
	return nil
}
