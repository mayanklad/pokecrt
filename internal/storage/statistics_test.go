package storage

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"testing"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

func TestTrainerReadStatisticsAndAchievementsNoMutation(t *testing.T) {
	r, path := fresh(t)
	ctx := context.Background()
	p := encounterProfile(t, r)
	other, err := r.CreateProfile(ctx, profileName(t, "Other"), 2)
	if err != nil {
		t.Fatal(err)
	}
	for i, ms := range []int64{30, 0, 20} {
		palette := "shiny"
		if i == 2 {
			palette = "regular"
		}
		if _, err = r.RecordEncounter(ctx, p.ID, encounterChoice(t, 6, "mega-x", "default", palette), ms); err != nil {
			t.Fatal(err)
		}
	}
	// Simulate a newly introduced definition already satisfied by old history.
	if _, err = r.db.Exec("DELETE FROM achievement_unlocks WHERE trainer_id=? AND achievement_id='transformation.first'", p.ID); err != nil {
		t.Fatal(err)
	}
	beforeXP := integer(t, r, "SELECT SUM(xp_total) FROM trainer_progress")
	beforeUnlocks := integer(t, r, "SELECT COUNT(*) FROM achievement_unlocks")
	beforeBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	beforeHash := sha256.Sum256(beforeBytes)
	reader, err := ReadOnly(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	records, err := reader.TrainerRecords(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if records.XP != 310 || records.Dex.Encounters != 3 || records.ShinyEncounters != 2 || len(records.Dex.Species) != 1 || len(records.Dex.Variants) != 2 || *records.FirstEncounterMS != 0 || *records.LastEncounterMS != 30 {
		t.Fatalf("records %+v", records)
	}
	targets, err := trainer.NewAchievementTargets(catalog.All(), func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
	if err != nil {
		t.Fatal(err)
	}
	views := targets.Views(records.AchievementState, records.Unlocks)
	found := false
	for _, v := range views.Locked {
		if v.ID == "transformation.first" {
			found = true
			if v.Current != 1 || v.Target != 1 {
				t.Fatal(v)
			}
		}
	}
	if !found {
		t.Fatal("browse awarded ready achievement")
	}
	empty, err := reader.TrainerRecords(ctx, other.ID)
	if err != nil || empty.XP != 0 || empty.Dex.Encounters != 0 || len(empty.Unlocks) != 0 || empty.FirstEncounterMS != nil {
		t.Fatal(empty, err)
	}
	if _, err = reader.TrainerRecords(ctx, 99999); !errors.Is(err, trainer.ErrNotFound) {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err = reader.TrainerRecords(cancelled, p.ID); err == nil {
		t.Fatal("cancel ignored")
	}
	reader.Close()
	afterBytes, _ := os.ReadFile(path)
	if sha256.Sum256(afterBytes) != beforeHash || integer(t, r, "SELECT SUM(xp_total) FROM trainer_progress") != beforeXP || integer(t, r, "SELECT COUNT(*) FROM achievement_unlocks") != beforeUnlocks {
		t.Fatal("read changed persisted state")
	}
	assertEncounterSums(t, r)
	// Award evaluation still happens only on the next committed encounter.
	next, err := r.RecordEncounter(ctx, p.ID, encounterChoice(t, 6, "mega-x", "default", "regular"), 40)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, u := range next.NewUnlocks() {
		if u.ID == "transformation.first" {
			found = true
			if u.UnlockedAtMS != 40 {
				t.Fatal(u)
			}
		}
	}
	if !found {
		t.Fatal("next encounter did not award pending goal")
	}
}

func TestTrainerRecordsConsistentDuringEncounters(t *testing.T) {
	writer, path := fresh(t)
	ctx := context.Background()
	p := encounterProfile(t, writer)
	reader, err := ReadOnly(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	choice := encounterChoice(t, 1, "standard", "default", "regular")
	errors := make(chan error, 1)
	start := make(chan struct{})
	go func() {
		<-start
		for i := int64(1); i <= 12; i++ {
			if _, err := writer.RecordEncounter(ctx, p.ID, choice, i); err != nil {
				errors <- err
				return
			}
		}
		errors <- nil
	}()
	close(start)
	for i := 0; i < 12; i++ {
		records, err := reader.TrainerRecords(ctx, p.ID)
		if err != nil {
			t.Fatal(err)
		}
		n := records.Dex.Encounters
		expected := int64(0)
		if n > 0 {
			expected = 60 + 10*n
		}
		if records.XP != expected || records.Dex.Species[1].Count != n {
			t.Fatalf("mixed snapshots: n=%d XP=%d discoveries=%d", n, records.XP, records.Dex.Species[1].Count)
		}
	}
	if err := <-errors; err != nil {
		t.Fatal(err)
	}
	records, err := reader.TrainerRecords(ctx, p.ID)
	if err != nil || records.Dex.Encounters != 12 || records.XP != 180 {
		t.Fatal(records, err)
	}
}
