package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.API.BaseURL != "https://log.concept2.com" {
		t.Fatalf("BaseURL = %q", cfg.API.BaseURL)
	}
	if cfg.Sync.MachineType != "rower" {
		t.Fatalf("MachineType = %q", cfg.Sync.MachineType)
	}
	if cfg.Goal.TargetMeters != 1000000 {
		t.Fatalf("TargetMeters = %d", cfg.Goal.TargetMeters)
	}
}

func TestParseGoalDate(t *testing.T) {
	if _, err := ParseGoalDate("2026-01-01"); err != nil {
		t.Fatalf("ParseGoalDate valid date: %v", err)
	}
	if _, err := ParseGoalDate("bad-date"); err == nil {
		t.Fatal("ParseGoalDate bad date returned nil error")
	}
}

func TestLoadMergesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	writeFileForTest(t, path, `{"api":{"token":"abc"}}
`)
	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.API.Token != "abc" || cfg.API.BaseURL == "" || cfg.Sync.MachineType != "rower" {
		t.Fatalf("LoadFromPath did not merge defaults: %#v", cfg)
	}
}

func TestLoadMissingFileReturnsDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "config.json")
	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg != Default() {
		t.Fatalf("LoadFromPath missing file = %#v", cfg)
	}
}

func TestLoadInvalidJSONReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeFileForTest(t, path, `{"api":`)
	if _, err := LoadFromPath(path); err == nil {
		t.Fatal("LoadFromPath invalid JSON returned nil error")
	}
}

func TestLoadTrailingInvalidJSONReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	writeFileForTest(t, path, `{"api":{"token":"abc"}} garbage`)
	if _, err := LoadFromPath(path); err == nil {
		t.Fatal("LoadFromPath trailing invalid JSON returned nil error")
	}
}

func TestSaveWritesPrettyJSONWithTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfg := Default()
	cfg.API.Token = "abc"
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "c2", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "api": {
    "base_url": "https://log.concept2.com",
    "token": "abc"
  },
  "sync": {
    "machine_type": "rower"
  },
  "goal": {
    "target_meters": 1000000,
    "start_date": "",
    "end_date": ""
  },
  "display": {
    "date_format": "%m/%d"
  }
}
`
	if string(got) != want {
		t.Fatalf("config file = %q", string(got))
	}
}

func TestSaveRestrictsExistingConfigPermissions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "c2", "config.json")
	writeFileForTest(t, path, `{"api":{"token":"old"}}`)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Default()
	cfg.API.Token = "secret-token"
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got&0o077 != 0 {
		t.Fatalf("config file mode = %v, want no group/world permissions", got)
	}
}

func TestEnsureDirsCreatesDataAndStrokesDirs(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	if err := EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(dir, "c2", "data"),
		filepath.Join(dir, "c2", "data", "strokes"),
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", path)
		}
	}
}

func writeFileForTest(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
