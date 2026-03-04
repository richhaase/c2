// Copyright (c) 2026 Rich Haase. All rights reserved.
// Use of this source code is governed by the MIT license.

package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2cli/internal/api"
	"github.com/richhaase/c2cli/internal/config"
)

var authCmd = &cobra.Command{
	Use:   "auth <token>",
	Short: "Save access token and verify",
	Long:  "Save your Concept2 personal access token (from log.concept2.com) and verify it works.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAuth(cmd.Context(), args[0])
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
}

func runAuth(ctx context.Context, token string) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}
	cfg.API.Token = token

	if err := config.EnsureDirs(); err != nil {
		return err
	}
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	fmt.Println("Token saved.")

	client := api.FromConfig(cfg)
	user, err := client.GetUser(ctx)
	if err != nil {
		return fmt.Errorf("token verification failed: %w", err)
	}

	fmt.Printf("Authenticated as: %s (ID: %d)\n", user.Username, user.ID)
	return nil
}
