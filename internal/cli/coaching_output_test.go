package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoteInputsPreserveExistingWhitespaceRules(t *testing.T) {
	testHome(t)
	text := "  note < & >\n\n"
	file := filepath.Join(t.TempDir(), "note.md")
	if err := os.WriteFile(file, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"positional text", []string{text}, strings.TrimSpace(text)},
		{"positional stdin", []string{"-"}, strings.TrimSpace(text)},
		{"implicit stdin", nil, strings.TrimSpace(text)},
		{"body flag", []string{"--body", text}, text},
		{"file flag", []string{"--file", file}, text},
		{"stdin flag", []string{"--file", "-"}, text},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"note", "add", "--json"}, tc.args...)
			got := runWithStdin(t, text, args...)
			if got.failed {
				t.Fatalf("%v = %+v", args, got)
			}
			note := savedNote(t, got.stdout)
			if note.Body != tc.want {
				t.Fatalf("body = %q, want %q", note.Body, tc.want)
			}
			if shown := savedNote(t, mustRun(t, "note", "show", note.ID, "--json").stdout); shown.Body != tc.want {
				t.Fatalf("stored body = %q, want %q", shown.Body, tc.want)
			}
		})
	}
}

func TestCoachingDocumentInputsAndReceipts(t *testing.T) {
	testHome(t)
	text := "  # Plan < & >\n\n"
	file := filepath.Join(t.TempDir(), "source.md")
	if err := os.WriteFile(file, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"plan", "playbook", "narrative"} {
		write := []string{name, "set"}
		if name == "narrative" {
			write = []string{name, "add", "2026-09-26"}
		}
		for _, source := range [][]string{{file}, {"--file", file}, {"--body", text}, {"--file", "-"}} {
			args := append(append([]string{}, write...), source...)
			got := runWithStdin(t, text, append(args, "--json")...)
			if got.failed {
				t.Fatalf("%v = %+v", args, got)
			}
			var env struct {
				Schema string          `json:"schema"`
				Data   documentPayload `json:"data"`
			}
			if err := json.Unmarshal([]byte(got.stdout), &env); err != nil || env.Schema != "c2.document.v1" || env.Data.Name != name || env.Data.Content != text {
				t.Fatalf("receipt = %s, %v", got.stdout, err)
			}
			if name == "narrative" && env.Data.Date != "2026-09-26" {
				t.Fatal("missing narrative date")
			}
			if shown := mustRun(t, name, "show"); shown.stdout != text {
				t.Fatalf("text changed: %q", shown.stdout)
			}
			var read struct {
				Data documentPayload `json:"data"`
			}
			if err := json.Unmarshal([]byte(mustRun(t, name, "show", "--json").stdout), &read); err != nil || read.Data != env.Data {
				t.Fatalf("read differs from receipt: %+v, %v", read, err)
			}
		}
		for _, source := range [][]string{{"--body", ""}, {file, "--file", file}, {"--body", "x", "--file", file}} {
			if !run(t, append(append([]string{}, write...), source...)...).failed {
				t.Fatalf("accepted invalid inputs: %v", source)
			}
			if got := mustRun(t, name, "show"); got.stdout != text {
				t.Fatal("invalid input changed saved document")
			}
		}
		got := mustRun(t, append(append([]string{}, write...), "--body", "no newline", "--json")...)
		var read struct {
			Data documentPayload `json:"data"`
		}
		if err := json.Unmarshal([]byte(got.stdout), &read); err != nil || read.Data.Content != "no newline\n" {
			t.Fatalf("receipt did not reflect final newline: %s, %v", got.stdout, err)
		}
	}
}

func TestGoalArchiveJSONReceipt(t *testing.T) {
	home := testHome(t)
	seedWorkouts(t, home)
	mustRun(t, "data", "prepare", "--timezone", "UTC")
	created := mustRun(t, "goal", "add", "Pace", "--kind", "pace", "--target", "2:30", "--json")
	var env struct {
		Schema string `json:"schema"`
		Data   struct {
			ID       string `json:"id"`
			Archived bool   `json:"archived"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(created.stdout), &env); err != nil || env.Data.ID == "" {
		t.Fatalf("created %s, %v", created.stdout, err)
	}
	id := env.Data.ID
	archived := mustRun(t, "goal", "archive", id, "--json")
	if err := json.Unmarshal([]byte(archived.stdout), &env); err != nil || env.Schema != "c2.goal.saved.v1" || env.Data.ID != id || !env.Data.Archived {
		t.Fatalf("archive receipt %s, %v", archived.stdout, err)
	}
	if strings.Contains(mustRun(t, "goal", "list").stdout, id) {
		t.Fatal("archived goal still active")
	}
}
