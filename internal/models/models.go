package models

import (
	"fmt"
	"time"
)

type UserProfile struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email,omitempty"`
}

// UserResponse wraps the API response for /api/users/me.
type UserResponse struct {
	Data UserProfile `json:"data"`
}

type HeartRate struct {
	Average int `json:"average,omitempty"`
	Min     int `json:"min,omitempty"`
	Max     int `json:"max,omitempty"`
}

// Workout represents a single result from the Concept2 Logbook API.
// The Time field is in tenths of a second (e.g., 19122 = 31:52.2).
type Workout struct {
	ID            int64      `json:"id"`
	UserID        int64      `json:"user_id"`
	Date          string     `json:"date"`
	Timezone      string     `json:"timezone,omitempty"`
	Distance      int64      `json:"distance"`
	MachineType   string     `json:"type"`
	Time          int64      `json:"time"`
	TimeFormatted string     `json:"time_formatted"`
	WorkoutType   string     `json:"workout_type,omitempty"`
	Source        string     `json:"source,omitempty"`
	WeightClass   string     `json:"weight_class,omitempty"`
	StrokeRate    int        `json:"stroke_rate,omitempty"`
	StrokeCount   int        `json:"stroke_count,omitempty"`
	CaloriesTotal int        `json:"calories_total,omitempty"`
	DragFactor    int        `json:"drag_factor,omitempty"`
	HeartRate     *HeartRate `json:"heart_rate,omitempty"`
	StrokeDataAvl bool       `json:"stroke_data,omitempty"`
	Comments      string     `json:"comments,omitempty"`
}

// ParsedDate parses the Date field ("2006-01-02 15:04:05") into a time.Time.
func (w *Workout) ParsedDate() (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", w.Date)
}

// Pace500m computes the /500m pace string from time (tenths of seconds) and distance.
func (w *Workout) Pace500m() string {
	if w.Distance == 0 || w.Time == 0 {
		return "-"
	}
	// Time is in tenths of a second
	paceSeconds := float64(w.Time) / 10.0 * 500.0 / float64(w.Distance)
	mins := int(paceSeconds) / 60
	secs := paceSeconds - float64(mins*60)
	return fmt.Sprintf("%d:%04.1f", mins, secs)
}

type StrokeData struct {
	Time      float64 `json:"time,omitempty"`
	Distance  float64 `json:"distance,omitempty"`
	Pace      float64 `json:"pace,omitempty"`
	SPM       int     `json:"spm,omitempty"`
	HeartRate int     `json:"heart_rate,omitempty"`
}

// ResultsResponse matches the API response for /api/users/me/results.
type ResultsResponse struct {
	Data []Workout    `json:"data"`
	Meta *ResultsMeta `json:"meta,omitempty"`
}

type ResultsMeta struct {
	Pagination *Pagination `json:"pagination,omitempty"`
}

type Pagination struct {
	Total       int `json:"total"`
	Count       int `json:"count"`
	PerPage     int `json:"per_page"`
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
}
