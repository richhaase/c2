package analysis

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/richhaase/c2/internal/models"
)

const (
	paceDistance                = 500
	shapeDriftThresholdSeconds  = 1.5
	shapeSpreadThresholdSeconds = 6
	bandWidthSeconds            = 5
	minSteadyDistance           = 1000
)

type SplitRow struct {
	Index           int      `json:"index"`
	Distance        *int     `json:"distance"`
	TimeSeconds     float64  `json:"time_seconds"`
	Pace500mSeconds *float64 `json:"pace_500m_seconds"`
	Pace500m        *string  `json:"pace_500m"`
	StrokeRate      *int     `json:"stroke_rate"`
	HRAvg           *int     `json:"hr_avg"`
	HRMax           *int     `json:"hr_max"`
}

type Shape string

const (
	ShapeEven     Shape = "even"
	ShapeNegative Shape = "negative"
	ShapePositive Shape = "positive"
	ShapeVariable Shape = "variable"
	ShapeUnknown  Shape = "unknown"
)

func roundTenth(x float64) float64 {
	return math.Round(x*10) / 10
}

func SplitTable(w models.Workout) []SplitRow {
	var splits []models.WorkoutSplit
	if w.Workout != nil {
		splits = w.Workout.Splits
	}
	rows := make([]SplitRow, 0, len(splits))
	for i, s := range splits {
		seconds := s.Time / models.TenthsPerSecond
		row := SplitRow{
			Index:       i + 1,
			Distance:    s.Distance,
			TimeSeconds: roundTenth(seconds),
			StrokeRate:  s.StrokeRate,
		}
		if s.Distance != nil && *s.Distance > 0 {
			pace := seconds * (paceDistance / float64(*s.Distance))
			rounded := roundTenth(pace)
			formatted := models.FormatSeconds(pace)
			row.Pace500mSeconds = &rounded
			row.Pace500m = &formatted
		}
		if s.HeartRate != nil {
			row.HRAvg = s.HeartRate.Average
			row.HRMax = s.HeartRate.Max
		}
		rows = append(rows, row)
	}
	return rows
}

func SplitShape(rows []SplitRow) Shape {
	paces := make([]float64, 0, len(rows))
	for _, r := range rows {
		if r.Pace500mSeconds != nil {
			paces = append(paces, *r.Pace500mSeconds)
		}
	}
	if len(paces) < 2 {
		return ShapeUnknown
	}

	mid := len(paces) / 2
	firstHalf := paces[:mid]
	secondHalf := paces[len(paces)-mid:]
	drift := mean(firstHalf) - mean(secondHalf)

	lowest, highest := paces[0], paces[0]
	for _, p := range paces {
		if p < lowest {
			lowest = p
		}
		if p > highest {
			highest = p
		}
	}
	spread := highest - lowest

	if math.Abs(drift) <= shapeDriftThresholdSeconds {
		if spread > shapeSpreadThresholdSeconds {
			return ShapeVariable
		}
		return ShapeEven
	}
	if drift > 0 {
		return ShapeNegative
	}
	return ShapePositive
}

func mean(xs []float64) float64 {
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

type StrokeSummaryResult struct {
	Samples            int      `json:"samples"`
	AvgPace500mSeconds *float64 `json:"avg_pace_500m_seconds"`
	AvgPace500m        *string  `json:"avg_pace_500m"`
	AvgSPM             *float64 `json:"avg_spm"`
	AvgHR              *int     `json:"avg_hr"`
	MaxHR              *float64 `json:"max_hr"`
}

func StrokeSummary(strokes []models.StrokeData) StrokeSummaryResult {
	paceSum, spmSum, hrSum := 0.0, 0.0, 0.0
	paceCount, spmCount, hrCount := 0, 0, 0
	maxHR := 0.0
	for _, s := range strokes {
		if s.P != nil && *s.P > 0 {
			paceSum += *s.P / models.TenthsPerSecond
			paceCount++
		}
		if s.SPM != nil && *s.SPM > 0 {
			spmSum += *s.SPM
			spmCount++
		}
		if s.HR != nil && *s.HR > 0 {
			hrSum += *s.HR
			hrCount++
			if *s.HR > maxHR {
				maxHR = *s.HR
			}
		}
	}

	out := StrokeSummaryResult{Samples: len(strokes)}
	if paceCount > 0 {
		avgPace := paceSum / float64(paceCount)
		rounded := roundTenth(avgPace)
		formatted := models.FormatSeconds(avgPace)
		out.AvgPace500mSeconds = &rounded
		out.AvgPace500m = &formatted
	}
	if spmCount > 0 {
		avgSPM := roundTenth(spmSum / float64(spmCount))
		out.AvgSPM = &avgSPM
	}
	if hrCount > 0 {
		avgHR := int(math.Round(hrSum / float64(hrCount)))
		out.AvgHR = &avgHR
	}
	if maxHR > 0 {
		out.MaxHR = &maxHR
	}
	return out
}

type HRPaceBand struct {
	BandStartSeconds int    `json:"band_start_seconds"`
	Band             string `json:"band"`
	Workouts         int    `json:"workouts"`
	AvgHR            int    `json:"avg_hr"`
	EarlyAvgHR       *int   `json:"early_avg_hr"`
	LateAvgHR        *int   `json:"late_avg_hr"`
	HRDelta          *int   `json:"hr_delta"`
}

type hrBucket struct {
	all   []int
	early []int
	late  []int
}

func HRAtPace(workouts []models.Workout, now time.Time, weeks int) []HRPaceBand {
	cutoff := now.AddDate(0, 0, -weeks*7)
	midpoint := cutoff.Add(now.Sub(cutoff) / 2)

	buckets := map[int]*hrBucket{}
	for _, w := range workouts {
		if models.IsIntervalWorkout(w) {
			continue
		}
		if w.Distance < minSteadyDistance {
			continue
		}
		if w.HeartRate == nil || w.HeartRate.Average == nil || *w.HeartRate.Average <= 0 {
			continue
		}
		at := models.ParsedDate(w)
		if at.Before(cutoff) || at.After(now) {
			continue
		}
		pace := models.Pace500mSeconds(w)
		if pace <= 0 {
			continue
		}

		bandStart := int(math.Floor(pace/bandWidthSeconds)) * bandWidthSeconds
		bucket := buckets[bandStart]
		if bucket == nil {
			bucket = &hrBucket{}
			buckets[bandStart] = bucket
		}
		hr := *w.HeartRate.Average
		bucket.all = append(bucket.all, hr)
		if at.Before(midpoint) {
			bucket.early = append(bucket.early, hr)
		} else {
			bucket.late = append(bucket.late, hr)
		}
	}

	starts := make([]int, 0, len(buckets))
	for start := range buckets {
		starts = append(starts, start)
	}
	sort.Ints(starts)

	bands := make([]HRPaceBand, 0, len(starts))
	for _, start := range starts {
		bucket := buckets[start]
		early := meanRounded(bucket.early)
		late := meanRounded(bucket.late)
		band := HRPaceBand{
			BandStartSeconds: start,
			Band: fmt.Sprintf("%s–%s",
				models.FormatSeconds(float64(start)),
				models.FormatSeconds(float64(start+bandWidthSeconds))),
			Workouts:   len(bucket.all),
			AvgHR:      *meanRounded(bucket.all),
			EarlyAvgHR: early,
			LateAvgHR:  late,
		}
		if early != nil && late != nil {
			delta := *late - *early
			band.HRDelta = &delta
		}
		bands = append(bands, band)
	}
	return bands
}

func meanRounded(xs []int) *int {
	if len(xs) == 0 {
		return nil
	}
	sum := 0
	for _, x := range xs {
		sum += x
	}
	v := int(math.Round(float64(sum) / float64(len(xs))))
	return &v
}
