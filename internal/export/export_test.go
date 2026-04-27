package export

import (
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/richhaase/c2/internal/model"
)

func testWorkout(id int, date string, distance int) model.Workout {
	return model.Workout{
		ID:            id,
		UserID:        1,
		Date:          date,
		Distance:      distance,
		Type:          "rower",
		Time:          distance * 35 / 10,
		TimeFormatted: "0:00.0",
	}
}

func ptrInt(v int) *int {
	return &v
}

func TestCSVPreservesHeaderOrder(t *testing.T) {
	got, err := CSV(nil)
	if err != nil {
		t.Fatalf("CSV() error = %v", err)
	}

	firstLine := strings.Split(strings.TrimRight(got, "\n"), "\n")[0]
	want := "id,date,distance,time_tenths,time_formatted,pace_500m,stroke_rate,stroke_count,calories,drag_factor,hr_avg,hr_min,hr_max,workout_type,rest_time_tenths,rest_distance,machine_type,comments"
	if firstLine != want {
		t.Fatalf("header mismatch\nwant: %s\n got: %s", want, firstLine)
	}
}

func TestCSVEscapesCommentFieldsLikeTypeScript(t *testing.T) {
	workouts := []model.Workout{
		{ID: 1, UserID: 1, Date: "2026-01-15 10:00:00", Distance: 5000, Type: "rower", Time: 17500, TimeFormatted: "29:10.0", Comments: "plain"},
		{ID: 2, UserID: 1, Date: "2026-01-16 10:00:00", Distance: 5000, Type: "rower", Time: 17500, TimeFormatted: "29:10.0", Comments: "a,b"},
		{ID: 3, UserID: 1, Date: "2026-01-17 10:00:00", Distance: 5000, Type: "rower", Time: 17500, TimeFormatted: "29:10.0", Comments: `say "hi"`},
		{ID: 4, UserID: 1, Date: "2026-01-18 10:00:00", Distance: 5000, Type: "rower", Time: 17500, TimeFormatted: "29:10.0", Comments: "line1\nline2"},
		{ID: 5, UserID: 1, Date: "2026-01-19 10:00:00", Distance: 5000, Type: "rower", Time: 17500, TimeFormatted: "29:10.0", Comments: " leading"},
	}

	got, err := CSV(workouts)
	if err != nil {
		t.Fatalf("CSV() error = %v", err)
	}

	for _, want := range []string{
		",rower,plain\n",
		`,rower,"a,b"` + "\n",
		`,rower,"say ""hi"""` + "\n",
		`,rower,"line1` + "\n" + `line2"` + "\n",
		",rower, leading\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("CSV output missing escaped field %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, `rower," leading"`) {
		t.Fatalf("CSV output quoted leading-space field, want TypeScript-compatible unquoted field:\n%s", got)
	}
}

func TestFilterByDateUsesInclusiveCalendarDateBounds(t *testing.T) {
	workouts := []model.Workout{
		testWorkout(1, "2026-01-15 10:00:00", 5000),
		testWorkout(2, "2026-02-15 10:00:00", 5000),
		testWorkout(3, "2026-03-15 10:00:00", 5000),
	}

	got, err := FilterByDate(workouts, "2026-02-15", "2026-03-15")
	if err != nil {
		t.Fatalf("FilterByDate() error = %v", err)
	}
	if len(got) != 2 || got[0].ID != 2 || got[1].ID != 3 {
		t.Fatalf("inclusive bounds returned IDs %v, want [2 3]", ids(got))
	}

	got, err = FilterByDate(workouts, "2026-02-01", "")
	if err != nil {
		t.Fatalf("FilterByDate(from only) error = %v", err)
	}
	if len(got) != 2 || got[0].ID != 2 || got[1].ID != 3 {
		t.Fatalf("from-only returned IDs %v, want [2 3]", ids(got))
	}

	got, err = FilterByDate(workouts, "", "2026-02-28")
	if err != nil {
		t.Fatalf("FilterByDate(to only) error = %v", err)
	}
	if len(got) != 2 || got[0].ID != 1 || got[1].ID != 2 {
		t.Fatalf("to-only returned IDs %v, want [1 2]", ids(got))
	}
}

func TestFilterByDateRejectsInvalidBounds(t *testing.T) {
	if _, err := FilterByDate(nil, "2026-99-99", ""); err == nil {
		t.Fatal("FilterByDate() error = nil, want invalid from date error")
	}
	if _, err := FilterByDate(nil, "", "not-a-date"); err == nil {
		t.Fatal("FilterByDate() error = nil, want invalid to date error")
	}
}

func TestCSVRowsMatchHeaderLengthAndIntervalColumns(t *testing.T) {
	workouts := []model.Workout{
		{
			ID:            1,
			UserID:        1,
			Date:          "2026-04-09 07:00:00",
			Distance:      5000,
			Type:          "rower",
			Time:          17155,
			TimeFormatted: "28:35.4",
			WorkoutType:   "FixedDistanceSplits",
		},
		{
			ID:            2,
			UserID:        1,
			Date:          "2026-04-11 09:14:00",
			Distance:      3000,
			Type:          "rower",
			Time:          8626,
			TimeFormatted: "20:22.6",
			WorkoutType:   "FixedDistanceInterval",
			RestTime:      ptrInt(3600),
			RestDistance:  ptrInt(660),
		},
		{
			ID:            3,
			UserID:        1,
			Date:          "2026-04-12 07:00:00",
			Distance:      5000,
			Type:          "rower",
			Time:          17155,
			TimeFormatted: "28:35.4",
			RestTime:      ptrInt(0),
			RestDistance:  ptrInt(0),
		},
	}

	got, err := CSV(workouts)
	if err != nil {
		t.Fatalf("CSV() error = %v", err)
	}
	records, err := csv.NewReader(strings.NewReader(got)).ReadAll()
	if err != nil {
		t.Fatalf("CSV output is not parseable: %v\n%s", err, got)
	}
	header := records[0]
	restTimeIdx := indexOf(header, "rest_time_tenths")
	restDistanceIdx := indexOf(header, "rest_distance")
	workoutTypeIdx := indexOf(header, "workout_type")
	if restTimeIdx < 0 || restDistanceIdx < 0 || workoutTypeIdx < 0 {
		t.Fatalf("missing interval columns in header %v", header)
	}

	for i, record := range records[1:] {
		if len(record) != len(header) {
			t.Fatalf("row %d length = %d, want %d", i+1, len(record), len(header))
		}
	}
	if records[1][restTimeIdx] != "" || records[1][restDistanceIdx] != "" {
		t.Fatalf("continuous row rest columns = %q/%q, want empty", records[1][restTimeIdx], records[1][restDistanceIdx])
	}
	if records[2][workoutTypeIdx] != "FixedDistanceInterval" || records[2][restTimeIdx] != "3600" || records[2][restDistanceIdx] != "660" {
		t.Fatalf("interval row columns = %q/%q/%q, want FixedDistanceInterval/3600/660", records[2][workoutTypeIdx], records[2][restTimeIdx], records[2][restDistanceIdx])
	}
	if records[3][restTimeIdx] != "0" || records[3][restDistanceIdx] != "0" {
		t.Fatalf("zero rest columns = %q/%q, want 0/0", records[3][restTimeIdx], records[3][restDistanceIdx])
	}
}

func TestJSONIsPrettyPrintedWithTrailingNewline(t *testing.T) {
	workouts := []model.Workout{testWorkout(1, "2026-01-15 10:00:00", 5000)}

	got, err := JSON(workouts)
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("JSON() does not end with newline: %q", got)
	}
	if !strings.Contains(got, "\n  {\n    \"id\": 1,") {
		t.Fatalf("JSON() is not pretty printed:\n%s", got)
	}
	var decoded []model.Workout
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("JSON() output is invalid JSON: %v", err)
	}
}

func TestJSONNilAndEmptySlicesEncodeAsEmptyArray(t *testing.T) {
	for name, workouts := range map[string][]model.Workout{
		"nil":   nil,
		"empty": {},
	} {
		got, err := JSON(workouts)
		if err != nil {
			t.Fatalf("JSON(%s) error = %v", name, err)
		}
		if got != "[]\n" {
			t.Fatalf("JSON(%s) = %q, want []\\n", name, got)
		}
	}
}

func TestJSONLEmitsCompactObjectPerLineWithTrailingNewline(t *testing.T) {
	workouts := []model.Workout{
		testWorkout(1, "2026-01-15 10:00:00", 5000),
		testWorkout(2, "2026-01-16 10:00:00", 6000),
	}

	got, err := JSONL(workouts)
	if err != nil {
		t.Fatalf("JSONL() error = %v", err)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("JSONL() does not end with newline: %q", got)
	}
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("JSONL() line count = %d, want 2", len(lines))
	}
	for _, line := range lines {
		if strings.Contains(line, "\n") || strings.Contains(line, "  ") {
			t.Fatalf("JSONL() line is not compact: %q", line)
		}
		var decoded model.Workout
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Fatalf("JSONL() line is invalid JSON: %v", err)
		}
	}

	empty, err := JSONL(nil)
	if err != nil {
		t.Fatalf("JSONL(nil) error = %v", err)
	}
	if empty != "" {
		t.Fatalf("JSONL(nil) = %q, want empty string", empty)
	}
}

func ids(workouts []model.Workout) []int {
	result := make([]int, 0, len(workouts))
	for _, workout := range workouts {
		result = append(result, workout.ID)
	}
	return result
}

func indexOf(values []string, needle string) int {
	for i, value := range values {
		if value == needle {
			return i
		}
	}
	return -1
}
