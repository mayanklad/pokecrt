package storage

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/mayanklad/pokecrt/internal/trainer"
)

func profileName(t *testing.T, raw string) trainer.Name {
	t.Helper()
	n, err := trainer.ParseName(raw)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestProfilesCreationSelectionAndIsolation(t *testing.T) {
	r, _ := fresh(t)
	ctx := context.Background()
	oak, err := r.CreateProfile(ctx, profileName(t, "  Professor Oak "), 1234)
	if err != nil || !oak.Active || oak.Name != "Professor Oak" || oak.CreatedAtMS != 1234 {
		t.Fatalf("first: %+v, %v", oak, err)
	}
	alice, err := r.CreateProfile(ctx, profileName(t, "Alice"), 4321)
	if err != nil || alice.Active {
		t.Fatalf("second: %+v, %v", alice, err)
	}
	profiles, err := r.ListProfiles(ctx)
	if err != nil || len(profiles) != 2 || profiles[0].Name != "Alice" || profiles[1].Name != "Professor Oak" || !profiles[1].Active {
		t.Fatalf("list: %+v, %v", profiles, err)
	}
	named, err := r.FindProfile(ctx, profileName(t, "ALICE"))
	if err != nil || named.ID != alice.ID || named.Active {
		t.Fatalf("named: %+v, %v", named, err)
	}
	active, err := r.ActiveProfile(ctx)
	if err != nil || active.ID != oak.ID {
		t.Fatalf("view switched active: %+v, %v", active, err)
	}
	if _, err = r.UseProfile(ctx, profileName(t, "missing")); !errors.Is(err, trainer.ErrNotFound) {
		t.Fatalf("unknown: %v", err)
	}
	if _, err = r.CreateProfile(ctx, profileName(t, "PROFESSOR OAK"), 99); !errors.Is(err, trainer.ErrDuplicateName) {
		t.Fatalf("duplicate: %v", err)
	}
	selected, err := r.UseProfile(ctx, profileName(t, "aLiCe"))
	if err != nil || selected.ID != alice.ID || !selected.Active {
		t.Fatalf("selected: %+v, %v", selected, err)
	}
	again, err := r.FindProfile(ctx, profileName(t, "Professor Oak"))
	if err != nil || again.Active || again.CreatedAtMS != 1234 {
		t.Fatalf("other profile mutated: %+v, %v", again, err)
	}
	if integer(t, r, "SELECT count(*) FROM trainer_progress WHERE xp_total = 0") != 2 {
		t.Fatal("profile progress not isolated at zero")
	}
	for _, table := range []string{"encounters", "species_discoveries", "variant_discoveries", "achievement_unlocks"} {
		if integer(t, r, "SELECT count(*) FROM "+table) != 0 {
			t.Fatalf("profiles wrote %s", table)
		}
	}
	// Additional creation must preserve an explicitly empty active choice.
	if err := r.Write(ctx, func(tx *Tx) error {
		_, err := tx.ExecContext(ctx, "UPDATE app_state SET active_trainer_id = NULL")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	third, err := r.CreateProfile(ctx, profileName(t, "Third"), 8)
	if err != nil || third.Active {
		t.Fatalf("later profile autoactivated: %+v, %v", third, err)
	}
	if _, err = r.ActiveProfile(ctx); !errors.Is(err, trainer.ErrNoActive) {
		t.Fatalf("empty active choice changed: %v", err)
	}
}

func TestProfileCanonicalCollisionsAndBoundValues(t *testing.T) {
	r, _ := fresh(t)
	ctx := context.Background()
	for _, pair := range [][2]string{{"Élodie", "E\u0301LODIE"}, {"Straße", "STRASSE"}, {"Σ", "ς"}} {
		if _, err := r.CreateProfile(ctx, profileName(t, pair[0]), 1); err != nil {
			t.Fatal(err)
		}
		if _, err := r.CreateProfile(ctx, profileName(t, pair[1]), 2); !errors.Is(err, trainer.ErrDuplicateName) {
			t.Fatalf("collision %v: %v", pair, err)
		}
	}
	raw := "Oak'; DROP TABLE trainers;--"
	created, err := r.CreateProfile(ctx, profileName(t, raw), 3)
	if err != nil || created.Name != raw || integer(t, r, "SELECT count(*) FROM trainers") != 4 {
		t.Fatalf("name treated as SQL: %+v, %v", created, err)
	}
	if _, err = r.CreateProfile(ctx, trainer.Name{}, 3); !errors.Is(err, trainer.ErrInvalidName) {
		t.Fatalf("zero name accepted: %v", err)
	}
}

func TestProfileCreationRollback(t *testing.T) {
	for _, table := range []string{"trainer_progress", "app_state"} {
		t.Run(table, func(t *testing.T) {
			r, _ := fresh(t)
			ctx := context.Background()
			event := "INSERT"
			if table == "app_state" {
				event = "UPDATE"
			}
			if err := r.Write(ctx, func(tx *Tx) error {
				_, err := tx.ExecContext(ctx, "CREATE TRIGGER fail_profile BEFORE "+event+" ON "+table+" BEGIN SELECT RAISE(ABORT, 'injected failure'); END")
				return err
			}); err != nil {
				t.Fatal(err)
			}
			p, err := r.CreateProfile(ctx, profileName(t, "Mayank"), 1)
			if err == nil || p.ID != 0 {
				t.Fatalf("uncommitted result: %+v, %v", p, err)
			}
			if integer(t, r, "SELECT count(*) FROM trainers") != 0 || integer(t, r, "SELECT count(*) FROM trainer_progress") != 0 || integer(t, r, "SELECT count(*) FROM app_state WHERE active_trainer_id IS NULL") != 1 {
				t.Fatal("partial profile committed")
			}
		})
	}
}

func TestConcurrentProfileCreation(t *testing.T) {
	for _, same := range []bool{false, true} {
		t.Run(map[bool]string{false: "distinct", true: "same canonical name"}[same], func(t *testing.T) {
			r, path := fresh(t)
			ctx := context.Background()
			other, err := Open(ctx, path)
			if err != nil {
				t.Fatal(err)
			}
			defer other.Close()
			names := []trainer.Name{profileName(t, "Oak"), profileName(t, "Alice")}
			if same {
				names[1] = profileName(t, "OAK")
			}
			var wg sync.WaitGroup
			results := make(chan struct {
				p   trainer.Profile
				err error
			}, 2)
			start := make(chan struct{})
			for i, repo := range []*Repository{r, other} {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					p, err := repo.CreateProfile(ctx, names[i], 1)
					results <- struct {
						p   trainer.Profile
						err error
					}{p, err}
				}()
			}
			close(start)
			wg.Wait()
			close(results)
			activated, duplicates := 0, 0
			for result := range results {
				if errors.Is(result.err, trainer.ErrDuplicateName) {
					duplicates++
					continue
				}
				if result.err != nil {
					t.Fatal(result.err)
				}
				if result.p.Active {
					activated++
				}
			}
			want := 2
			if same {
				want = 1
				if duplicates != 1 {
					t.Fatal("canonical race did not reject duplicate")
				}
			}
			if activated != 1 || integer(t, r, "SELECT count(*) FROM trainers") != want || integer(t, r, "SELECT count(*) FROM trainer_progress") != want {
				t.Fatal("concurrent profile initialization was not atomic")
			}
		})
	}
}
