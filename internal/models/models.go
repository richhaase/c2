package models

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	TenthsPerSecond = 10
	paceDistance    = 500
)

type HeartRate struct {
	Average *int `json:"average,omitempty"`
	Min     *int `json:"min,omitempty"`
	Max     *int `json:"max,omitempty"`
	Ending  *int `json:"ending,omitempty"`
}

type WorkoutSplit struct {
	Type          string     `json:"type,omitempty"`
	Time          float64    `json:"time"`
	Distance      *int       `json:"distance,omitempty"`
	CaloriesTotal *int       `json:"calories_total,omitempty"`
	StrokeRate    *int       `json:"stroke_rate,omitempty"`
	HeartRate     *HeartRate `json:"heart_rate,omitempty"`
}

type WorkoutTargets struct {
	Pace *float64 `json:"pace,omitempty"`
}

type WorkoutDetail struct {
	Targets *WorkoutTargets `json:"targets,omitempty"`
	Splits  []WorkoutSplit  `json:"splits,omitempty"`
}

type Workout struct {
	ID            int64           `json:"id"`
	UserID        int64           `json:"user_id"`
	Date          string          `json:"date"`
	Timezone      string          `json:"timezone,omitempty"`
	Distance      int             `json:"distance"`
	Type          string          `json:"type"`
	Time          int             `json:"time"`
	TimeFormatted string          `json:"time_formatted"`
	WorkoutType   string          `json:"workout_type,omitempty"`
	Source        string          `json:"source,omitempty"`
	WeightClass   string          `json:"weight_class,omitempty"`
	StrokeRate    *int            `json:"stroke_rate,omitempty"`
	StrokeCount   *int            `json:"stroke_count,omitempty"`
	CaloriesTotal *int            `json:"calories_total,omitempty"`
	DragFactor    *int            `json:"drag_factor,omitempty"`
	HeartRate     *HeartRate      `json:"heart_rate,omitempty"`
	StrokeData    bool            `json:"stroke_data,omitempty"`
	RestTime      *int            `json:"rest_time,omitempty"`
	RestDistance  *int            `json:"rest_distance,omitempty"`
	Comments      string          `json:"comments,omitempty"`
	Workout       *WorkoutDetail  `json:"workout,omitempty"`
	Raw           json.RawMessage `json:"-"`
}

func (w *Workout) UnmarshalJSON(data []byte) error {
	type plain Workout
	var p plain
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}
	*w = Workout(p)
	w.Raw = append(json.RawMessage(nil), data...)
	return nil
}

func (w Workout) MarshalJSON() ([]byte, error) {
	if len(w.Raw) > 0 {
		return w.Raw, nil
	}
	type plain Workout
	return json.Marshal(plain(w))
}

type StrokeData struct {
	T   *float64 `json:"t,omitempty"`
	D   *float64 `json:"d,omitempty"`
	P   *float64 `json:"p,omitempty"`
	SPM *float64 `json:"spm,omitempty"`
	HR  *float64 `json:"hr,omitempty"`
}

type UserProfile struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email,omitempty"`
}

type UserResponse struct {
	Data UserProfile `json:"data"`
}

type Pagination struct {
	Total       int `json:"total"`
	Count       int `json:"count"`
	PerPage     int `json:"per_page"`
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
}

type ResultsMeta struct {
	Pagination *Pagination `json:"pagination,omitempty"`
}

type ResultsResponse struct {
	Data []Workout    `json:"data"`
	Meta *ResultsMeta `json:"meta,omitempty"`
}

type StrokeDataResponse struct {
	Data []StrokeData `json:"data"`
}

func ParsedDate(w Workout) time.Time {
	return ParseLocal(w.Date)
}

func ParseLocal(s string) time.Time {
	normalized := strings.Replace(s, " ", "T", 1)
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, normalized, time.Local); err == nil {
			return t
		}
	}
	return time.Time{}
}

func CalendarDay(w Workout) string {
	if len(w.Date) < 10 {
		return w.Date
	}
	return w.Date[:10]
}

func Pace500mSeconds(w Workout) float64 {
	if w.Distance == 0 || w.Time == 0 {
		return 0
	}
	return (float64(w.Time) / TenthsPerSecond) * (paceDistance / float64(w.Distance))
}

func Pace500m(w Workout) string {
	secs := Pace500mSeconds(w)
	if secs == 0 {
		return "-"
	}
	return FormatSeconds(secs)
}

func IsIntervalWorkout(w Workout) bool {
	if strings.Contains(w.WorkoutType, "Interval") {
		return true
	}
	if w.RestTime != nil && *w.RestTime > 0 {
		return true
	}
	if w.RestDistance != nil && *w.RestDistance > 0 {
		return true
	}
	return false
}

func RestSeconds(w Workout) float64 {
	if w.RestTime == nil {
		return 0
	}
	return float64(*w.RestTime) / TenthsPerSecond
}

func FormatSeconds(totalSeconds float64) string {
	if totalSeconds <= 0 {
		return "0:00.0"
	}
	mins := int(math.Floor(totalSeconds / 60))
	rem := totalSeconds - float64(mins)*60
	return fmt.Sprintf("%d:%s", mins, padStart(strconv.FormatFloat(rem, 'f', 1, 64), 4, '0'))
}

func padStart(s string, width int, pad byte) string {
	if len(s) >= width {
		return s
	}
	return strings.Repeat(string(pad), width-len(s)) + s
}

var ymdPattern = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})$`)

func IsValidYMD(s string) bool {
	m := ymdPattern.FindStringSubmatch(s)
	if m == nil {
		return false
	}
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return false
	}
	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])
	return t.Year() == year && int(t.Month()) == month && t.Day() == day
}

func FilterByDate(workouts []Workout, from, to string) []Workout {
	if from == "" && to == "" {
		return workouts
	}
	out := make([]Workout, 0, len(workouts))
	for _, w := range workouts {
		date := CalendarDay(w)
		if from != "" && date < from {
			continue
		}
		if to != "" && date > to {
			continue
		}
		out = append(out, w)
	}
	return out
}

func ResolveWorkout(workouts []Workout, ref string) *Workout {
	if ref == "last" {
		if len(workouts) == 0 {
			return nil
		}
		latest := 0
		for i, w := range workouts {
			if w.Date > workouts[latest].Date {
				latest = i
			}
		}
		return &workouts[latest]
	}
	id, err := strconv.ParseInt(ref, 10, 64)
	if err != nil {
		return nil
	}
	for i := range workouts {
		if workouts[i].ID == id {
			return &workouts[i]
		}
	}
	return nil
}
