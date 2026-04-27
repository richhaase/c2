package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/richhaase/c2/internal/model"
)

func TestGetUserSendsBearerToken(t *testing.T) {
	var gotAuth string
	var gotUserAgent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users/me" {
			t.Fatalf("path = %q, want /api/users/me", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		gotUserAgent = r.Header.Get("User-Agent")
		writeJSON(t, w, model.UserResponse{Data: model.UserProfile{ID: 7, Username: "rower"}})
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, Token: "secret", HTTPClient: server.Client(), UserAgent: "c2/test"}
	user, err := client.GetUser(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "rower" {
		t.Fatalf("Username = %q, want rower", user.Username)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("Authorization = %q, want Bearer secret", gotAuth)
	}
	if gotUserAgent != "c2/test" {
		t.Fatalf("User-Agent = %q, want c2/test", gotUserAgent)
	}
}

func TestGetAllResultsPaginates(t *testing.T) {
	var pages []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users/me/results" {
			t.Fatalf("path = %q, want /api/users/me/results", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("type") != "rower" {
			t.Fatalf("type query = %q, want rower", q.Get("type"))
		}
		if q.Get("from") != "2026-04-01T00:00:00Z" || q.Get("to") != "2026-04-30T00:00:00Z" {
			t.Fatalf("date query = from %q to %q", q.Get("from"), q.Get("to"))
		}
		page := q.Get("page")
		pages = append(pages, page)
		switch page {
		case "1":
			writeJSON(t, w, model.ResultsResponse{
				Data: []model.Workout{{ID: 101}},
				Meta: &model.ResultsMeta{Pagination: &model.Pagination{
					CurrentPage: 1,
					TotalPages:  2,
				}},
			})
		case "2":
			writeJSON(t, w, model.ResultsResponse{
				Data: []model.Workout{{ID: 102}},
				Meta: &model.ResultsMeta{Pagination: &model.Pagination{
					CurrentPage: 2,
					TotalPages:  2,
				}},
			})
		default:
			t.Fatalf("unexpected page %q", page)
		}
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, Token: "secret", HTTPClient: server.Client(), UserAgent: "c2/test"}
	workouts, err := client.GetAllResults(context.Background(), "2026-04-01T00:00:00Z", "2026-04-30T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if len(workouts) != 2 || workouts[0].ID != 101 || workouts[1].ID != 102 {
		t.Fatalf("workouts = %#v, want IDs 101 and 102", workouts)
	}
	if strings.Join(pages, ",") != "1,2" {
		t.Fatalf("pages = %v, want [1 2]", pages)
	}
}

func TestGetStrokes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users/me/results/101/strokes" {
			t.Fatalf("path = %q, want strokes path", r.URL.Path)
		}
		tVal := 12.5
		writeJSON(t, w, model.StrokeDataResponse{Data: []model.StrokeData{{T: &tVal}}})
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, Token: "secret", HTTPClient: server.Client(), UserAgent: "c2/test"}
	strokes, err := client.GetStrokes(context.Background(), 101)
	if err != nil {
		t.Fatal(err)
	}
	if len(strokes) != 1 || strokes[0].T == nil || *strokes[0].T != 12.5 {
		t.Fatalf("strokes = %#v, want one stroke with t=12.5", strokes)
	}
}

func TestAPIErrorIncludesStatusAndPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	}))
	defer server.Close()

	client := &Client{BaseURL: server.URL, Token: "secret", HTTPClient: server.Client(), UserAgent: "c2/test"}
	_, err := client.GetUser(context.Background())
	if err == nil {
		t.Fatal("GetUser returned nil error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "418") || !strings.Contains(msg, "/api/users/me") {
		t.Fatalf("error = %q, want status and path", msg)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatal(err)
	}
}
