package envelope

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

type testData struct {
	Text string `json:"text"`
}

func TestNewUsesUTCWithMillisecondPrecision(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 34, 56, 789_123_000, time.FixedZone("offset", -6*60*60))
	env := New("c2.test.v1", testData{Text: "ok"}, now)
	if env.Schema != "c2.test.v1" || env.GeneratedAt != "2026-07-20T18:34:56.789Z" {
		t.Fatalf("envelope = %#v", env)
	}
}

func TestPrintKeepsFieldOrderAndDoesNotEscapeHTML(t *testing.T) {
	var out bytes.Buffer
	if err := Print(&out, "c2.test.v1", testData{Text: "<row>&"}); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.HasPrefix(got, "{\n  \"schema\": \"c2.test.v1\",\n  \"generated_at\":") {
		t.Fatalf("output = %s", got)
	}
	if !strings.Contains(got, `"text": "<row>&"`) {
		t.Fatalf("output = %s", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("output has no trailing newline: %q", got)
	}
}
