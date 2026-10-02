package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/render"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

func encounterChoice(t *testing.T, id int, formID, gender, palette string) trainer.Choice {
	t.Helper()
	s, ok := catalog.ByNumber(id)
	if !ok {
		t.Fatal("species missing")
	}
	for _, f := range s.Forms {
		if f.ID != formID {
			continue
		}
		f.Genders = []string{gender}
		s.Forms = []catalog.Form{f}
		p, err := trainer.NewPool([]catalog.Species{s}, func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
		if err != nil {
			t.Fatal(err)
		}
		choice, err := p.Select(func(n int) (int, error) {
			if n == 4096 && palette == "regular" {
				return 1, nil
			}
			return 0, nil
		})
		if err != nil || choice.Key().Palette != palette {
			t.Fatalf("choice: %+v %v", choice.Key(), err)
		}
		return choice
	}
	t.Fatalf("form %d/%s missing", id, formID)
	return trainer.Choice{}
}

func encounterProfile(t *testing.T, r *Repository) trainer.Profile {
	t.Helper()
	p, err := r.CreateProfile(context.Background(), profileName(t, "Mayank"), 1)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func assertEncounterSums(t *testing.T, r *Repository) {
	t.Helper()
	queries := []string{
		"SELECT count(*) FROM pragma_foreign_key_check",
		`SELECT count(*) FROM trainer_progress p WHERE p.xp_total != COALESCE((SELECT sum(e.xp_awarded) FROM encounters e WHERE e.trainer_id=p.trainer_id),0)`,
		`SELECT count(*) FROM species_discoveries d WHERE d.encounter_count != (SELECT count(*) FROM encounters e WHERE e.trainer_id=d.trainer_id AND e.species_id=d.species_id)`,
		`SELECT count(*) FROM variant_discoveries d WHERE d.encounter_count != (SELECT count(*) FROM encounters e WHERE e.trainer_id=d.trainer_id AND e.species_id=d.species_id AND e.form_id=d.form_id AND e.gender_key=d.gender_key AND e.palette=d.palette)`,
		`SELECT count(*) FROM (SELECT trainer_id,species_id,sum(first_species) n FROM encounters GROUP BY trainer_id,species_id) WHERE n!=1`,
		`SELECT count(*) FROM (SELECT trainer_id,species_id,form_id,gender_key,palette,sum(first_variant) n FROM encounters GROUP BY trainer_id,species_id,form_id,gender_key,palette) WHERE n!=1`,
	}
	for _, q := range queries {
		if n := integer(t, r, q); n != 0 {
			t.Fatalf("invariant failed (%d): %s", n, q)
		}
	}
}

func TestEncounterShinyFirstRepeatedVariantAndClock(t *testing.T) {
	r, _ := fresh(t)
	ctx := context.Background()
	profile := encounterProfile(t, r)
	shiny := encounterChoice(t, 25, "standard", "default", "shiny")
	regular := encounterChoice(t, 25, "standard", "default", "regular")
	first, err := r.RecordEncounter(ctx, profile.ID, shiny, 100)
	if err != nil || !first.FirstSpecies || !first.FirstVariant || first.Species != 1 || first.Variants != 1 || first.Encounters != 1 {
		t.Fatalf("first: %+v %v", first, err)
	}
	if integer(t, r, "SELECT count(*) FROM variant_discoveries WHERE palette='regular'") != 0 {
		t.Fatal("shiny first inferred regular")
	}
	second, err := r.RecordEncounter(ctx, profile.ID, regular, 50)
	if err != nil || second.FirstSpecies || !second.FirstVariant || second.Variants != 2 {
		t.Fatalf("regular: %+v %v", second, err)
	}
	repeat, err := r.RecordEncounter(ctx, profile.ID, shiny, 75)
	if err != nil || repeat.FirstSpecies || repeat.FirstVariant || repeat.Encounters != 3 {
		t.Fatalf("repeat: %+v %v", repeat, err)
	}
	if integer(t, r, "SELECT first_seen_ms FROM species_discoveries") != 50 || integer(t, r, "SELECT last_seen_ms FROM species_discoveries") != 100 {
		t.Fatal("clock MIN/MAX")
	}
	if integer(t, r, "SELECT first_seen_ms FROM variant_discoveries WHERE palette='shiny'") != 75 || integer(t, r, "SELECT last_seen_ms FROM variant_discoveries WHERE palette='shiny'") != 100 {
		t.Fatal("variant clock MIN/MAX")
	}
	rows, err := r.QueryContext(ctx, "SELECT encountered_at_ms FROM encounters ORDER BY encountered_at_ms DESC,id DESC")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	want := []int64{100, 75, 50}
	i := 0
	for rows.Next() {
		var stamp int64
		if err := rows.Scan(&stamp); err != nil {
			t.Fatal(err)
		}
		if stamp != want[i] {
			t.Fatal("history time mutated")
		}
		i++
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if i != 3 {
		t.Fatal("missing history")
	}
	assertEncounterSums(t, r)
}

func TestEncounterSnapshotsAndProfileIsolation(t *testing.T) {
	r, _ := fresh(t)
	ctx := context.Background()
	one := encounterProfile(t, r)
	two, err := r.CreateProfile(ctx, profileName(t, "Oak"), 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		choice trainer.Choice
		id     int64
	}{
		{encounterChoice(t, 6, "mega-x", "default", "regular"), one.ID},
		{encounterChoice(t, 26, "alola", "default", "regular"), one.ID},
		{encounterChoice(t, 678, "standard", "female", "shiny"), two.ID},
	} {
		record, err := r.RecordEncounter(ctx, tc.id, tc.choice, 123)
		if err != nil {
			t.Fatal(err)
		}
		var dataset, speciesName, formName, type1 string
		var type2 sql.NullString
		var regional, transformation bool
		if err := r.QueryRowContext(ctx, "SELECT dataset_id,species_name_snapshot,form_name_snapshot,type1_snapshot,type2_snapshot,regional_snapshot,transformation_snapshot FROM encounters WHERE id=?", record.ID).Scan(&dataset, &speciesName, &formName, &type1, &type2, &regional, &transformation); err != nil {
			t.Fatal(err)
		}
		snapshot := tc.choice.Snapshot()
		if dataset != catalog.DatasetID || speciesName != snapshot.SpeciesName || formName != snapshot.FormName || type1 != snapshot.Type1 || type2.String != snapshot.Type2 || type2.Valid != (snapshot.Type2 != "") || regional != snapshot.Regional || transformation != snapshot.Transformation {
			t.Fatal("snapshot differs")
		}
	}
	if integer(t, r, fmt.Sprintf("SELECT count(*) FROM encounters WHERE trainer_id=%d", one.ID)) != 2 || integer(t, r, fmt.Sprintf("SELECT count(*) FROM encounters WHERE trainer_id=%d", two.ID)) != 1 {
		t.Fatal("profile histories mixed")
	}
	assertEncounterSums(t, r)
	if _, err := r.RecordEncounter(ctx, 99999, encounterChoice(t, 25, "standard", "default", "regular"), 1); !errors.Is(err, trainer.ErrNotFound) {
		t.Fatalf("missing trainer: %v", err)
	}
	if _, err := r.RecordEncounter(ctx, one.ID, trainer.Choice{}, 1); err == nil {
		t.Fatal("zero choice accepted")
	}
	if integer(t, r, "SELECT count(*) FROM encounters") != 3 {
		t.Fatal("invalid record mutated history")
	}
}

func TestEncounterFailureRollsBackAllStages(t *testing.T) {
	for _, table := range []string{"encounters", "species_discoveries", "variant_discoveries"} {
		t.Run(table, func(t *testing.T) {
			r, _ := fresh(t)
			ctx := context.Background()
			profile := encounterProfile(t, r)
			if err := r.Write(ctx, func(tx *Tx) error {
				_, err := tx.ExecContext(ctx, "CREATE TRIGGER fail_record BEFORE INSERT ON "+table+" BEGIN SELECT RAISE(ABORT,'injected failure'); END")
				return err
			}); err != nil {
				t.Fatal(err)
			}
			record, err := r.RecordEncounter(ctx, profile.ID, encounterChoice(t, 25, "standard", "default", "regular"), 1)
			if err == nil || record.ID != 0 {
				t.Fatal("failed recording returned success")
			}
			for _, table := range []string{"encounters", "species_discoveries", "variant_discoveries", "achievement_unlocks"} {
				if integer(t, r, "SELECT count(*) FROM "+table) != 0 {
					t.Fatalf("partial %s persisted", table)
				}
			}
			assertEncounterSums(t, r)
		})
	}
	// Trigger fails the repeat upsert after history insertion: existing state stays.
	r, _ := fresh(t)
	ctx := context.Background()
	p := encounterProfile(t, r)
	c := encounterChoice(t, 25, "standard", "default", "regular")
	if _, err := r.RecordEncounter(ctx, p.ID, c, 1); err != nil {
		t.Fatal(err)
	}
	if err := r.Write(ctx, func(tx *Tx) error {
		_, err := tx.ExecContext(ctx, "CREATE TRIGGER fail_repeat BEFORE UPDATE ON variant_discoveries BEGIN SELECT RAISE(ABORT,'repeat failed'); END")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.RecordEncounter(ctx, p.ID, c, 2); err == nil {
		t.Fatal("repeat failure ignored")
	}
	if integer(t, r, "SELECT count(*) FROM encounters") != 1 || integer(t, r, "SELECT encounter_count FROM species_discoveries") != 1 {
		t.Fatal("repeat failure partially committed")
	}
	assertEncounterSums(t, r)
}

func TestConcurrentFirstEncountersAndCapturedSwitch(t *testing.T) {
	r, path := fresh(t)
	ctx := context.Background()
	one := encounterProfile(t, r)
	other, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	c := encounterChoice(t, 25, "standard", "default", "shiny")
	results := make(chan trainer.Record, 2)
	errs := make(chan error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, repo := range []*Repository{r, other} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			record, err := repo.RecordEncounter(ctx, one.ID, c, 100)
			results <- record
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	firstSpecies, firstVariant := 0, 0
	for record := range results {
		if record.FirstSpecies {
			firstSpecies++
		}
		if record.FirstVariant {
			firstVariant++
		}
	}
	if firstSpecies != 1 || firstVariant != 1 || integer(t, r, "SELECT count(*) FROM encounters") != 2 || integer(t, r, "SELECT encounter_count FROM variant_discoveries") != 2 {
		t.Fatal("concurrent first flags/counts")
	}
	if integer(t, r, "SELECT xp_total FROM trainer_progress") != 280 || integer(t, r, "SELECT count(*) FROM achievement_unlocks") != 2 {
		t.Fatal("concurrent XP/unlocks duplicated")
	}
	two, err := r.CreateProfile(ctx, profileName(t, "Oak"), 2)
	if err != nil {
		t.Fatal(err)
	}
	species, _ := catalog.ByNumber(25)
	pool, err := trainer.NewPool([]catalog.Species{species}, func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
	if err != nil {
		t.Fatal(err)
	}
	service := trainer.EncounterService{Repository: r, Pool: pool, Index: func(int) (int, error) { return 0, nil }, Clock: func() time.Time { return time.UnixMilli(101) }, Prepare: func(choice trainer.Choice) ([]byte, error) {
		_, err := other.UseProfile(ctx, profileName(t, "Oak"))
		if err != nil {
			return nil, err
		}
		pixels, err := sprite.Decode(choice.Key())
		if err != nil {
			return nil, err
		}
		return render.Render(pixels, true)
	}}
	result, err := service.Encounter(ctx)
	if err != nil || result.Record.TrainerID != one.ID {
		t.Fatalf("switch redirected action: %+v %v", result.Record, err)
	}
	active, err := r.ActiveProfile(ctx)
	if err != nil || active.ID != two.ID {
		t.Fatal("switch not preserved")
	}
	if integer(t, r, fmt.Sprintf("SELECT count(*) FROM encounters WHERE trainer_id=%d", two.ID)) != 0 {
		t.Fatal("new active received old action")
	}
	assertEncounterSums(t, r)
}

func TestEncounterCommitFailureAndCancellation(t *testing.T) {
	r, _ := fresh(t)
	ctx := context.Background()
	p := encounterProfile(t, r)
	c := encounterChoice(t, 25, "standard", "default", "regular")
	if err := r.Write(ctx, func(tx *Tx) error {
		_, err := tx.ExecContext(ctx, "CREATE TRIGGER fail_commit AFTER INSERT ON encounters BEGIN INSERT INTO trainer_progress VALUES (999999, 0); END")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	// Force a deferred foreign-key error at COMMIT, after all recording stages.
	if _, err := r.db.ExecContext(ctx, "PRAGMA defer_foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	result, err := r.RecordEncounter(ctx, p.ID, c, 1)
	if err == nil || result.ID != 0 {
		t.Fatal("failed commit returned success")
	}
	for _, table := range []string{"encounters", "species_discoveries", "variant_discoveries"} {
		if integer(t, r, "SELECT count(*) FROM "+table) != 0 {
			t.Fatal("failed commit persisted " + table)
		}
	}
	if integer(t, r, "SELECT count(*) FROM trainer_progress") != 1 {
		t.Fatal("deferred orphan persisted")
	}
	assertEncounterSums(t, r)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := r.RecordEncounter(canceled, p.ID, c, 2); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestEncounterProgressionAwardsUnlocksAndRollback(t *testing.T) {
	r, _ := fresh(t)
	p := encounterProfile(t, r)
	ctx := context.Background()
	regular := encounterChoice(t, 25, "standard", "default", "regular")
	shiny := encounterChoice(t, 25, "standard", "default", "shiny")
	for i, c := range []struct {
		choice trainer.Choice
		award  int64
	}{{regular, 70}, {regular, 10}, {shiny, 130}, {shiny, 110}} {
		record, err := r.RecordEncounter(ctx, p.ID, c.choice, int64(i+10))
		if err != nil {
			t.Fatal(err)
		}
		if record.XPAwarded != c.award || record.After.Total-record.Before.Total != c.award {
			t.Fatalf("XP %+v", record)
		}
		if i == 0 {
			u := record.NewUnlocks()
			if len(u) != 1 || u[0].ID != "encounters.1" {
				t.Fatalf("first unlocks %+v", u)
			}
			u[0].ID = "mutated"
			if record.NewUnlocks()[0].ID != "encounters.1" {
				t.Fatal("unlock slice aliases")
			}
		}
		if i == 2 {
			u := record.NewUnlocks()
			if len(u) != 1 || u[0].ID != "shiny.first" {
				t.Fatalf("shiny unlocks %+v", u)
			}
		}
		if i == 1 || i == 3 {
			if len(record.NewUnlocks()) != 0 {
				t.Fatal("duplicate unlock")
			}
		}
		assertEncounterSums(t, r)
	}
	// A newly introduced, already satisfied goal unlocks only on the next write.
	if _, err := r.db.Exec("DELETE FROM achievement_unlocks WHERE achievement_id='encounters.1'"); err != nil {
		t.Fatal(err)
	}
	record, err := r.RecordEncounter(ctx, p.ID, regular, 99)
	if err != nil {
		t.Fatal(err)
	}
	if u := record.NewUnlocks(); len(u) != 1 || u[0].ID != "encounters.1" || u[0].UnlockedAtMS != 99 || u[0].Target != 1 {
		t.Fatalf("new definition %+v", u)
	}
	for _, trigger := range []string{
		`CREATE TRIGGER fail_xp BEFORE UPDATE ON trainer_progress BEGIN SELECT RAISE(ABORT,'XP failed'); END`,
		`CREATE TRIGGER fail_unlock BEFORE INSERT ON achievement_unlocks BEGIN SELECT RAISE(ABORT,'unlock failed'); END`,
	} {
		before := integer(t, r, "SELECT count(*) FROM encounters")
		xp := integer(t, r, "SELECT xp_total FROM trainer_progress")
		if _, err := r.db.Exec(trigger); err != nil {
			t.Fatal(err)
		}
		if _, err := r.RecordEncounter(ctx, p.ID, regular, 100); err == nil {
			t.Fatal("failure accepted")
		}
		if integer(t, r, "SELECT count(*) FROM encounters") != before || integer(t, r, "SELECT xp_total FROM trainer_progress") != xp {
			t.Fatal("partial transaction")
		}
		name := "fail_xp"
		if strings.Contains(trigger, "CREATE TRIGGER fail_unlock ") {
			name = "fail_unlock"
		}
		if _, err := r.db.Exec("DROP TRIGGER " + name); err != nil {
			t.Fatal(err)
		}
		assertEncounterSums(t, r)
	}
}

func TestEncounterXPOverflowAndLevelBoundary(t *testing.T) {
	r, _ := fresh(t)
	p := encounterProfile(t, r)
	ctx := context.Background()
	choice := encounterChoice(t, 25, "standard", "default", "regular")
	if _, err := r.db.Exec("UPDATE trainer_progress SET xp_total=990"); err != nil {
		t.Fatal(err)
	}
	record, err := r.RecordEncounter(ctx, p.ID, choice, 10)
	if err != nil {
		t.Fatal(err)
	}
	if record.Before.Level != 1 || record.After.Level != 2 || record.After.InLevel != 60 {
		t.Fatalf("level %+v", record)
	}
	if _, err := r.db.Exec("UPDATE trainer_progress SET xp_total=9223372036854775807"); err != nil {
		t.Fatal(err)
	}
	before := integer(t, r, "SELECT count(*) FROM encounters")
	record, err = r.RecordEncounter(ctx, p.ID, choice, 11)
	if err == nil || record.ID != 0 {
		t.Fatal("overflow accepted")
	}
	if integer(t, r, "SELECT count(*) FROM encounters") != before {
		t.Fatal("overflow wrote history")
	}
}
