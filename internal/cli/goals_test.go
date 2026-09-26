package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/goals"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/store"
)

func mustRun(t *testing.T, args ...string) result {
	t.Helper()
	r := run(t, args...)
	if r.failed {
		t.Fatalf("%v failed: %s / %s", args, r.stdout, r.stderr)
	}
	return r
}

func TestPortableGoalsThroughCLI(t *testing.T) {
	home := testHome(t)
	root := seedWorkouts(t, home)
	if r := runWithStdin(t, "Keep <145 bpm & stay calm", "plan", "set", "-"); r.failed {
		t.Fatal(r.stderr)
	}
	note := mustRun(t, "note", "add", "--type", "subjective", "Portable <note> & details")
	mustRun(t, "note", "add", "--type", "lesson", "--date", "2026-01-01", "--workout", "1", "Archived lesson")
	mustRun(t, "data", "compact")
	if r := runWithStdin(t, "Portable playbook", "playbook", "set", "-"); r.failed {
		t.Fatal(r.stderr)
	}
	if r := runWithStdin(t, "Portable narrative", "narrative", "add", ymd(-1), "-"); r.failed {
		t.Fatal(r.stderr)
	}
	if err := os.WriteFile(filepath.Join(root, "strokes", "1.jsonl"), []byte("{\"t\":1,\"d\":5}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	mustRun(t, "data", "prepare", "--timezone", "America/Denver")
	p := paths.For(root)
	c, err := goals.Read(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.AccountID != 1 || len(c.Goals) != 1 || c.Goals[0].Equipment != "all" {
		t.Fatalf("prepared: %+v", c)
	}
	added := mustRun(t, "goal", "add", "Pace & patience", "--kind", "pace", "--target", "2:30", "--json")
	if !strings.Contains(added.stdout, "Pace & patience") {
		t.Fatal("text preservation")
	}
	c, err = goals.Read(p)
	if err != nil {
		t.Fatal(err)
	}
	id := c.Goals[1].ID
	mustRun(t, "goal", "add", "Ten kilometers", "--kind", "distance", "--target", "10000")
	got := mustRun(t, "goal", "show", id, "--json")
	if !strings.Contains(got.stdout, `"achieved": true`) {
		t.Fatalf("pace: %s", got.stdout)
	}
	mustRun(t, "goal", "update", id, "--target", "1:00", "--from", ymd(-1))
	got = mustRun(t, "goal", "show", id, "--json")
	if !strings.Contains(got.stdout, `"value": null`) {
		t.Fatal("changed eligibility not applied")
	}
	mustRun(t, "goal", "archive", id)
	if strings.Contains(mustRun(t, "goal", "list").stdout, id) {
		t.Fatal("archived goal in default list")
	}
	mustRun(t, "goal", "update", id, "--archived=false", "--from", "", "--target", "2:30")
	before, err := os.ReadFile(goals.File(p))
	if err != nil {
		t.Fatal(err)
	}
	mustRun(t, "data", "prepare", "--timezone", "America/Denver")
	after, err := os.ReadFile(goals.File(p))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("prepare should be idempotent")
	}
	for _, command := range [][]string{{"playbook", "show"}, {"narrative", "show"}, {"note", "list", "--json"}} {
		mustRun(t, command...)
	}
	target := paths.For(filepath.Join(t.TempDir(), "transferred"))
	if _, err := store.Copy(p, target, nil); err != nil {
		t.Fatal(err)
	}
	otherHome := testHome(t)
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	cfg.API.Token = ""
	cfg.Goal.TargetMeters = 42
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
	mustRun(t, "data", "use", target.Root)
	loaded, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.API.Token != "" || loaded.DataDir != paths.CanonicalRoot(target.Root) {
		t.Fatal("adoption changed credentials or failed")
	}
	got = mustRun(t, "goal", "list", "--json")
	if !strings.Contains(got.stdout, `"target": 1000000`) || strings.Contains(got.stdout, `"target": 42`) {
		t.Fatal("destination goal leaked into adopted store")
	}
	if !strings.Contains(mustRun(t, "note", "list", "--json").stdout, "Portable <note> & details") {
		t.Fatalf("note lost: %s", note.stdout)
	}
	if !strings.Contains(mustRun(t, "plan", "show").stdout, "Keep <145 bpm") {
		t.Fatal("plan lost")
	}
	if !strings.Contains(mustRun(t, "note", "list", "--json").stdout, "Archived lesson") {
		t.Fatal("archived note lost")
	}
	if !strings.Contains(mustRun(t, "playbook", "show").stdout, "Portable playbook") {
		t.Fatal("playbook lost")
	}
	if !strings.Contains(mustRun(t, "narrative", "show").stdout, "Portable narrative") {
		t.Fatal("narrative lost")
	}
	if _, err := os.Stat(target.StrokeFile(1)); err != nil {
		t.Fatal("stroke file lost", err)
	}
	reportPath := filepath.Join(otherHome, "report.html")
	mustRun(t, "report", "--no-open", "-o", reportPath)
	html, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "Pace &amp; patience") {
		t.Fatal("HTML missing escaped goal")
	}
	for _, command := range [][]string{{"report", "--data"}, {"report", "--json"}, {"status", "--json"}, {"stats", "goal", "--json"}} {
		mustRun(t, command...)
	}
	for _, command := range [][]string{{"report", "--legacy", "--data"}, {"status", "--legacy", "--json"}, {"stats", "goal", "--legacy", "--json"}} {
		r := mustRun(t, command...)
		if !strings.Contains(r.stdout, ".v1") {
			t.Fatal("legacy compatibility")
		}
	}
}

func TestAdoptionRejectsCorruptionWithoutChangingConfig(t *testing.T) {
	home := testHome(t)
	seedWorkouts(t, home)
	cfgPath := filepath.Join(home, ".config", "c2", "config.json")
	before, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	bad := t.TempDir()
	if err := os.WriteFile(filepath.Join(bad, "meta.json"), []byte(`{"schema_version":99}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if !run(t, "data", "use", bad).failed {
		t.Fatal("unsupported store accepted")
	}
	after, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("failed adoption changed config")
	}
	if !run(t, "data", "use", filepath.Join(home, ".config", "c2", "data")).failed {
		t.Fatal("unprepared store adopted")
	}
}

func TestGoalValidationLeavesPreparedDataUnchanged(t *testing.T) {
	home := testHome(t)
	root := seedWorkouts(t, home)
	mustRun(t, "data", "prepare", "--timezone", "UTC")
	p := paths.For(root)
	before, err := os.ReadFile(goals.File(p))
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"goal", "add", "x", "--kind", "pace", "--target", "NaN"},
		{"goal", "add", "x", "--kind", "pace", "--target", "2:99"},
		{"goal", "add", "x", "--kind", "distance", "--target", "2.5"},
		{"goal", "add", "x", "--target", "2000", "--from", "2026-02-30"},
		{"goal", "add", "x", "--target", "2000", "--from", "2026-09-26", "--to", "2026-01-01"},
		{"goal", "update", "missing", "--target", "10"},
		{"data", "prepare", "--timezone", "America/Denver"},
	} {
		if !run(t, args...).failed {
			t.Fatalf("accepted invalid %v", args)
		}
	}
	after, err := os.ReadFile(goals.File(p))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("invalid operation changed goals")
	}
}
