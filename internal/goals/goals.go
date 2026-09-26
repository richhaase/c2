package goals

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/stats"
)

type Goal struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Kind        string  `json:"kind"`
	Target      float64 `json:"target"`
	Equipment   string  `json:"equipment"`
	From        string  `json:"from,omitempty"`
	To          string  `json:"to,omitempty"`
	MinDistance int     `json:"min_distance,omitempty"`
	Effort      string  `json:"effort"`
	Archived    bool    `json:"archived"`
}

type Evidence struct {
	WorkoutID   int64   `json:"workout_id"`
	Date        string  `json:"date"`
	Distance    int     `json:"distance"`
	Pace        float64 `json:"pace_500m_seconds"`
	WorkoutType string  `json:"workout_type"`
	Continuity  string  `json:"continuity"`
}

type Progress struct {
	Goal               Goal                  `json:"goal"`
	Value              *float64              `json:"value"`
	Remaining          *float64              `json:"remaining"`
	Achieved           bool                  `json:"achieved"`
	AchievedOn         string                `json:"achieved_on,omitempty"`
	Evidence           *Evidence             `json:"evidence"`
	QualifyingWorkouts int                   `json:"qualifying_workouts"`
	UnknownEfforts     int                   `json:"unknown_efforts"`
	CoverageFrom       string                `json:"coverage_from,omitempty"`
	CoverageTo         string                `json:"coverage_to,omitempty"`
	Volume             *stats.GoalProgress   `json:"volume,omitempty"`
	Projection         *stats.GoalProjection `json:"projection,omitempty"`
}

func Validate(g Goal) error {
	if strings.TrimSpace(g.ID) == "" || strings.TrimSpace(g.Name) == "" {
		return fmt.Errorf("Goal ID and name are required.")
	}
	if g.Kind != "volume" && g.Kind != "distance" && g.Kind != "pace" {
		return fmt.Errorf("Goal kind must be volume, distance, or pace.")
	}
	if math.IsNaN(g.Target) || math.IsInf(g.Target, 0) || g.Target <= 0 || g.Target > 1e12 {
		return fmt.Errorf("Goal target must be positive and at most 1e12.")
	}
	if g.Kind != "pace" && math.Trunc(g.Target) != g.Target {
		return fmt.Errorf("Distance targets must be whole meters.")
	}
	if g.Equipment == "" {
		return fmt.Errorf("Goal equipment is required (use all for any equipment).")
	}
	for _, date := range []string{g.From, g.To} {
		if date != "" && !models.IsValidYMD(date) {
			return fmt.Errorf("Invalid goal date %q; expected YYYY-MM-DD.", date)
		}
	}
	if g.From != "" && g.To != "" && g.From > g.To {
		return fmt.Errorf("Goal end date must not precede its start date.")
	}
	if g.Effort != "workout" && g.Effort != "continuous" {
		return fmt.Errorf("Effort must be workout or continuous.")
	}
	if g.MinDistance < 0 || (g.Kind != "pace" && g.MinDistance != 0) {
		return fmt.Errorf("Minimum distance is a nonnegative pace-goal qualification.")
	}
	return nil
}

func Continuity(w models.Workout) string {
	if models.IsIntervalWorkout(w) {
		return "interval"
	}
	switch w.WorkoutType {
	case "JustRow", "FixedDistanceSplits", "FixedTimeSplits", "FixedCalorie", "FixedWattMinute":
		return "continuous"
	default:
		return "unknown"
	}
}

func paceMeetsTarget(value, target float64) bool {
	roundingTolerance := 4 * (math.Nextafter(target, math.Inf(1)) - target)
	return value <= target+roundingTolerance
}

func Evaluate(g Goal, workouts []models.Workout, now time.Time) Progress {
	return evaluateOrdered(g, orderedWorkouts(workouts), now)
}

func orderedWorkouts(workouts []models.Workout) []models.Workout {
	ordered := slices.Clone(workouts)
	slices.SortFunc(ordered, func(a, b models.Workout) int {
		if c := strings.Compare(a.Date, b.Date); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return ordered
}

func evaluateOrdered(g Goal, ordered []models.Workout, now time.Time) Progress {
	p := Progress{Goal: g}
	eligible := make([]models.Workout, 0)
	for _, w := range ordered {
		day := models.CalendarDay(w)
		if !models.IsValidYMD(day) || w.Distance <= 0 || day > now.Format("2006-01-02") {
			continue
		}
		if p.CoverageFrom == "" {
			p.CoverageFrom = day
		}
		p.CoverageTo = day
		if g.From != "" && day < g.From || g.To != "" && day > g.To {
			continue
		}
		if g.Equipment != "all" && w.Type != g.Equipment {
			continue
		}
		continuity := Continuity(w)
		if g.Effort == "continuous" && continuity != "continuous" {
			if continuity == "unknown" {
				p.UnknownEfforts++
			}
			continue
		}
		if w.Distance < g.MinDistance || g.Kind == "pace" && w.Time <= 0 {
			continue
		}
		p.QualifyingWorkouts++
		eligible = append(eligible, w)
		value := float64(w.Distance)
		if g.Kind == "pace" {
			value = models.Pace500mSeconds(w)
		}
		if g.Kind == "volume" {
			if p.Value == nil {
				p.Value = new(float64)
			}
			*p.Value += value
		} else if p.Value == nil || g.Kind == "pace" && value < *p.Value || g.Kind == "distance" && value > *p.Value {
			p.Value = new(value)
			p.Evidence = &Evidence{WorkoutID: w.ID, Date: w.Date, Distance: w.Distance, Pace: models.Pace500mSeconds(w), WorkoutType: w.WorkoutType, Continuity: continuity}
		}
		if !p.Achieved && p.Value != nil && (g.Kind == "pace" && paceMeetsTarget(*p.Value, g.Target) || g.Kind != "pace" && *p.Value >= g.Target) {
			p.Achieved = true
			p.AchievedOn = day
		}
	}
	if g.Kind == "volume" && p.Value == nil {
		p.Value = new(float64)
	}
	if p.Value != nil {
		remaining := g.Target - *p.Value
		if g.Kind == "pace" {
			remaining = *p.Value - g.Target
		}
		if p.Achieved {
			remaining = 0
		}
		p.Remaining = new(math.Max(remaining, 0))
	}
	if g.Kind == "volume" && g.From != "" && g.To != "" {
		cfg := config.Config{Goal: config.GoalConfig{TargetMeters: int(g.Target), StartDate: g.From, EndDate: g.To}, Location: now.Location()}
		v, err := stats.ComputeGoalProgress(eligible, cfg, now)
		if err == nil {
			p.Volume = &v
			end, _ := time.Parse("2006-01-02", g.To)
			today := now.Format("2006-01-02")
			if today >= g.From && today <= g.To {
				projection := stats.ProjectGoal(v, end.AddDate(0, 0, 1), now)
				p.Projection = &projection
			}
		}
	}
	return p
}

func EvaluateAll(items []Goal, workouts []models.Workout, now time.Time) []Progress {
	ordered := orderedWorkouts(workouts)
	result := make([]Progress, 0, len(items))
	for _, g := range items {
		if !g.Archived {
			result = append(result, evaluateOrdered(g, ordered, now))
		}
	}
	return result
}
