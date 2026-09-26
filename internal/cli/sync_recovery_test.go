package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/storage"
)

type syncTransport func(*http.Request) (*http.Response, error)

func (f syncTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestSyncRecoversStrokeCacheWithoutBypassingStoreGuards(t *testing.T) {
	for _, tc := range []struct {
		name        string
		prepared    bool
		account     int64
		corruptNote bool
	}{
		{name: "legacy", account: 1},
		{name: "prepared", prepared: true, account: 1},
		{name: "wrong account", prepared: true, account: 2},
		{name: "corrupt note", prepared: true, account: 1, corruptNote: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := paths.For(seedWorkouts(t, testHome(t)))
			if _, err := storage.UpsertWorkouts(p, []models.Workout{{ID: 1, UserID: 1, Date: "2026-09-06 12:00:00", Distance: 5000, Time: 15000, Type: "rower", StrokeData: true}}); err != nil {
				t.Fatal(err)
			}
			if tc.prepared {
				mustRun(t, "data", "prepare", "--timezone", "UTC")
			}
			broken := "{\"t\":1,\"d\":5}\nnot-json\n"
			if err := os.WriteFile(p.StrokeFile(1), []byte(broken), 0o644); err != nil {
				t.Fatal(err)
			}
			if r := run(t, "data", "prepare", "--timezone", "UTC"); !r.failed {
				t.Fatal("transfer preparation accepted corrupt strokes")
			}
			if tc.corruptNote {
				if err := os.MkdirAll(p.NotesDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(p.NotesDir, "broken.json"), []byte("not-json"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.ReadFile(p.Meta)
			if err != nil {
				t.Fatal(err)
			}
			results, strokes := 0, 0
			prior := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = prior })
			http.DefaultTransport = syncTransport(func(r *http.Request) (*http.Response, error) {
				var body string
				switch r.URL.Path {
				case "/api/users/me":
					body = fmt.Sprintf(`{"data":{"id":%d}}`, tc.account)
				case "/api/users/me/results":
					results++
					body = `{"data":[],"meta":{"pagination":{"total_pages":1}}}`
				case "/api/users/me/results/1/strokes":
					strokes++
					body = `{"data":[{"t":2,"d":10}]}`
				default:
					return nil, fmt.Errorf("Unexpected API request: %s", r.URL.Path)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
			})
			r := run(t, "sync")
			blocked := tc.account != 1 || tc.corruptNote
			if r.failed != blocked {
				t.Fatalf("sync failed=%v, want %v: %s / %s", r.failed, blocked, r.stdout, r.stderr)
			}
			if blocked {
				after, err := os.ReadFile(p.Meta)
				if err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(p.StrokeFile(1))
				if err != nil {
					t.Fatal(err)
				}
				if results != 0 || strokes != 0 || string(before) != string(after) || string(data) != broken {
					t.Fatal("blocked sync fetched results or changed store data")
				}
				return
			}
			data, err := storage.ReadStrokeData(p, 1)
			if err != nil || len(data) != 1 || results != 1 || strokes != 1 {
				t.Fatalf("recovery: results=%d strokes=%d data=%+v err=%v", results, strokes, data, err)
			}
			if valid, err := storage.HasStrokeData(p, 1); err != nil || !valid {
				t.Fatalf("recovered cache invalid: %v", err)
			}
			mustRun(t, "data", "prepare", "--timezone", "UTC")
		})
	}
}
