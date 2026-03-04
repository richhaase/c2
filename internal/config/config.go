// Copyright (c) 2026 Rich Haase. All rights reserved.
// Use of this source code is governed by the MIT license.

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	API     APIConfig     `toml:"api"`
	Sync    SyncConfig    `toml:"sync"`
	Goal    GoalConfig    `toml:"goal"`
	Display DisplayConfig `toml:"display"`
}

type APIConfig struct {
	BaseURL string `toml:"base_url"`
	Token   string `toml:"token"`
}

type SyncConfig struct {
	LastSync    string `toml:"last_sync,omitempty"`
	MachineType string `toml:"machine_type"`
}

type GoalConfig struct {
	TargetMeters int64  `toml:"target_meters"`
	StartDate    string `toml:"start_date"`
	EndDate      string `toml:"end_date"`
}

type DisplayConfig struct {
	DateFormat string `toml:"date_format"`
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	return &Config{
		API: APIConfig{
			BaseURL: "https://log.concept2.com",
		},
		Sync: SyncConfig{
			MachineType: "rower",
		},
		Goal: GoalConfig{
			TargetMeters: 1_000_000,
		},
		Display: DisplayConfig{
			DateFormat: "01/02",
		},
	}
}

// Dir returns the config directory: ~/.config/c2cli/
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "c2cli"), nil
}

// DataDir returns the data directory: ~/.config/c2cli/data/
func DataDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "data"), nil
}

// EnsureDirs creates all required directories.
func EnsureDirs() error {
	dataDir, err := DataDir()
	if err != nil {
		return err
	}
	strokesDir := filepath.Join(dataDir, "strokes")
	return os.MkdirAll(strokesDir, 0o755)
}

// Load reads config from ~/.config/c2cli/config.toml.
func Load() (*Config, error) {
	dir, err := Dir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "config.toml")

	cfg := Default()
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("read config at %s: %w", path, err)
	}

	return cfg, nil
}

// Save writes config to ~/.config/c2cli/config.toml.
func Save(cfg *Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "config.toml")

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		f.Close() //nolint:errcheck,gosec // closing on error path
		return fmt.Errorf("encode config: %w", err)
	}
	return f.Close()
}

// ParseGoalDate parses a date string in "2006-01-02" format.
func ParseGoalDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}
