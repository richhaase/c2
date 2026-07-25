package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	"os"
	"path/filepath"
	"time"

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

func Dir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "c2")
	}
	return filepath.Join(home, ".config", "c2")
}

func File() string {
	return filepath.Join(Dir(), "config.json")
}

func DefaultDataDir() string {
	return filepath.Join(Dir(), "data")
}

func Default() Config {
	return Config{
		DataDir: DefaultDataDir(),
		API:     APIConfig{BaseURL: "https://log.concept2.com"},
		Goal:    GoalConfig{TargetMeters: 1_000_000},
		Display: DisplayConfig{DateFormat: "%m/%d"},
	}
}

func Load() (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(File())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), err
	}
	if cfg.DataDir == "" {
		cfg.DataDir = DefaultDataDir()
	}
	return cfg, nil
}

func Save(cfg Config) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	data, err := jsonx.Indent(cfg)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(File(), data, 0o600); err != nil {
		return err
	}
	return os.Chmod(File(), 0o600)
}

func ParseGoalDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("Invalid date: %s", s)
	}
	return t, nil
}
