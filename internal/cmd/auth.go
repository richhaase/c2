package cmd

import (
	"fmt"

	"github.com/richhaase/c2cli/internal/api"
	"github.com/richhaase/c2cli/internal/config"
)

func RunAuth(token string) error {
	cfg, err := config.Load()
	if err != nil {
		// Config might not exist yet — create a default one
		cfg = &config.Config{
			API: config.APIConfig{
				BaseURL: "https://log.concept2.com",
				Token:   token,
			},
			Sync: config.SyncConfig{MachineType: "rower"},
			Goal: config.GoalConfig{TargetMeters: 1_000_000},
			Display: config.DisplayConfig{DateFormat: "01/02"},
		}
	} else {
		cfg.API.Token = token
	}

	if err := config.EnsureDirs(); err != nil {
		return err
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Println("Token saved.")

	// Verify by fetching user profile
	client := api.FromConfig(cfg)
	user, err := client.GetUser()
	if err != nil {
		return fmt.Errorf("token verification failed: %w", err)
	}

	fmt.Printf("Authenticated as: %s (ID: %d)\n", user.Username, user.ID)
	return nil
}
