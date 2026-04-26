package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	API     APIConfig     `json:"api"`
	Sync    SyncConfig    `json:"sync"`
	Goal    GoalConfig    `json:"goal"`
	Display DisplayConfig `json:"display"`
}

type APIConfig struct {
	BaseURL string `json:"base_url"`
	Token   string `json:"token"`
}

type SyncConfig struct {
	LastSync    string `json:"last_sync,omitempty"`
	MachineType string `json:"machine_type"`
}

type GoalConfig struct {
	TargetMeters int    `json:"target_meters"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
}

type DisplayConfig struct {
	DateFormat string `json:"date_format"`
}

func Default() Config {
	return Config{
		API: APIConfig{
			BaseURL: "https://log.concept2.com",
			Token:   "",
		},
		Sync: SyncConfig{
			MachineType: "rower",
		},
		Goal: GoalConfig{
			TargetMeters: 1000000,
			StartDate:    "",
			EndDate:      "",
		},
		Display: DisplayConfig{
			DateFormat: "%m/%d",
		},
	}
}

func Load() (Config, error) {
	return LoadFromPath(ConfigPath())
}

func LoadFromPath(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return Config{}, err
	}

	var parsed partialConfig
	if err := json.Unmarshal(data, &parsed); err != nil {
		return Config{}, err
	}

	cfg := Default()
	mergePartial(&cfg, parsed)
	return cfg, nil
}

func Save(cfg Config) error {
	return saveToPath(ConfigPath(), cfg)
}

func EnsureDirs() error {
	if err := os.MkdirAll(DataDir(), 0o755); err != nil {
		return err
	}
	return os.MkdirAll(StrokesDir(), 0o755)
}

func ParseGoalDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, time.Local)
}

func saveToPath(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(cfg)
}

type partialConfig struct {
	API     *partialAPIConfig     `json:"api"`
	Sync    *partialSyncConfig    `json:"sync"`
	Goal    *partialGoalConfig    `json:"goal"`
	Display *partialDisplayConfig `json:"display"`
}

type partialAPIConfig struct {
	BaseURL *string `json:"base_url"`
	Token   *string `json:"token"`
}

type partialSyncConfig struct {
	LastSync    *string `json:"last_sync"`
	MachineType *string `json:"machine_type"`
}

type partialGoalConfig struct {
	TargetMeters *int    `json:"target_meters"`
	StartDate    *string `json:"start_date"`
	EndDate      *string `json:"end_date"`
}

type partialDisplayConfig struct {
	DateFormat *string `json:"date_format"`
}

func mergePartial(cfg *Config, parsed partialConfig) {
	if parsed.API != nil {
		if parsed.API.BaseURL != nil {
			cfg.API.BaseURL = *parsed.API.BaseURL
		}
		if parsed.API.Token != nil {
			cfg.API.Token = *parsed.API.Token
		}
	}
	if parsed.Sync != nil {
		if parsed.Sync.LastSync != nil {
			cfg.Sync.LastSync = *parsed.Sync.LastSync
		}
		if parsed.Sync.MachineType != nil {
			cfg.Sync.MachineType = *parsed.Sync.MachineType
		}
	}
	if parsed.Goal != nil {
		if parsed.Goal.TargetMeters != nil {
			cfg.Goal.TargetMeters = *parsed.Goal.TargetMeters
		}
		if parsed.Goal.StartDate != nil {
			cfg.Goal.StartDate = *parsed.Goal.StartDate
		}
		if parsed.Goal.EndDate != nil {
			cfg.Goal.EndDate = *parsed.Goal.EndDate
		}
	}
	if parsed.Display != nil {
		if parsed.Display.DateFormat != nil {
			cfg.Display.DateFormat = *parsed.Display.DateFormat
		}
	}
}
