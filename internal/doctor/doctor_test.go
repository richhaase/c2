package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/store"
)

var testNow = time.Date(2026, 7, 6, 12, 0, 0, 0, time.Local)

func tempStore(t *testing.T) paths.DataPaths {
	t.Helper()
	p := paths.For(filepath.Join(t.TempDir(), "store"))
	if err := os.MkdirAll(p.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := store.Init(p, testNow, nil); err != nil {
		t.Fatal(err)
	}
	return p
}

func hasIssueContaining(report Report, substr string) bool {
	for _, issue := range report.Issues {
		if strings.Contains(issue, substr) {
			return true
		}
	}
	return false
}

func TestCleanStoreHasNoIssues(t *testing.T) {
	p := tempStore(t)
	line := `{"id":1,"user_id":1,"date":"2026-07-01 08:00:00","distance":8000,"type":"rower","time":12000,"time_formatted":"20:00.0"}`
	if err := os.WriteFile(p.Workouts, []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.StrokeFile(1), []byte(`{"t":1,"d":500,"p":1750,"spm":24,"hr":110}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report := Run(p)
	if len(report.Issues) != 0 {
		t.Fatalf("issues: %v", report.Issues)
	}
	if report.CheckedFiles == 0 {
		t.Fatal("expected checked files")
	}
}

func TestMixedOffsetArchivesPassDoctorAfterCompaction(t *testing.T) {
	p := tempStore(t)
	write := func(id, date, body string) {
		t.Helper()
		if err := notes.Write(p, notes.Record{ID: id, Date: date, Type: "observation", Body: body, Author: "athlete"}); err != nil {
			t.Fatal(err)
		}
	}
	write("TZ1", "2026-06-05T23:30:00-06:00", "denver")
	write("TZ2", "2026-06-06T01:00:00+02:00", "europe")

	result, err := notes.Compact(p, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if result.Archived != 2 {
		t.Fatalf("archived %d", result.Archived)
	}
	report := Run(p)
	if len(report.Issues) != 0 {
		t.Fatalf("issues: %v", report.Issues)
	}
}

func TestCorruptWorkoutLineIsReported(t *testing.T) {
	p := tempStore(t)
	body := `{"id":1,"user_id":1,"date":"2026-07-01 08:00:00","distance":8000,"type":"rower","time":12000,"time_formatted":"20:00.0"}` + "\n" +
		"not json\n" +
		`{"id":"two","date":"2026-07-02 08:00:00","distance":1,"time":1}` + "\n"
	if err := os.WriteFile(p.Workouts, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	report := Run(p)
	if !hasIssueContaining(report, "line 2 is not valid JSON") {
		t.Fatalf("issues: %v", report.Issues)
	}
	if !hasIssueContaining(report, "line 3 malformed workout record") {
		t.Fatalf("issues: %v", report.Issues)
	}
}

func TestMalformedStrokeRecordIsReported(t *testing.T) {
	p := tempStore(t)
	if err := os.WriteFile(p.StrokeFile(4), []byte(`{"t":1,"p":"fast"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report := Run(p)
	if !hasIssueContaining(report, "malformed stroke record") {
		t.Fatalf("issues: %v", report.Issues)
	}
}

func TestMissingSchemaVersionIsReported(t *testing.T) {
	p := tempStore(t)
	if err := os.WriteFile(p.Meta, []byte(`{"created":"2026-07-05T12:00:00.000Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	report := Run(p)
	if !hasIssueContaining(report, "missing numeric schema_version") {
		t.Fatalf("issues: %v", report.Issues)
	}
}

func TestUnsupportedAndFractionalSchemaVersionsAreReported(t *testing.T) {
	for _, body := range []string{
		`{"schema_version":999}`,
		`{"schema_version":1.5}`,
	} {
		p := tempStore(t)
		if err := os.WriteFile(p.Meta, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		report := Run(p)
		if len(report.Issues) == 0 {
			t.Fatalf("body %s produced no issues", body)
		}
	}
}

func TestCorruptMetaIsReportedAsInvalidJSON(t *testing.T) {
	p := tempStore(t)
	if err := os.WriteFile(p.Meta, []byte("{ truncated"), 0o644); err != nil {
		t.Fatal(err)
	}
	report := Run(p)
	if !hasIssueContaining(report, "meta.json: not valid JSON") {
		t.Fatalf("issues: %v", report.Issues)
	}
}

func TestDivergentLooseNoteCopiesAreReported(t *testing.T) {
	p := tempStore(t)
	a := `{"id":"SAME","date":"2026-07-05T10:00:00-06:00","type":"observation","body":"one","author":"athlete"}`
	b := `{"id":"SAME","date":"2026-07-05T10:00:00-06:00","type":"observation","body":"two","author":"athlete"}`
	if err := os.WriteFile(filepath.Join(p.NotesDir, "SAME.json"), []byte(a), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.NotesDir, "SAME.sync-conflict.json"), []byte(b), 0o644); err != nil {
		t.Fatal(err)
	}
	report := Run(p)
	if !hasIssueContaining(report, "divergent copies of note SAME") {
		t.Fatalf("issues: %v", report.Issues)
	}
}

func TestArchiveOrderingAndDuplicatesAreReported(t *testing.T) {
	p := tempStore(t)
	older := `{"id":"B","date":"2026-01-02T08:00:00-06:00","type":"observation","body":"x","author":"athlete"}`
	newer := `{"id":"A","date":"2026-01-01T08:00:00-06:00","type":"observation","body":"y","author":"athlete"}`
	dup := `{"id":"B","date":"2026-01-03T08:00:00-06:00","type":"observation","body":"z","author":"athlete"}`
	body := older + "\n" + newer + "\n" + dup + "\n"
	if err := os.WriteFile(p.ArchiveFile(2026), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	report := Run(p)
	if !hasIssueContaining(report, "out of (date, id) order") {
		t.Fatalf("issues: %v", report.Issues)
	}
	if !hasIssueContaining(report, "duplicate note id B") {
		t.Fatalf("issues: %v", report.Issues)
	}
}

func TestNullStrokeRecordIsRejected(t *testing.T) {
	p := tempStore(t)
	if err := os.WriteFile(p.StrokeFile(5), []byte("{\"t\":1}\nnull\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report := Run(p)
	if !hasIssueContaining(report, "line 2 malformed stroke record") {
		t.Fatalf("null stroke line must be reported: %v", report.Issues)
	}
}

func TestMalformedWorkoutDatesAndDuplicateIDsAreReported(t *testing.T) {
	p := tempStore(t)
	body := strings.Join([]string{
		`{"id":1,"date":"not-a-date","distance":1000,"time":1000}`,
		`{"id":2,"date":"2026-01-01","distance":1000,"time":1000}`,
		`{"id":2,"date":"2026-01-02","distance":1000,"time":1000}`,
	}, "\n") + "\n"
	if err := os.WriteFile(p.Workouts, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	report := Run(p)
	if !hasIssueContaining(report, "malformed workout record") {
		t.Fatalf("issues = %v", report.Issues)
	}
	if !hasIssueContaining(report, "duplicate workout id 2") {
		t.Fatalf("issues = %v", report.Issues)
	}
}

func TestStrokeShapeAcceptsOnlyObjects(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{`{"t":1,"d":2,"p":3,"spm":4,"hr":5}`, true},
		{`{}`, true},
		{`{"t":1,"extra":"ok"}`, true},
		{`{"t":null}`, false},
		{`{"t":1,"hr":null}`, false},
		{`null`, false},
		{`[1,2]`, false},
		{`"str"`, false},
		{`123`, false},
		{`{"t":"bad"}`, false},
	}
	for _, tc := range cases {
		if got := isStrokeShaped([]byte(tc.in)); got != tc.want {
			t.Errorf("isStrokeShaped(%s) = %v want %v", tc.in, got, tc.want)
		}
	}
}
