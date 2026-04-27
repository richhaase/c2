package model

import (
	"strings"
	"time"
)

type HeartRate struct {
	Average *int `json:"average,omitempty"`
	Min     *int `json:"min,omitempty"`
	Max     *int `json:"max,omitempty"`
}

type Workout struct {
	ID            int        `json:"id"`
	UserID        int        `json:"user_id"`
	Date          string     `json:"date"`
	Timezone      string     `json:"timezone,omitempty"`
	Distance      int        `json:"distance"`
	Type          string     `json:"type"`
	Time          int        `json:"time"`
	TimeFormatted string     `json:"time_formatted"`
	WorkoutType   string     `json:"workout_type,omitempty"`
	Source        string     `json:"source,omitempty"`
	WeightClass   string     `json:"weight_class,omitempty"`
	StrokeRate    *int       `json:"stroke_rate,omitempty"`
	StrokeCount   *int       `json:"stroke_count,omitempty"`
	CaloriesTotal *int       `json:"calories_total,omitempty"`
	DragFactor    *int       `json:"drag_factor,omitempty"`
	HeartRate     *HeartRate `json:"heart_rate,omitempty"`
	StrokeData    bool       `json:"stroke_data,omitempty"`
	RestTime      *int       `json:"rest_time,omitempty"`
	RestDistance  *int       `json:"rest_distance,omitempty"`
	Comments      string     `json:"comments,omitempty"`
}

type StrokeData struct {
	T   *float64 `json:"t,omitempty"`
	D   *float64 `json:"d,omitempty"`
	P   *float64 `json:"p,omitempty"`
	SPM *float64 `json:"spm,omitempty"`
	HR  *float64 `json:"hr,omitempty"`
}

type UserProfile struct {
	ID        int    `json:"id"`
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

func ParsedDate(w Workout) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", w.Date, time.Local)
}

func CalendarDay(w Workout) string {
	if len(w.Date) < 10 {
		return w.Date
	}
	return w.Date[:10]
}

func IsIntervalWorkout(w Workout) bool {
	if strings.Contains(w.WorkoutType, "Interval") {
		return true
	}
	if w.RestTime != nil && *w.RestTime > 0 {
		return true
	}
	return w.RestDistance != nil && *w.RestDistance > 0
}
