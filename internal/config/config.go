package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"os"
	"path/filepath"
	"time"

	"github.com/richhaase/c2/internal/atomicfile"
	"github.com/richhaase/c2/internal/jsonx"
)

type APIConfig struct {
	BaseURL string `json:"base_url"`
	Token   string `json:"token"`
}

type GoalConfig struct {
	TargetMeters int    `json:"target_meters"`
	StartDate    string `json:"start_date"`
	EndDate      string `json:"end_date"`
}

type DisplayConfig struct {
	DateFormat string `json:"date_format"`
}

type Config struct {
	DataDir string        `json:"data_dir"`
	API     APIConfig     `json:"api"`
	Goal    GoalConfig    `json:"goal"`
	Display DisplayConfig `json:"display"`
}

func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("Cannot locate home directory: %w", err)
	}
	return filepath.Join(home, ".config", "c2"), nil
}

func File() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func DefaultDataDir() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "data"), nil
}

func Default() (Config, error) {
	dataDir, err := DefaultDataDir()
	if err != nil {
		return Config{}, err
	}
	return Config{
		DataDir: dataDir,
		API:     APIConfig{BaseURL: "https://log.concept2.com"},
		Goal:    GoalConfig{TargetMeters: 1_000_000},
		Display: DisplayConfig{DateFormat: "%m/%d"},
	}, nil
}

func Load() (Config, error) {
	cfg, err := Default()
	if err != nil {
		return Config{}, err
	}
	path, err := File()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.DataDir == "" {
		cfg.DataDir, err = DefaultDataDir()
		if err != nil {
			return Config{}, err
		}
	}
	return cfg, nil
}

func Save(cfg Config) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	path, err := File()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := jsonx.Indent(cfg)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicfile.Write(path, data, 0o600)
}

func ParseGoalDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("Invalid date: %s", s)
	}
	return t, nil
}
