package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/goals"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/store"
)

func TestDoctorReportsInvalidGoalsAlongsideOtherCorruption(t *testing.T) {
	p := paths.For(seedWorkouts(t, testHome(t)))
	for _, file := range []string{goals.File(p), p.StrokeFile(1)} {
		if err := os.WriteFile(file, []byte("not-json"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := run(t, "data", "doctor")
	if !r.failed || !strings.Contains(r.stderr, "goals.json") || !strings.Contains(r.stderr, "strokes/1.jsonl") || !strings.Contains(r.stderr, "2 problems found") {
		t.Fatalf("incomplete diagnosis: %+v", r)
	}
	for _, file := range []string{goals.File(p), p.StrokeFile(1)} {
		data, err := os.ReadFile(file)
		if err != nil || string(data) != "not-json" {
			t.Fatalf("doctor changed %s: %s, %v", file, data, err)
		}
	}
}

func TestSetupRecoversFromInvalidCurrentGoals(t *testing.T) {
	for _, mode := range []string{"replacement", "retain current", "invalid replacement"} {
		t.Run(mode, func(t *testing.T) {
			current := paths.For(seedWorkouts(t, testHome(t)))
			if err := os.WriteFile(goals.File(current), []byte("not-json"), 0o644); err != nil {
				t.Fatal(err)
			}
			target := current
			if mode != "retain current" {
				target = paths.For(t.TempDir())
				if err := store.Init(target, time.Now(), nil); err != nil {
					t.Fatal(err)
				}
				if err := goals.Write(target, goals.Collection{Version: 1, Timezone: "UTC", Goals: []goals.Goal{}}); err != nil {
					t.Fatal(err)
				}
				if mode == "invalid replacement" {
					if err := os.WriteFile(goals.File(target), []byte("not-json"), 0o644); err != nil {
						t.Fatal(err)
					}
				}
			}
			before, err := os.ReadFile(goals.File(target))
			if err != nil {
				t.Fatal(err)
			}
			prior := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = prior })
			http.DefaultTransport = syncTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path != "/api/users/me" || r.Header.Get("Authorization") != "Bearer replacement-token" {
					return nil, fmt.Errorf("Unexpected setup verification request")
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":{"id":1,"username":"test"}}`)), Header: make(http.Header)}, nil
			})
			r := runWithStdin(t, "replacement-token\n"+target.Root+"\n", "setup")
			if r.failed || !strings.Contains(r.stderr, "goals.json") || !strings.Contains(r.stdout, "Authenticated as: test") {
				t.Fatalf("setup recovery failed: %+v", r)
			}
			cfg, err := config.Load()
			if err != nil {
				t.Fatal(err)
			}
			want := target.Root
			if mode == "invalid replacement" {
				want = current.Root
			}
			if paths.CanonicalRoot(cfg.DataDir) != paths.CanonicalRoot(want) || cfg.API.Token != "replacement-token" {
				t.Fatal("setup did not save the credentials and appropriate store")
			}
			after, err := os.ReadFile(goals.File(target))
			if err != nil || string(after) != string(before) {
				t.Fatal("setup changed selected store goals")
			}
			original, err := os.ReadFile(goals.File(current))
			if err != nil || string(original) != "not-json" {
				t.Fatal("setup changed damaged current goals")
			}
		})
	}
}
