package report

import (
	"math"
	"slices"
	"sort"

	"github.com/richhaase/c2/internal/analysis"
	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/stats"
)

const reportRecentWorkouts = 10

func sortedByDateDesc(workouts []models.Workout) []models.Workout {
	sorted := slices.Clone(workouts)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[j].Date < sorted[i].Date
	})
	return sorted
}

func roundHalfUp(v float64) float64 {
	return math.Floor(v + 0.5)
}

type Period struct {
	Weeks int     `json:"weeks"`
	To    *string `json:"to"`
}

type Summary struct {
	TotalMeters        int      `json:"total_meters"`
	Sessions           int      `json:"sessions"`
	AvgPace500mSeconds *float64 `json:"avg_pace_500m_seconds"`
	AvgHR              *int     `json:"avg_hr"`
}

type Splits struct {
	WorkoutID  int64               `json:"workout_id"`
	Date       string              `json:"date"`
	SplitShape analysis.Shape      `json:"split_shape"`
	Splits     []analysis.SplitRow `json:"splits"`
}

type Payload struct {
	Period         Period                  `json:"period"`
	Summary        Summary                 `json:"summary"`
	Goal           stats.GoalProgress      `json:"goal"`
	Projection     stats.GoalProjection    `json:"projection"`
	Weekly         []stats.WeekSummaryData `json:"weekly"`
	RecentWorkouts []display.WorkoutOutput `json:"recent_workouts"`
	LatestSplits   *Splits                 `json:"latest_splits"`
	Narrative      *Narrative              `json:"narrative"`
	Notes          []notes.Record          `json:"notes"`
	PlanExcerpt    *string                 `json:"plan_excerpt"`
}

type Result struct {
	HTML    string
	Payload Payload
}

func avgPaceForWorkouts(workouts []models.Workout) float64 {
	sum := 0.0
	count := 0
	for _, w := range workouts {
		if p := models.Pace500mSeconds(w); p > 0 {
			sum += p
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func avgHRForWorkouts(workouts []models.Workout) int {
	sum := 0
	count := 0
	for _, w := range workouts {
		if w.HeartRate != nil && w.HeartRate.Average != nil && *w.HeartRate.Average > 0 {
			sum += *w.HeartRate.Average
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return int(roundHalfUp(float64(sum) / float64(count)))
}

type activityData struct {
	summary        Summary
	weekly         []stats.WeekSummaryData
	recentWorkouts []display.WorkoutOutput
	latestSplits   *Splits
	latestDate     *string
}

func buildActivityData(workouts, windowed []models.Workout, summaries []stats.WeekSummary) activityData {
	sorted := sortedByDateDesc(workouts)

	var latestDate *string
	var splitsSource *models.Workout
	if len(sorted) > 0 {
		day := models.CalendarDay(sorted[0])
		latestDate = &day

		for _, w := range sorted {
			if models.CalendarDay(w) != day {
				continue
			}
			if w.Workout == nil || len(w.Workout.Splits) == 0 {
				continue
			}
			if splitsSource == nil || w.Distance > splitsSource.Distance {
				splitsSource = &w
			}
		}
	}

	var latestSplits *Splits
	if splitsSource != nil {
		rows := analysis.SplitTable(*splitsSource)
		if len(rows) > 0 {
			latestSplits = &Splits{
				WorkoutID:  splitsSource.ID,
				Date:       splitsSource.Date,
				SplitShape: analysis.SplitShape(rows),
				Splits:     rows,
			}
		}
	}

	summary := Summary{
		Sessions: stats.SessionCount(windowed),
	}
	if pace := math.Round(avgPaceForWorkouts(windowed)*10) / 10; pace != 0 {
		summary.AvgPace500mSeconds = &pace
	}
	if hr := avgHRForWorkouts(windowed); hr != 0 {
		summary.AvgHR = &hr
	}

	limit := min(reportRecentWorkouts, len(sorted))
	recent := make([]display.WorkoutOutput, 0, limit)
	for _, w := range sorted[:limit] {
		recent = append(recent, display.WorkoutOutputOf(w))
	}

	return activityData{summary: summary, weekly: stats.WeekSummariesDataOf(summaries), recentWorkouts: recent, latestSplits: latestSplits, latestDate: latestDate}
}

func buildReportPayload(
	workouts []models.Workout,
	windowed []models.Workout,
	weeks int,
	goal stats.GoalProgress,
	projection stats.GoalProjection,
	summaries []stats.WeekSummary,
	coaching coachingContent,
) Payload {
	activity := buildActivityData(workouts, windowed, summaries)
	activity.summary.TotalMeters = goal.TotalMeters
	records := coaching.notes
	if records == nil {
		records = []notes.Record{}
	}
	return Payload{
		Period:         Period{Weeks: weeks, To: activity.latestDate},
		Summary:        activity.summary,
		Goal:           goal,
		Projection:     projection,
		Weekly:         activity.weekly,
		RecentWorkouts: activity.recentWorkouts,
		LatestSplits:   activity.latestSplits,
		Narrative:      coaching.narrative,
		Notes:          records,
		PlanExcerpt:    coaching.planExcerpt,
	}
}
