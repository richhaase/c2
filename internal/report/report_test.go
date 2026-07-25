package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/jsonx"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/paths"
)

func reportFixture(t *testing.T) (config.Config, paths.DataPaths, []models.Workout, time.Time) {
	t.Helper()
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.Local)
	p := paths.For(t.TempDir())
	if err := os.MkdirAll(p.ReportsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Plan, []byte("# Plan\n\n## Current\n\nHold <145 bpm.\n\n## Later\n\nBuild volume.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.NarrativeFile("2026-07-19"), []byte("## Focus\n\n- Stay <smooth>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	n := notes.Record{
		ID:     "01K0NOTE000000000000000000",
		Date:   "2026-07-19T12:00:00-06:00",
		Type:   "observation",
		Body:   "Calm <strong> finish",
		Author: "coach",
	}
	if err := notes.Write(p, n); err != nil {
		t.Fatal(err)
	}
	hr := 140
	spm := 24
	splitDistance := 1000
	workouts := []models.Workout{
		{
			ID:            1,
			Date:          "2026-07-18 08:00:00",
			Distance:      5000,
			Time:          9000,
			TimeFormatted: "15:00.0",
			StrokeRate:    &spm,
			HeartRate:     &models.HeartRate{Average: &hr},
		},
		{
			ID:            2,
			Date:          "2026-07-19 08:00:00",
			Distance:      2000,
			Time:          3600,
			TimeFormatted: "6:00.0",
			Workout: &models.WorkoutDetail{Splits: []models.WorkoutSplit{
				{Time: 1800, Distance: &splitDistance},
				{Time: 1800, Distance: &splitDistance},
			}},
		},
		{
			ID:            3,
			Date:          "2026-07-19 09:00:00",
			Distance:      1000,
			Time:          1750,
			TimeFormatted: "2:55.0",
			Workout: &models.WorkoutDetail{Splits: []models.WorkoutSplit{
				{Time: 1750, Distance: &splitDistance},
			}},
		},
	}
	cfg := config.Default()
	cfg.Goal.TargetMeters = 1_000_000
	cfg.Goal.StartDate = "2026-01-01"
	cfg.Goal.EndDate = "2026-12-31"
	return cfg, p, workouts, now
}

func TestBuildProducesEscapedHTMLAndStructuredPayload(t *testing.T) {
	cfg, p, workouts, now := reportFixture(t)
	result, err := Build(cfg, p, workouts, now, 12)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"<title>Rowing Progress — 2026</title>",
		"Coach's Report &mdash; 2026-07-19",
		"Stay &lt;smooth&gt;",
		"Calm &lt;strong&gt; finish",
		"Hold &lt;145 bpm.",
		"Weekly Volume",
		"Year-End Projection",
	} {
		if !strings.Contains(result.HTML, want) {
			t.Errorf("HTML missing %q", want)
		}
	}
	for _, unwanted := range []string{"Stay <smooth>", "Calm <strong> finish", "Hold <145 bpm."} {
		if strings.Contains(result.HTML, unwanted) {
			t.Errorf("HTML contains unescaped %q", unwanted)
		}
	}

	if result.Payload.Period.Weeks != 12 || result.Payload.Period.To == nil || *result.Payload.Period.To != "2026-07-19" {
		t.Fatalf("period = %#v", result.Payload.Period)
	}
	if len(result.Payload.RecentWorkouts) != 3 {
		t.Fatalf("recent workouts = %d", len(result.Payload.RecentWorkouts))
	}
	if result.Payload.RecentWorkouts[0].ID != 3 || result.Payload.RecentWorkouts[2].ID != 1 {
		t.Fatalf("recent workout order = %d, %d, %d",
			result.Payload.RecentWorkouts[0].ID,
			result.Payload.RecentWorkouts[1].ID,
			result.Payload.RecentWorkouts[2].ID)
	}
	if result.Payload.LatestSplits == nil || result.Payload.LatestSplits.WorkoutID != 2 {
		t.Fatalf("latest splits = %#v", result.Payload.LatestSplits)
	}
	if result.Payload.Narrative == nil || result.Payload.Narrative.Date != "2026-07-19" {
		t.Fatalf("narrative = %#v", result.Payload.Narrative)
	}
	if len(result.Payload.Notes) != 1 || result.Payload.Notes[0].ID != "01K0NOTE000000000000000000" {
		t.Fatalf("notes = %#v", result.Payload.Notes)
	}
	if result.Payload.PlanExcerpt == nil || !strings.Contains(*result.Payload.PlanExcerpt, "Hold <145 bpm.") {
		t.Fatalf("plan excerpt = %#v", result.Payload.PlanExcerpt)
	}
}

func TestPayloadJSONFieldOrder(t *testing.T) {
	cfg, p, workouts, now := reportFixture(t)
	result, err := Build(cfg, p, workouts, now, 4)
	if err != nil {
		t.Fatal(err)
	}
	out, err := jsonx.Compact(result.Payload)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	wantPrefix := `{"period":`
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("payload starts %q, want %q", got[:min(len(got), len(wantPrefix))], wantPrefix)
	}
	keys := []string{
		`"period":`,
		`"summary":`,
		`"goal":`,
		`"projection":`,
		`"weekly":`,
		`"recent_workouts":`,
		`"latest_splits":`,
		`"narrative":`,
		`"notes":`,
		`"plan_excerpt":`,
	}
	last := -1
	for _, key := range keys {
		at := strings.Index(got, key)
		if at <= last {
			t.Fatalf("key %s out of order in %s", key, got)
		}
		last = at
	}
}

func TestBuildWithoutCoachingContentUsesEmptyCollections(t *testing.T) {
	cfg := config.Default()
	cfg.Goal.StartDate = "2026-01-01"
	cfg.Goal.EndDate = "2026-12-31"
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.Local)
	p := paths.For(filepath.Join(t.TempDir(), "missing"))

	result, err := Build(cfg, p, nil, now, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Payload.Weekly) != 2 {
		t.Fatalf("weekly = %d", len(result.Payload.Weekly))
	}
	if result.Payload.RecentWorkouts == nil {
		t.Fatal("recent workouts is nil")
	}
	if result.Payload.Notes == nil {
		t.Fatal("notes is nil")
	}
	if result.Payload.LatestSplits != nil || result.Payload.Narrative != nil || result.Payload.PlanExcerpt != nil {
		t.Fatalf("unexpected optional content: %#v", result.Payload)
	}
}
