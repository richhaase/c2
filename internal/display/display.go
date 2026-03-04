// Copyright (c) 2026 Rich Haase. All rights reserved.
// Use of this source code is governed by the MIT license.

package display

import (
	"fmt"
	"strings"

	"github.com/richhaase/c2cli/internal/models"
)

// FormatMeters formats meters with comma separators: 1000000 → "1,000,000"
func FormatMeters(m int64) string {
	s := fmt.Sprintf("%d", m)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	offset := len(s) % 3
	if offset > 0 {
		b.WriteString(s[:offset])
	}
	for i := offset; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

// FormatPercent formats a ratio as a percentage: 0.1234 → "12.3%"
func FormatPercent(ratio float64) string {
	return fmt.Sprintf("%.1f%%", ratio*100)
}

// FormatMetersPerWeek formats as "19,231m/week".
func FormatMetersPerWeek(m int64) string {
	return FormatMeters(m) + "m/week"
}

// FormatWorkoutLine formats a workout as a compact log line.
func FormatWorkoutLine(w *models.Workout, dateFormat string) string {
	dateStr := w.Date[:5] // fallback: first 5 chars
	if t, err := w.ParsedDate(); err == nil {
		dateStr = t.Format(goDateFormat(dateFormat))
	}

	distance := fmt.Sprintf("%sm", FormatMeters(w.Distance))
	pace := w.Pace500m()
	spm := intFieldSuffix(w.StrokeRate, "spm")
	hr := "-"
	if w.HeartRate != nil && w.HeartRate.Average > 0 {
		hr = fmt.Sprintf("%dbpm", w.HeartRate.Average)
	}
	df := intFieldSuffix(w.DragFactor, "df")

	return fmt.Sprintf("%s  %s  %s  %s/500m  %s  %s  %s",
		dateStr, distance, w.TimeFormatted, pace, spm, hr, df)
}


func intFieldSuffix(v int, suffix string) string {
	if v == 0 {
		return "-"
	}
	return fmt.Sprintf("%d%s", v, suffix)
}

// goDateFormat converts a strftime-style format to Go's reference time format.
func goDateFormat(f string) string {
	r := strings.NewReplacer(
		"%m", "01",
		"%d", "02",
		"%Y", "2006",
		"%y", "06",
		"%H", "15",
		"%M", "04",
	)
	return r.Replace(f)
}
