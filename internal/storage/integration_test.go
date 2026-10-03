package storage

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

// Compare optimized evidence with the historical SQL predicates, including
// shiny-first, repeated and retained snapshots, across isolated profiles.
func TestAchievementEvidenceMatchesHistory(t *testing.T) {
	r, _ := fresh(t)
	ctx := context.Background()
	goals, err := trainer.NewAchievementTargets(catalog.All(), func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
	if err != nil {
		t.Fatal(err)
	}
	for mask := 0; mask < 8; mask++ {
		name, _ := trainer.ParseName(string(rune('A' + mask)))
		p, err := r.CreateProfile(ctx, name, 0)
		if err != nil {
			t.Fatal(err)
		}
		check := func() {
			t.Helper()
			got, err := r.TrainerRecords(ctx, p.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := got.AchievementState
			if err := r.db.QueryRowContext(ctx, `SELECT
 EXISTS(SELECT 1 FROM encounters WHERE trainer_id=? AND palette='shiny'),
 EXISTS(SELECT 1 FROM encounters WHERE trainer_id=? AND regional_snapshot=1),
 EXISTS(SELECT 1 FROM encounters WHERE trainer_id=? AND transformation_snapshot=1)`, p.ID, p.ID, p.ID).Scan(&want.Shiny, &want.Regional, &want.Transformation); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.AchievementState, want) || !reflect.DeepEqual(goals.Goals(got.AchievementState), goals.Goals(want)) {
				t.Fatalf("mask %d: evidence or goals differ", mask)
			}
		}
		check() // Another profile's discoveries cannot supply flags to an empty one.
		palette := "regular"
		if mask&1 != 0 {
			palette = "shiny"
		}
		choice := encounterChoice(t, 25, "standard", "default", palette)
		for i := 0; i < 3; i++ {
			if _, err := r.RecordEncounter(ctx, p.ID, choice, int64(i)); err != nil {
				t.Fatal(err)
			}
		}
		// Retain an older form classification/type snapshot even when today's
		// catalog differs. Later repeated encounters must not erase this evidence.
		if err := r.Write(ctx, func(tx *Tx) error {
			_, err := tx.ExecContext(ctx, `UPDATE encounters SET regional_snapshot=?,transformation_snapshot=?,type1_snapshot='water',type2_snapshot='ice' WHERE trainer_id=? AND encountered_at_ms=0`, mask&2 != 0, mask&4 != 0, p.ID)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		check()
		if _, err := r.RecordEncounter(ctx, p.ID, choice, 3); err != nil {
			t.Fatal(err)
		}
		check()
		assertEncounterSums(t, r)
	}
}

// Exercise future migration/backup mechanics with actual gameplay state;
// schema 002 remains a test fixture, not a production migration.
func TestGameplaySurvivesUpgradeAndBackup(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "rollback"}[fail], func(t *testing.T) {
			r, path := fresh(t)
			ctx := context.Background()
			p := encounterProfile(t, r)
			for _, palette := range []string{"shiny", "regular", "shiny"} {
				if _, err := r.RecordEncounter(ctx, p.ID, encounterChoice(t, 25, "standard", "default", palette), 10); err != nil {
					t.Fatal(err)
				}
			}
			before, err := r.TrainerRecords(ctx, p.ID)
			if err != nil {
				t.Fatal(err)
			}
			profiles, err := r.ListProfiles(ctx)
			if err != nil {
				t.Fatal(err)
			}
			active, err := r.ActiveProfile(ctx)
			if err != nil {
				t.Fatal(err)
			}
			script := "CREATE TABLE integration_upgrade (id INTEGER);"
			if fail {
				script += " UPDATE trainer_progress SET xp_total=0; INSERT INTO absent_table VALUES(1);"
			}
			err = r.migrate(ctx, fstest.MapFS{"migrations/002_test.sql": {Data: []byte(script)}}, 2)
			if (err != nil) != fail {
				t.Fatal(err)
			}
			backups, _ := filepath.Glob(path + ".backup-v1-to-v2-*")
			if len(backups) != 1 {
				t.Fatal(backups)
			}
			backup, err := ReadOnly(ctx, backups[0])
			if err != nil {
				t.Fatal(err)
			}
			defer backup.Close()
			for _, repo := range []*Repository{r, backup} {
				after, err := repo.TrainerRecords(ctx, p.ID)
				if err != nil || !reflect.DeepEqual(before, after) {
					t.Fatalf("gameplay changed: %v", err)
				}
				ps, err := repo.ListProfiles(ctx)
				if err != nil || !reflect.DeepEqual(profiles, ps) {
					t.Fatal("profiles changed", err)
				}
				ap, err := repo.ActiveProfile(ctx)
				if err != nil || ap != active {
					t.Fatal("active profile changed", err)
				}
				assertEncounterSums(t, repo)
			}
		})
	}
}
