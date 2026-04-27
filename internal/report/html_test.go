package report

import (
	"strings"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/model"
)

func TestHTMLIncludesMajorReportSections(t *testing.T) {
	html, err := HTML(sampleWorkouts(), config.GoalConfig{
		TargetMeters: 1000000,
		StartDate:    "2026-01-01",
		EndDate:      "2026-12-31",
	}, 8, time.Date(2026, 4, 15, 12, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatalf("HTML() error = %v", err)
	}

	for _, want := range []string{
		"Goal Progress",
		"Weekly Volume",
		"Weekly Trends",
		"Recent Workouts",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("HTML() missing %q in:\n%s", want, html)
		}
	}
}

func TestHTMLEscapesWorkoutComments(t *testing.T) {
	workouts := sampleWorkouts()
	workouts[0].Comments = `<script>alert("x")</script>`

	html, err := HTML(workouts, config.GoalConfig{
		TargetMeters: 1000000,
		StartDate:    "2026-01-01",
		EndDate:      "2026-12-31",
	}, 8, time.Date(2026, 4, 15, 12, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatalf("HTML() error = %v", err)
	}

	if strings.Contains(html, `<script>alert("x")</script>`) {
		t.Fatalf("HTML() included raw script comment:\n%s", html)
	}
	if !strings.Contains(html, `&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;`) {
		t.Fatalf("HTML() missing escaped script comment:\n%s", html)
	}
}

func TestHTMLAnnotatesSameDayRecentWorkouts(t *testing.T) {
	slowSPM := 20
	hardSPM := 30
	hardHR := 145
	workouts := []model.Workout{
		{
			ID:            1,
			UserID:        1,
			Date:          "2026-04-09 07:00:00",
			Distance:      1000,
			Type:          "rower",
			Time:          4000,
			TimeFormatted: "6:40.0",
			StrokeRate:    &slowSPM,
		},
		{
			ID:            2,
			UserID:        1,
			Date:          "2026-04-09 07:30:00",
			Distance:      5000,
			Type:          "rower",
			Time:          15000,
			TimeFormatted: "25:00.0",
			StrokeRate:    &hardSPM,
			HeartRate:     &model.HeartRate{Average: &hardHR},
		},
		{
			ID:            3,
			UserID:        1,
			Date:          "2026-04-09 08:00:00",
			Distance:      1000,
			Type:          "rower",
			Time:          4200,
			TimeFormatted: "7:00.0",
			StrokeRate:    &slowSPM,
		},
	}

	html, err := HTML(workouts, config.GoalConfig{
		TargetMeters: 1000000,
		StartDate:    "2026-01-01",
		EndDate:      "2026-12-31",
	}, 8, time.Date(2026, 4, 15, 12, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatalf("HTML() error = %v", err)
	}

	for _, want := range []string{
		`<span style="font-size:10px;">(warmup)</span>`,
		`<span style="font-size:10px; color:#3fb950;">(hard)</span>`,
		`<span style="font-size:10px;">(cooldown)</span>`,
		`<tr style="color:#8b949e;">`,
		`<td class="r" style="color:#3fb950;">2:30.0</td>`,
		`<td class="r" style="color:#f85149;">145</td>`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("HTML() missing %q in:\n%s", want, html)
		}
	}

	warmupIdx := strings.Index(html, "(warmup)")
	hardIdx := strings.Index(html, "(hard)")
	cooldownIdx := strings.Index(html, "(cooldown)")
	if !(warmupIdx < hardIdx && hardIdx < cooldownIdx) {
		t.Fatalf("recent workout annotations are out of order: warmup=%d hard=%d cooldown=%d", warmupIdx, hardIdx, cooldownIdx)
	}
}

func sampleWorkouts() []model.Workout {
	spm := 24
	hr := 140
	return []model.Workout{
		{
			ID:            1,
			UserID:        1,
			Date:          "2026-04-09 07:00:00",
			Distance:      5000,
			Type:          "rower",
			Time:          17155,
			TimeFormatted: "28:35.4",
			StrokeRate:    &spm,
			HeartRate:     &model.HeartRate{Average: &hr},
			Comments:      "steady row",
		},
		{
			ID:            2,
			UserID:        1,
			Date:          "2026-04-11 09:14:00",
			Distance:      3000,
			Type:          "rower",
			Time:          8626,
			TimeFormatted: "20:22.6",
			StrokeRate:    &spm,
			HeartRate:     &model.HeartRate{Average: &hr},
			Comments:      "intervals",
		},
	}
}
