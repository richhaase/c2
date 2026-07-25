package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	client, err := newClient(server.URL, "secret-token", "v1.2.3", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestNewRejectsUnsafeOrMalformedBaseURLs(t *testing.T) {
	for _, raw := range []string{
		"",
		"http://log.concept2.com",
		"https://",
		"https://user:pass@log.concept2.com",
		"https://log.concept2.com/api",
		"https://log.concept2.com?debug=true",
		"https://log.concept2.com#fragment",
	} {
		if _, err := New(raw, "token", "dev"); err == nil {
			t.Errorf("New(%q) succeeded", raw)
		}
	}
}

func TestGetUserSendsRequiredHeaders(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users/me" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("User-Agent"); got != "c2/v1.2.3" {
			t.Errorf("User-Agent = %q", got)
		}
		if got := r.Header.Get("Accept"); got != acceptType {
			t.Errorf("Accept = %q", got)
		}
		fmt.Fprint(w, `{"data":{"id":7,"username":"rower"}}`)
	})
	user, err := client.GetUser(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != 7 || user.Username != "rower" {
		t.Fatalf("user = %#v", user)
	}
}

func TestGetAllResultsPaginatesAndEncodesFilters(t *testing.T) {
	pages := 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		pages++
		query := r.URL.Query()
		if query.Get("type") != "rower" {
			t.Errorf("type = %q", query.Get("type"))
		}
		if query.Get("from") != "2026-01-01" {
			t.Errorf("from = %q", query.Get("from"))
		}
		if query.Get("to") != "2026-12-31" {
			t.Errorf("to = %q", query.Get("to"))
		}
		if query.Get("updated_after") != "2026-07-20 12:34:56" {
			t.Errorf("updated_after = %q", query.Get("updated_after"))
		}
		if query.Get("page") == "1" {
			fmt.Fprint(w, `{"data":[{"id":1,"date":"2026-07-20","distance":1000,"time":1000}],"meta":{"pagination":{"current_page":1,"total_pages":2}}}`)
			return
		}
		fmt.Fprint(w, `{"data":[{"id":2,"date":"2026-07-20","distance":1000,"time":1000}],"meta":{"pagination":{"current_page":2,"total_pages":2}}}`)
	})
	workouts, err := client.GetAllResults(context.Background(), ResultsFilter{
		From:         "2026-01-01",
		To:           "2026-12-31",
		UpdatedAfter: "2026-07-20 12:34:56",
	})
	if err != nil {
		t.Fatal(err)
	}
	if pages != 2 || len(workouts) != 2 || workouts[0].ID != 1 || workouts[1].ID != 2 {
		t.Fatalf("pages = %d, workouts = %#v", pages, workouts)
	}
}

func TestGetStrokesAndAPIErrors(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/99/strokes") {
			fmt.Fprint(w, `{"data":[{"t":1,"p":1750}]}`)
			return
		}
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	})
	strokes, err := client.GetStrokes(context.Background(), 99)
	if err != nil {
		t.Fatal(err)
	}
	if len(strokes) != 1 || strokes[0].T == nil || *strokes[0].T != 1 {
		t.Fatalf("strokes = %#v", strokes)
	}
	if _, err := client.GetUser(context.Background()); err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("error = %v", err)
	}
}

func TestRequestHonorsContextCancellation(t *testing.T) {
	started := make(chan struct{})
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := client.GetUser(ctx)
		done <- err
	}()
	<-started
	cancel()
	if err := <-done; err == nil {
		t.Fatal("GetUser succeeded")
	}
}
