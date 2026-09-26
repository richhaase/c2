package syncer

import (
	"context"
	"os"
	"testing"

	"github.com/richhaase/c2/internal/goals"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/storage"
)

func TestAccountMismatchStopsBeforeChangingRecords(t *testing.T) {
	p, now := syncFixture(t)
	if _, err := storage.AppendWorkouts(p, []models.Workout{{ID: 1, UserID: 1, Date: "2026-07-01", Distance: 5000, Time: 9000}}); err != nil {
		t.Fatal(err)
	}
	if err := goals.Write(p, goals.Collection{Version: 1, Timezone: "UTC", AccountID: 1, Goals: []goals.Goal{}}); err != nil {
		t.Fatal(err)
	}
	workouts, err := os.ReadFile(p.Workouts)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := os.ReadFile(p.Meta)
	if err != nil {
		t.Fatal(err)
	}
	client := &fakeClient{results: []models.Workout{{ID: 2, UserID: 2, Date: "2026-07-02", Distance: 7000}}}
	for _, account := range []int64{1, 2} {
		if _, err := RunForAccount(context.Background(), p, client, now, nil, account); err == nil {
			t.Fatal("mismatch accepted")
		}
		after, err := os.ReadFile(p.Workouts)
		if err != nil {
			t.Fatal(err)
		}
		if string(workouts) != string(after) {
			t.Fatal("mismatch changed workouts")
		}
		after, err = os.ReadFile(p.Meta)
		if err != nil {
			t.Fatal(err)
		}
		if string(meta) != string(after) {
			t.Fatal("mismatch advanced cursor")
		}
	}
}
