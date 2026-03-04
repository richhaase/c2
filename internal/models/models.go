package models

import "time"

type UserProfile struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Email     string `json:"email,omitempty"`
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
	WeightClass   string     `json:"weight_class,omitempty"`
	StrokeRate    int        `json:"stroke_rate,omitempty"`
	StrokeCount   int        `json:"stroke_count,omitempty"`
	CaloriesTotal int        `json:"calories_total,omitempty"`
	DragFactor    int        `json:"drag_factor,omitempty"`
	HeartRate     *HeartRate `json:"heart_rate,omitempty"`
	Pace500m      string     `json:"pace_500m,omitempty"`
	AverageWatts  int        `json:"average_watts,omitempty"`
	Comments      string     `json:"comments,omitempty"`
	UpdatedAt     string     `json:"updated_at,omitempty"`
}

// ParsedDate parses the Date field ("2006-01-02 15:04:05") into a time.Time.
func (w *Workout) ParsedDate() (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", w.Date)
}

type StrokeData struct {
	Time      float64 `json:"time,omitempty"`
	Distance  float64 `json:"distance,omitempty"`
	Pace      float64 `json:"pace,omitempty"`
	SPM       int     `json:"spm,omitempty"`
	HeartRate int     `json:"heart_rate,omitempty"`
}

type PaginatedResponse struct {
	Data []Workout      `json:"data"`
	Meta *PaginationMeta `json:"meta,omitempty"`
}

type PaginationMeta struct {
	CurrentPage int `json:"current_page,omitempty"`
	LastPage    int `json:"last_page,omitempty"`
	Total       int `json:"total,omitempty"`
}
