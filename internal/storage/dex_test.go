package storage

import (
	"context"
	"reflect"
	"testing"
)

func TestDexRecordsReadOnlyIsolationAndSnapshots(t *testing.T) {
	r, path := fresh(t)
	ctx := context.Background()
	p := encounterProfile(t, r)
	other, err := r.CreateProfile(ctx, profileName(t, "Other"), 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, ms := range []int64{20, 10} {
		if _, err = r.RecordEncounter(ctx, p.ID, encounterChoice(t, 6, "mega-x", "default", "shiny"), ms); err != nil {
			t.Fatal(err)
		}
	}
	beforeXP := integer(t, r, "SELECT SUM(xp_total) FROM trainer_progress")
	beforeUnlocks := integer(t, r, "SELECT COUNT(*) FROM achievement_unlocks")
	reader, err := ReadOnly(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	got, err := reader.DexRecords(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Encounters != 2 || got.Species[6].Count != 2 || got.Species[6].FirstMS != 10 || got.Species[6].LastMS != 20 || len(got.Variants) != 1 || got.Variants[0].Count != 2 || !reflect.DeepEqual(got.Variants[0].Types, []string{"fire", "dragon"}) {
		t.Fatalf("records %+v", got)
	}
	empty, err := reader.DexRecords(ctx, other.ID)
	if err != nil || empty.Encounters != 0 || len(empty.Species)+len(empty.Variants) != 0 {
		t.Fatal("trainer isolation", empty, err)
	}
	if integer(t, r, "SELECT SUM(xp_total) FROM trainer_progress") != beforeXP || integer(t, r, "SELECT COUNT(*) FROM achievement_unlocks") != beforeUnlocks || integer(t, r, "SELECT COUNT(*) FROM encounters") != 2 {
		t.Fatal("browsing mutated state")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err = reader.DexRecords(cancelled, p.ID); err == nil {
		t.Fatal("cancel ignored")
	}
}
