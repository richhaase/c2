package stats

import (
	"testing"

	"github.com/richhaase/c2/internal/model"
)

func makeSessionWorkout(id int, date string, distance int) model.Workout {
	return model.Workout{
		ID:            id,
		UserID:        1,
		Date:          date,
		Distance:      distance,
		Type:          "rower",
		Time:          int(float64(distance)*3.5 + 0.5),
		TimeFormatted: "0:00.0",
	}
}

func TestGroupIntoSessionsGroupsSameDayWorkouts(t *testing.T) {
	workouts := []model.Workout{
		makeSessionWorkout(1, "2026-03-07 09:21:00", 1000),
		makeSessionWorkout(2, "2026-03-07 09:45:00", 2500),
		makeSessionWorkout(3, "2026-03-07 09:53:00", 1000),
	}
	sessions := GroupIntoSessions(workouts)
	if len(sessions) != 1 {
		t.Fatalf("len(GroupIntoSessions()) = %d, want 1", len(sessions))
	}
	if sessions[0].Date != "2026-03-07" {
		t.Fatalf("session date = %q, want 2026-03-07", sessions[0].Date)
	}
	if sessions[0].TotalDistance != 4500 {
		t.Fatalf("session total distance = %d, want 4500", sessions[0].TotalDistance)
	}
	if len(sessions[0].Workouts) != 3 {
		t.Fatalf("session workouts = %d, want 3", len(sessions[0].Workouts))
	}
}

func TestGroupIntoSessionsKeepsDifferentDaysSeparate(t *testing.T) {
	workouts := []model.Workout{
		makeSessionWorkout(1, "2026-03-05 10:00:00", 5000),
		makeSessionWorkout(2, "2026-03-07 09:00:00", 5000),
	}
	sessions := GroupIntoSessions(workouts)
	if len(sessions) != 2 {
		t.Fatalf("len(GroupIntoSessions()) = %d, want 2", len(sessions))
	}
	if sessions[0].Date != "2026-03-05" || sessions[1].Date != "2026-03-07" {
		t.Fatalf("session dates = %q, %q; want 2026-03-05, 2026-03-07", sessions[0].Date, sessions[1].Date)
	}
}

func TestGroupIntoSessionsSortsSessionsByDate(t *testing.T) {
	workouts := []model.Workout{
		makeSessionWorkout(2, "2026-03-07 09:00:00", 5000),
		makeSessionWorkout(1, "2026-03-05 10:00:00", 5000),
	}
	sessions := GroupIntoSessions(workouts)
	if sessions[0].Date != "2026-03-05" || sessions[1].Date != "2026-03-07" {
		t.Fatalf("session dates = %q, %q; want sorted ascending", sessions[0].Date, sessions[1].Date)
	}
}

func TestGroupIntoSessionsSortsWorkoutsWithinSession(t *testing.T) {
	workouts := []model.Workout{
		makeSessionWorkout(2, "2026-03-07 09:53:00", 1000),
		makeSessionWorkout(1, "2026-03-07 09:21:00", 1000),
		makeSessionWorkout(3, "2026-03-07 09:45:00", 1000),
	}
	sessions := GroupIntoSessions(workouts)
	got := []int{sessions[0].Workouts[0].ID, sessions[0].Workouts[1].ID, sessions[0].Workouts[2].ID}
	want := []int{1, 3, 2}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("workout order = %v, want %v", got, want)
		}
	}
}

func TestGroupIntoSessionsSingleWorkout(t *testing.T) {
	workouts := []model.Workout{makeSessionWorkout(1, "2026-03-07 09:00:00", 5500)}
	sessions := GroupIntoSessions(workouts)
	if len(sessions) != 1 {
		t.Fatalf("len(GroupIntoSessions()) = %d, want 1", len(sessions))
	}
	if sessions[0].TotalDistance != 5500 {
		t.Fatalf("session total distance = %d, want 5500", sessions[0].TotalDistance)
	}
	if len(sessions[0].Workouts) != 1 {
		t.Fatalf("session workouts = %d, want 1", len(sessions[0].Workouts))
	}
}

func TestGroupIntoSessionsEmpty(t *testing.T) {
	if sessions := GroupIntoSessions(nil); len(sessions) != 0 {
		t.Fatalf("len(GroupIntoSessions(nil)) = %d, want 0", len(sessions))
	}
}

func TestSessionCountCountsUniqueCalendarDays(t *testing.T) {
	workouts := []model.Workout{
		makeSessionWorkout(1, "2026-03-07 09:21:00", 1000),
		makeSessionWorkout(2, "2026-03-07 09:45:00", 2500),
		makeSessionWorkout(3, "2026-03-07 09:53:00", 1000),
		makeSessionWorkout(4, "2026-03-05 14:00:00", 5000),
	}
	if got := SessionCount(workouts); got != 2 {
		t.Fatalf("SessionCount() = %d, want 2", got)
	}
}

func TestSessionCountEmpty(t *testing.T) {
	if got := SessionCount(nil); got != 0 {
		t.Fatalf("SessionCount(nil) = %d, want 0", got)
	}
}
