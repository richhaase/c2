package config

import (
	"os"
	"path/filepath"
)

func Dir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "c2")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".config", "c2")
	}
	return filepath.Join(home, ".config", "c2")
}

func DataDir() string {
	return filepath.Join(Dir(), "data")
}

func ConfigPath() string {
	return filepath.Join(Dir(), "config.json")
}

func WorkoutsPath() string {
	return filepath.Join(DataDir(), "workouts.jsonl")
}

func StrokesDir() string {
	return filepath.Join(DataDir(), "strokes")
}
