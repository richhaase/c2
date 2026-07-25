package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func ymd(offsetDays int) string {
	return time.Now().AddDate(0, 0, offsetDays).Format("2006-01-02")
}

func testHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".config", "c2")
	dataDir := filepath.Join(configDir, "data")
	if err := os.MkdirAll(filepath.Join(dataDir, "strokes"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := fmt.Sprintf(`{
  "data_dir": %q,
  "api": {"base_url": "https://log.concept2.com", "token": "tok"},
  "goal": {"target_meters": 1000000, "start_date": %q, "end_date": %q},
  "display": {"date_format": "%%m/%%d"}
}`, dataDir, ymd(-180), ymd(180))
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	return home
}

func seedWorkouts(t *testing.T, home string) string {
	t.Helper()
	dataDir := filepath.Join(home, ".config", "c2", "data")
	recent := ymd(-2)
	older := ymd(-9)
	lines := []string{
		fmt.Sprintf(`{"id":1,"user_id":1,"date":"%s 08:00:00","distance":8000,"type":"rower","time":12000,"time_formatted":"20:00.0","stroke_rate":24,"heart_rate":{"average":140},"comments":"felt strong","workout":{"splits":[{"type":"distance","time":6000,"distance":4000,"stroke_rate":24,"heart_rate":{"average":138,"max":146}},{"type":"distance","time":6000,"distance":4000,"stroke_rate":25,"heart_rate":{"average":142,"max":150}}]}}`, recent),
		fmt.Sprintf(`{"id":2,"user_id":1,"date":"%s 07:30:00","distance":5000,"type":"rower","time":9000,"time_formatted":"15:00.0","stroke_rate":26}`, older),
	}
	if err := os.WriteFile(filepath.Join(dataDir, "workouts.jsonl"), []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "meta.json"), []byte(`{"schema_version":1,"created":"2026-01-01T00:00:00.000Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return dataDir
}

type result struct {
	stdout string
	stderr string
	failed bool
}

func run(t *testing.T, args ...string) result {
	t.Helper()
	return runWithStdin(t, "", args...)
}

func runWithStdin(t *testing.T, stdin string, args ...string) result {
	t.Helper()
	root := newRoot(build{version: "test", commit: "none", date: "unknown"})
	var out, errBuf bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&errBuf)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(args)
	err := root.Execute()
	return result{stdout: out.String(), stderr: errBuf.String(), failed: err != nil}
}

func decodeEnvelope(t *testing.T, raw string) map[string]any {
	t.Helper()
	var env map[string]any
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, raw)
	}
	return env
}

func TestBareCommandPrintsHelpAndSucceeds(t *testing.T) {
	testHome(t)
	got := run(t)
	if got.failed {
		t.Fatalf("expected success, stderr=%s", got.stderr)
	}
	if !strings.Contains(got.stdout, "Concept2 Logbook CLI") {
		t.Fatalf("stdout=%s", got.stdout)
	}
}

func TestUnknownCommandFails(t *testing.T) {
	testHome(t)
	got := run(t, "bogus")
	if !got.failed {
		t.Fatal("expected failure")
	}
}

func TestEnvelopesOnEmptyStore(t *testing.T) {
	testHome(t)
	for _, args := range [][]string{
		{"log", "--json"},
		{"status", "--json"},
		{"trend", "--json"},
		{"stats", "weekly", "--json"},
		{"note", "list", "--json"},
	} {
		got := run(t, args...)
		if got.failed {
			t.Fatalf("%v failed: %s", args, got.stderr)
		}
		env := decodeEnvelope(t, got.stdout)
		schema, _ := env["schema"].(string)
		if !strings.HasPrefix(schema, "c2.") || !strings.HasSuffix(schema, ".v1") {
			t.Fatalf("%v schema=%q", args, schema)
		}
		if _, ok := env["generated_at"].(string); !ok {
			t.Fatalf("%v missing generated_at", args)
		}
		if _, ok := env["data"]; !ok {
			t.Fatalf("%v missing data", args)
		}
	}
}

func TestLogRendersCommentsAndFiltersDates(t *testing.T) {
	home := testHome(t)
	seedWorkouts(t, home)

	got := run(t, "log")
	if got.failed || !strings.Contains(got.stdout, "— felt strong") {
		t.Fatalf("stdout=%q stderr=%q", got.stdout, got.stderr)
	}

	filtered := run(t, "log", "--from", ymd(-3))
	if filtered.failed {
		t.Fatal(filtered.stderr)
	}
	if strings.Count(strings.TrimSpace(filtered.stdout), "\n") != 0 {
		t.Fatalf("expected one row, got %q", filtered.stdout)
	}

	bad := run(t, "log", "--from", "07-01-2026")
	if !bad.failed || !strings.Contains(bad.stderr, "expected YYYY-MM-DD") {
		t.Fatalf("stderr=%q", bad.stderr)
	}
}

func TestShowResolvesLastAndIDs(t *testing.T) {
	home := testHome(t)
	seedWorkouts(t, home)

	last := run(t, "show", "last")
	if last.failed || !strings.Contains(last.stdout, "Id: 1") {
		t.Fatalf("stdout=%q stderr=%q", last.stdout, last.stderr)
	}
	if !strings.Contains(last.stdout, "Splits (") {
		t.Fatalf("expected splits, got %q", last.stdout)
	}

	byID := run(t, "show", "2")
	if byID.failed || !strings.Contains(byID.stdout, "Id: 2") {
		t.Fatalf("stdout=%q", byID.stdout)
	}

	missing := run(t, "show", "999")
	if !missing.failed || !strings.Contains(missing.stderr, "No workout with id 999") {
		t.Fatalf("stderr=%q", missing.stderr)
	}
}

func TestShowJSONEmitsFullDetail(t *testing.T) {
	home := testHome(t)
	seedWorkouts(t, home)

	got := run(t, "show", "last", "--json")
	if got.failed {
		t.Fatal(got.stderr)
	}
	env := decodeEnvelope(t, got.stdout)
	if env["schema"] != "c2.show.v1" {
		t.Fatalf("schema=%v", env["schema"])
	}
	data := env["data"].(map[string]any)
	for _, key := range []string{"workout", "raw", "splits", "split_shape", "stroke_summary", "notes"} {
		if _, ok := data[key]; !ok {
			t.Fatalf("missing %q", key)
		}
	}
	raw := data["raw"].(map[string]any)
	if raw["comments"] != "felt strong" {
		t.Fatalf("raw lost fields: %v", raw)
	}
}

func TestStatsSplitsHandlesWorkoutsWithoutSplitData(t *testing.T) {
	home := testHome(t)
	seedWorkouts(t, home)
	got := run(t, "stats", "splits", "2")
	if got.failed || !strings.Contains(got.stdout, "has no split data") {
		t.Fatalf("stdout=%q stderr=%q", got.stdout, got.stderr)
	}
}

func TestNoteRoundTripThroughCLI(t *testing.T) {
	home := testHome(t)
	seedWorkouts(t, home)

	added := run(t, "note", "add", "--type", "subjective", "--workout", "last", "--tags", "a,b", "felt slow early")
	if added.failed {
		t.Fatal(added.stderr)
	}
	id := strings.TrimSpace(added.stdout)
	if len(id) != 26 {
		t.Fatalf("expected ULID, got %q", id)
	}

	listed := run(t, "note", "list")
	if listed.failed || !strings.Contains(listed.stdout, "felt slow early") {
		t.Fatalf("stdout=%q", listed.stdout)
	}
	if !strings.Contains(listed.stdout, "#a #b") || !strings.Contains(listed.stdout, "w:1") {
		t.Fatalf("tags/workout missing: %q", listed.stdout)
	}

	shown := run(t, "note", "show", id)
	if shown.failed || !strings.Contains(shown.stdout, "felt slow early") {
		t.Fatalf("stdout=%q stderr=%q", shown.stdout, shown.stderr)
	}

	linked := run(t, "show", "last")
	if !strings.Contains(linked.stdout, "Notes:") {
		t.Fatalf("note not linked into show: %q", linked.stdout)
	}
}

func TestNoteAddReadsStdin(t *testing.T) {
	testHome(t)
	got := runWithStdin(t, "from stdin\n", "note", "add")
	if got.failed {
		t.Fatal(got.stderr)
	}
	listed := run(t, "note", "list")
	if !strings.Contains(listed.stdout, "from stdin") {
		t.Fatalf("stdout=%q", listed.stdout)
	}
}

func TestNoteRejectsEmptyBodyAndBadFlags(t *testing.T) {
	testHome(t)
	if got := runWithStdin(t, "   \n", "note", "add"); !got.failed || !strings.Contains(got.stderr, "note body is empty") {
		t.Fatalf("stderr=%q", got.stderr)
	}
	if got := run(t, "note", "add", "--type", "wrong", "x"); !got.failed || !strings.Contains(got.stderr, "--type must be one of") {
		t.Fatalf("stderr=%q", got.stderr)
	}
	if got := run(t, "note", "add", "--author", "llm", "x"); !got.failed || !strings.Contains(got.stderr, "--author must be one of") {
		t.Fatalf("stderr=%q", got.stderr)
	}
}

func TestInvalidDateRejectsBeforeStoreSideEffects(t *testing.T) {
	home := testHome(t)
	dataDir := filepath.Join(home, ".config", "c2", "data")
	if err := os.RemoveAll(dataDir); err != nil {
		t.Fatal(err)
	}

	got := run(t, "note", "add", "--date", "notadate", "body")
	if !got.failed || !strings.Contains(got.stderr, "invalid --date") {
		t.Fatalf("stderr=%q", got.stderr)
	}
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Fatal("store was created despite invalid date")
	}
}

func TestFailedWorkoutLinkLeavesNoStoreBehind(t *testing.T) {
	home := testHome(t)
	dataDir := filepath.Join(home, ".config", "c2", "data")
	if err := os.RemoveAll(dataDir); err != nil {
		t.Fatal(err)
	}

	got := run(t, "note", "add", "--workout", "999", "body")
	if !got.failed || !strings.Contains(got.stderr, "no workout matching") {
		t.Fatalf("stderr=%q", got.stderr)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "notes")); !os.IsNotExist(err) {
		t.Fatal("notes dir created despite failed link")
	}
}

func TestFirstCoachingWriteInitializesStore(t *testing.T) {
	home := testHome(t)
	dataDir := filepath.Join(home, ".config", "c2", "data")
	if err := os.RemoveAll(dataDir); err != nil {
		t.Fatal(err)
	}

	if got := run(t, "note", "add", "first note"); got.failed {
		t.Fatal(got.stderr)
	}
	for _, sub := range []string{"meta.json", "strokes", filepath.Join("notes", "archive"), "reports"} {
		if _, err := os.Stat(filepath.Join(dataDir, sub)); err != nil {
			t.Fatalf("missing %s: %v", sub, err)
		}
	}
}

func TestBackdatedNotesCompactViaDataCompact(t *testing.T) {
	testHome(t)
	if got := run(t, "note", "add", "--date", ymd(-30), "old note"); got.failed {
		t.Fatal(got.stderr)
	}
	if got := run(t, "note", "add", "recent note"); got.failed {
		t.Fatal(got.stderr)
	}

	compacted := run(t, "data", "compact")
	if compacted.failed || !strings.Contains(compacted.stdout, "Compacted 1 note") {
		t.Fatalf("stdout=%q stderr=%q", compacted.stdout, compacted.stderr)
	}

	listed := run(t, "note", "list")
	if !strings.Contains(listed.stdout, "old note") || !strings.Contains(listed.stdout, "recent note") {
		t.Fatalf("compaction lost notes: %q", listed.stdout)
	}
	if again := run(t, "data", "compact"); !strings.Contains(again.stdout, "Nothing to compact.") {
		t.Fatalf("stdout=%q", again.stdout)
	}
}

func TestDocumentsRoundTrip(t *testing.T) {
	testHome(t)

	if got := runWithStdin(t, "# my plan\n", "plan", "set", "-"); got.failed {
		t.Fatal(got.stderr)
	}
	if got := run(t, "plan", "show"); got.failed || !strings.Contains(got.stdout, "# my plan") {
		t.Fatalf("stdout=%q stderr=%q", got.stdout, got.stderr)
	}
	if got := runWithStdin(t, "  \n", "plan", "set", "-"); !got.failed || !strings.Contains(got.stderr, "refusing to save an empty plan") {
		t.Fatalf("stderr=%q", got.stderr)
	}

	if got := runWithStdin(t, "# playbook\n", "playbook", "set", "-"); got.failed {
		t.Fatal(got.stderr)
	}
	if got := run(t, "playbook", "show"); got.failed || !strings.Contains(got.stdout, "# playbook") {
		t.Fatalf("stdout=%q", got.stdout)
	}

	date := ymd(-1)
	if got := runWithStdin(t, "coach says hi\n", "narrative", "add", date, "-"); got.failed {
		t.Fatal(got.stderr)
	}
	if got := run(t, "narrative", "show"); got.failed || !strings.Contains(got.stdout, "coach says hi") {
		t.Fatalf("stdout=%q", got.stdout)
	}
	if got := run(t, "narrative", "list"); got.failed || !strings.Contains(got.stdout, date) {
		t.Fatalf("stdout=%q", got.stdout)
	}
	if got := run(t, "narrative", "add", "07-01-2026", "-"); !got.failed || !strings.Contains(got.stderr, "invalid date") {
		t.Fatalf("stderr=%q", got.stderr)
	}
}

func TestCoachingReadsRejectForeignDirectories(t *testing.T) {
	home := testHome(t)
	foreign := filepath.Join(home, "foreign")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign, "novel.docx"), []byte("chapter one"), 0o644); err != nil {
		t.Fatal(err)
	}
	pointConfigAt(t, home, foreign)

	for _, args := range [][]string{{"note", "list"}, {"plan", "show"}, {"narrative", "list"}} {
		got := run(t, args...)
		if !got.failed || !strings.Contains(got.stderr, "is not a c2 data store") {
			t.Fatalf("%v stderr=%q", args, got.stderr)
		}
	}
}

func TestForeignDataDirGivesCleanErrors(t *testing.T) {
	home := testHome(t)
	foreign := filepath.Join(home, "foreign")
	if err := os.MkdirAll(foreign, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(foreign, "novel.docx"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	pointConfigAt(t, home, foreign)

	for _, args := range [][]string{
		{"data", "info"},
		{"log"},
		{"status"},
		{"trend"},
		{"export"},
		{"show", "last"},
		{"stats", "weekly"},
	} {
		got := run(t, args...)
		if !got.failed || !strings.Contains(got.stderr, "is not a c2 data store") {
			t.Fatalf("%v stderr=%q", args, got.stderr)
		}
	}
}

func pointConfigAt(t *testing.T, home, dataDir string) {
	t.Helper()
	cfg := fmt.Sprintf(`{
  "data_dir": %q,
  "api": {"base_url": "https://log.concept2.com", "token": "tok"},
  "goal": {"target_meters": 1000000, "start_date": %q, "end_date": %q},
  "display": {"date_format": "%%m/%%d"}
}`, dataDir, ymd(-180), ymd(180))
	if err := os.WriteFile(filepath.Join(home, ".config", "c2", "config.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDataInfoReportsTheStore(t *testing.T) {
	home := testHome(t)
	seedWorkouts(t, home)

	got := run(t, "data", "info")
	if got.failed {
		t.Fatal(got.stderr)
	}
	for _, want := range []string{"Data store:", "Schema version: 1", "Workouts: 2"} {
		if !strings.Contains(got.stdout, want) {
			t.Fatalf("missing %q in %q", want, got.stdout)
		}
	}

	asJSON := run(t, "data", "info", "--json")
	env := decodeEnvelope(t, asJSON.stdout)
	if env["schema"] != "c2.data.info.v1" {
		t.Fatalf("schema=%v", env["schema"])
	}
	data := env["data"].(map[string]any)
	if data["state"] != "store" || data["workouts"].(float64) != 2 {
		t.Fatalf("data=%v", data)
	}
}

func TestDataDoctorReportsCorruption(t *testing.T) {
	home := testHome(t)
	dataDir := seedWorkouts(t, home)

	if got := run(t, "data", "doctor"); got.failed {
		t.Fatalf("clean store should pass: %s", got.stderr)
	}

	if err := os.WriteFile(filepath.Join(dataDir, "workouts.jsonl"), []byte("not json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := run(t, "data", "doctor")
	if !got.failed || !strings.Contains(got.stderr, "not valid JSON") {
		t.Fatalf("stderr=%q", got.stderr)
	}
}

func TestDataMoveRelocatesStoreAndUpdatesConfig(t *testing.T) {
	home := testHome(t)
	seedWorkouts(t, home)
	target := filepath.Join(home, "moved")

	got := run(t, "data", "move", target)
	if got.failed {
		t.Fatal(got.stderr)
	}
	if !strings.Contains(got.stdout, "Config updated: data_dir = ") {
		t.Fatalf("stdout=%q", got.stdout)
	}
	if _, err := os.Stat(filepath.Join(target, "workouts.jsonl")); err != nil {
		t.Fatal(err)
	}

	info := run(t, "data", "info")
	if !strings.Contains(info.stdout, string(filepath.Separator)+"moved") {
		t.Fatalf("config not updated: %q", info.stdout)
	}
	if !filepath.IsAbs(configuredDataDir(t)) {
		t.Fatalf("data_dir must be absolute, got %q", configuredDataDir(t))
	}
}

func configuredDataDir(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".config", "c2", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		DataDir string `json:"data_dir"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatal(err)
	}
	return cfg.DataDir
}

func TestDataMoveRefusesNestedTargets(t *testing.T) {
	home := testHome(t)
	dataDir := seedWorkouts(t, home)

	got := run(t, "data", "move", filepath.Join(dataDir, "inner"))
	if !got.failed || !strings.Contains(got.stderr, "must not be inside") {
		t.Fatalf("stderr=%q", got.stderr)
	}
}

func TestExportRejectsInvalidDatesButAllowsEmptyRanges(t *testing.T) {
	home := testHome(t)
	seedWorkouts(t, home)

	bad := run(t, "export", "--from", "nope")
	if !bad.failed || !strings.Contains(bad.stderr, "expected YYYY-MM-DD") {
		t.Fatalf("stderr=%q", bad.stderr)
	}

	empty := run(t, "export", "--from", ymd(365))
	if empty.failed {
		t.Fatal(empty.stderr)
	}
	if !strings.Contains(empty.stderr, "No workouts match") {
		t.Fatalf("stderr=%q", empty.stderr)
	}
	if !strings.HasPrefix(empty.stdout, "id,date,distance") {
		t.Fatalf("expected header, got %q", empty.stdout)
	}

	bogus := run(t, "export", "-f", "xml")
	if !bogus.failed || !strings.Contains(bogus.stderr, "Unsupported format") {
		t.Fatalf("stderr=%q", bogus.stderr)
	}
}

func TestExportJSONEmitsVersionedEnvelope(t *testing.T) {
	home := testHome(t)
	seedWorkouts(t, home)

	got := run(t, "export", "-f", "json")
	if got.failed {
		t.Fatal(got.stderr)
	}
	env := decodeEnvelope(t, got.stdout)
	if env["schema"] != "c2.export.v1" {
		t.Fatalf("schema=%v", env["schema"])
	}
	data := env["data"].(map[string]any)
	if data["count"].(float64) != 2 || len(data["workouts"].([]any)) != 2 {
		t.Fatalf("data=%v", data)
	}
}

func TestSyncRequiresToken(t *testing.T) {
	home := testHome(t)
	cfg := fmt.Sprintf(`{"data_dir": %q, "api": {"base_url": "https://log.concept2.com", "token": ""}}`,
		filepath.Join(home, ".config", "c2", "data"))
	if err := os.WriteFile(filepath.Join(home, ".config", "c2", "config.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	got := run(t, "sync")
	if !got.failed || !strings.Contains(got.stderr, "No API token configured") {
		t.Fatalf("stderr=%q", got.stderr)
	}
}

func TestStatusRequiresGoalDates(t *testing.T) {
	home := testHome(t)
	cfg := fmt.Sprintf(`{"data_dir": %q, "api": {"base_url": "x", "token": "tok"}}`,
		filepath.Join(home, ".config", "c2", "data"))
	if err := os.WriteFile(filepath.Join(home, ".config", "c2", "config.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	got := run(t, "status")
	if !got.failed || !strings.Contains(got.stderr, "Goal dates not configured") {
		t.Fatalf("stderr=%q", got.stderr)
	}
}
