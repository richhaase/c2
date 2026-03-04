// Copyright (c) 2026 Rich Haase. All rights reserved.
// Use of this source code is governed by the MIT license.

package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2cli/internal/api"
	"github.com/richhaase/c2cli/internal/config"
	"github.com/richhaase/c2cli/internal/storage"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Pull new workouts from the API",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync(cmd.Context())
		},
	}
}

func runSync(ctx context.Context) error {
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

	workouts, err := client.GetAllResults(ctx, from, "")
	if err != nil {
		return fmt.Errorf("fetch workouts: %w", err)
	}

	written, err := storage.AppendWorkouts(workouts)
	if err != nil {
		return fmt.Errorf("store workouts: %w", err)
	}

	fmt.Printf("Fetched %d workouts, %d new.\n", len(workouts), written)

	// Fetch stroke data for workouts that have it
	strokeCount := 0
	for _, w := range workouts {
		if !w.StrokeDataAvl || storage.HasStrokeData(w.ID) {
			continue
		}
		strokes, err := client.GetStrokes(ctx, w.ID)
		if err != nil {
			fmt.Printf("Warning: failed to fetch strokes for workout %d: %v\n", w.ID, err)
			continue
		}
		if len(strokes) > 0 {
			if err := storage.WriteStrokeData(w.ID, strokes); err != nil {
				fmt.Printf("Warning: failed to write strokes for workout %d: %v\n", w.ID, err)
				continue
			}
			strokeCount++
		}
	}
	if strokeCount > 0 {
		fmt.Printf("Fetched stroke data for %d workouts.\n", strokeCount)
	}

	cfg.Sync.LastSync = time.Now().UTC().Format("2006-01-02T15:04:05Z")
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("update config: %w", err)
	}

	total, err := storage.WorkoutCount()
	if err != nil {
		return fmt.Errorf("count workouts: %w", err)
	}
	fmt.Printf("Total workouts: %d\n", total)
	return nil
}
