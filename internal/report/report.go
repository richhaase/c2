package report

import (
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/stats"
)

func Build(cfg config.Config, p paths.DataPaths, workouts []models.Workout, now time.Time, weeks int) (Result, error) {
	loc := cfg.Location
	if loc == nil {
		loc = time.Local
	}
	now = now.In(loc)
	goal, err := stats.ComputeGoalProgress(workouts, cfg, now)
	if err != nil {
		return Result{}, err
	}
	end, err := time.Parse("2006-01-02", cfg.Goal.EndDate)
	if err != nil {
		return Result{}, err
	}
	start, _ := time.Parse("2006-01-02", cfg.Goal.StartDate)
	projection := stats.ProjectGoal(goal, start, end.AddDate(0, 0, 1), now)
	summaries := stats.BuildWeekSummaries(workouts, now, weeks)
	cutoff := stats.MondayOf(now).AddDate(0, 0, -(weeks-1)*7)
	windowed := stats.WorkoutsInRange(workouts, cutoff, now)
	coaching, err := gatherCoaching(p, now)
	if err != nil {
		return Result{}, err
	}
	return Result{
		HTML:    buildHTML(goal, projection, summaries, workouts, windowed, reportRecentWorkouts, coaching, now),
		Payload: buildReportPayload(workouts, windowed, weeks, goal, projection, summaries, coaching),
	}, nil
}
