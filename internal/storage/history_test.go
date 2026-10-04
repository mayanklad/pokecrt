package storage

import (
	"context"
	"testing"
)

func TestRecentEncountersSnapshotsIsolationLimitAndOrder(t *testing.T) {
	r, _ := fresh(t)
	ctx := context.Background()
	p := encounterProfile(t, r)
	other, err := r.CreateProfile(ctx, profileName(t, "Other"), 0)
	if err != nil {
		t.Fatal(err)
	}
	c := encounterChoice(t, 25, "standard", "default", "regular")
	for i := 0; i < 55; i++ {
		if _, err := r.RecordEncounter(ctx, p.ID, c, 100); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.RecordEncounter(ctx, other.ID, c, 200); err != nil {
		t.Fatal(err)
	}
	history, err := r.RecentEncounters(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 50 {
		t.Fatalf("got %d rows", len(history))
	}
	for i, h := range history {
		if h.Snapshot.SpeciesName != c.Snapshot().SpeciesName || h.Key != c.Key() || h.EncounteredAtMS != 100 {
			t.Fatalf("bad snapshot %#v", h)
		}
		if i > 0 && h.ID >= history[i-1].ID {
			t.Fatal("unstable tie order")
		}
	}
	rows, err := r.RecentEncounters(ctx, other.ID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("isolation: %v %#v", err, rows)
	}
}
