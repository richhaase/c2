package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func useHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestLoadMissingAndPartialConfigUsesDefaults(t *testing.T) {
	home := useHome(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != filepath.Join(home, ".config", "c2", "data") {
		t.Fatalf("DataDir = %q", cfg.DataDir)
	}
	if cfg.API.BaseURL != "https://log.concept2.com" || cfg.Goal.TargetMeters != 1_000_000 {
		t.Fatalf("config = %#v", cfg)
	}

	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path, err := File()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"api":{"token":"abc"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.API.Token != "abc" || cfg.API.BaseURL != "https://log.concept2.com" {
		t.Fatalf("partial config = %#v", cfg)
	}
}

func TestSaveRoundTripsWithoutEscapingAndLocksMode(t *testing.T) {
	useHome(t)
	cfg, err := Default()
	if err != nil {
		t.Fatal(err)
	}
	cfg.API.Token = "<secret&token>"
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	path, err := File()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "<secret&token>") {
		t.Fatalf("data = %s", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o", got)
	}
	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.API.Token != cfg.API.Token {
		t.Fatalf("loaded = %#v", loaded)
	}
}

func TestLoadRejectsMalformedConfig(t *testing.T) {
	useHome(t)
	dir, err := Dir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path, err := File()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("Load succeeded")
	}
}

func TestConfigPathsFailWithoutHomeDirectory(t *testing.T) {
	t.Setenv("HOME", "")
	if _, err := Dir(); err == nil {
		t.Fatal("Dir succeeded")
	}
	if _, err := Default(); err == nil {
		t.Fatal("Default succeeded")
	}
	if _, err := Load(); err == nil {
		t.Fatal("Load succeeded")
	}
	if err := Save(Config{}); err == nil {
		t.Fatal("Save succeeded")
	}
}
