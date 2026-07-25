package cli

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/richhaase/c2/internal/models"
)

func TestWriteCSVQuotesEveryFieldThatRequiresIt(t *testing.T) {
	var out bytes.Buffer
	workout := models.Workout{
		ID:            1,
		Date:          "2026-07-20 08:00:00",
		Distance:      1000,
		Time:          1750,
		TimeFormatted: "2:55.0",
		Type:          "rower,indoor",
		WorkoutType:   `Fixed"Distance`,
		Comments:      "line 1,\n\"line 2\"",
	}
	if err := writeCSV(csv.NewWriter(&out), []models.Workout{workout}); err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(strings.NewReader(out.String())).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("records = %#v", records)
	}
	row := records[1]
	if row[13] != workout.WorkoutType || row[16] != workout.Type || row[17] != workout.Comments {
		t.Fatalf("row = %#v", row)
	}
}
