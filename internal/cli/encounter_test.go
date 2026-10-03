package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/render"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

func selectedEncounterPool(t *testing.T, id int, formID, gender string) *trainer.Pool {
	t.Helper()
	species, ok := catalog.ByNumber(id)
	if !ok {
		t.Fatal("species missing")
	}
	for _, form := range species.Forms {
		if form.ID == formID {
			form.Genders = []string{gender}
			species.Forms = []catalog.Form{form}
			pool, err := trainer.NewPool([]catalog.Species{species}, func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
			if err != nil {
				t.Fatal(err)
			}
			return pool
		}
	}
	t.Fatal("form missing")
	return nil
}

func fixedEncounterFactory(t *testing.T, id int, form, gender string, shiny bool) encounterFactory {
	t.Helper()
	pool := selectedEncounterPool(t, id, form, gender)
	return func(repo encounterRepository) (trainer.EncounterService, error) {
		return trainer.EncounterService{
			Repository: repo, Pool: pool, Index: func(n int) (int, error) {
				if n == 4096 && !shiny {
					return 1, nil
				}
				return 0, nil
			}, Clock: func() time.Time { return time.UnixMilli(1234) },
			Prepare: func(choice trainer.Choice) ([]byte, error) {
				pixels, err := sprite.Decode(choice.Key())
				if err != nil {
					return nil, err
				}
				return render.Render(pixels, false)
			},
		}, nil
	}
}

func encounterPath(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state", "trainers.sqlite3")
	repo, err := storage.Initialize(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	name, err := trainer.ParseName("Mayank")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateProfile(context.Background(), name, 1); err != nil {
		t.Fatal(err)
	}
	if err := repo.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}
func testEncounterOpen(ctx context.Context, path string) (encounterRepository, error) {
	return storage.Open(ctx, path)
}
func invokeEncounter(t *testing.T, path string, args []string, out io.Writer, factory encounterFactory) (int, string) {
	t.Helper()
	var errout bytes.Buffer
	status := executeEncounter(args, out, &errout, func() (string, error) { return path, nil }, testEncounterOpen, factory)
	return status, errout.String()
}
func readEncounterState(t *testing.T, path string) []int64 {
	t.Helper()
	repo, err := storage.ReadOnly(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	var values [8]int64
	err = repo.QueryRowContext(context.Background(), `SELECT
 (SELECT count(*) FROM encounters),(SELECT COALESCE(sum(xp_awarded),0) FROM encounters),
 (SELECT count(*) FROM species_discoveries),(SELECT count(*) FROM variant_discoveries),
 (SELECT count(*) FROM achievement_unlocks),(SELECT COALESCE(sum(xp_total),0) FROM trainer_progress),
 (SELECT COALESCE(sum(first_species),0) FROM encounters),(SELECT COALESCE(sum(first_variant),0) FROM encounters)`).Scan(&values[0], &values[1], &values[2], &values[3], &values[4], &values[5], &values[6], &values[7])
	if err != nil {
		t.Fatal(err)
	}
	return values[:]
}

func TestEncounterSyntaxHelpBeforeStorage(t *testing.T) {
	cases := []struct {
		args   []string
		status int
	}{{[]string{"--help"}, 0}, {[]string{"-h"}, 0}, {[]string{"--output", "sprite", "--help"}, 0}}
	for _, args := range [][]string{{"--output"}, {"--output="}, {"--output", "unknown"}, {"--output", "full", "--output", "sprite"}, {"--help", "-h"}, {"--help", "--unknown"}, {"--name", "charizard"}, {"--gen", "1"}, {"--form", "mega-x"}, {"--shiny"}, {"--gender", "female"}, {"--sprite-only"}, {"--help", "--output", "invalid"}, {"-output", "sprite"}, {"extra"}, {"--", "extra"}} {
		cases = append(cases, struct {
			args   []string
			status int
		}{args, 2})
	}
	for _, c := range cases {
		t.Run(strings.Join(c.args, " "), func(t *testing.T) {
			touched := false
			var out, errout bytes.Buffer
			status := executeEncounter(c.args, &out, &errout, func() (string, error) { touched = true; return "", errors.New("unexpected storage") }, func(context.Context, string) (encounterRepository, error) {
				touched = true
				return nil, errors.New("unexpected storage")
			}, func(encounterRepository) (trainer.EncounterService, error) {
				touched = true
				return trainer.EncounterService{}, errors.New("unexpected service")
			})
			if touched || status != c.status {
				t.Fatalf("status %d touched %v %q", status, touched, errout.String())
			}
			if c.status == 0 && (out.Len() == 0 || errout.Len() != 0) {
				t.Fatal("help streams")
			}
			if c.status == 2 && (out.Len() != 0 || errout.Len() == 0) {
				t.Fatal("syntax streams")
			}
		})
	}
}

func TestEncounterFormatterFiveModesSameRecord(t *testing.T) {
	pool := selectedEncounterPool(t, 6, "mega-x", "default")
	choice, err := pool.Select(func(int) (int, error) { return 0, nil })
	if err != nil {
		t.Fatal(err)
	}
	record := trainer.Record{Choice: choice, FirstVariant: true, XPAwarded: 130, Before: trainer.Progress{Level: 1}, After: trainer.Progress{Level: 3}, Completion: trainer.Completion{Species: 3, SpeciesTotal: 1017, Variants: 4, VariantsTotal: 2669}}.WithUnlocks([]trainer.Unlock{{Name: "10 Encounters", Description: "Recorded 10 encounters."}, {Name: "Shiny Discovery", Description: "Record a shiny encounter."}})
	artwork := []byte("[SPRITE]\n")
	identity := "\n#006 Charizard · Mega X · Shiny\n"
	typing := "Fire / Dragon\n"
	progress := "\nNEW VARIANT DISCOVERED\nSHINY ENCOUNTER\nShiny variant collected; regular variant not yet collected.\nPokédex: 3 / 1017\nVariants: 4 / 2669\nXP gained: 130\nLEVEL UP - 1 → 3\n"
	achievements := "\n🏆 10 Encounters\nRecorded 10 encounters.\n\n🏆 Shiny Discovery\nRecord a shiny encounter.\n"
	for _, c := range []struct{ mode, want string }{{"full", string(artwork) + identity + typing + progress + achievements}, {"compact", string(artwork) + identity}, {"no-title", string(artwork) + progress + achievements}, {"achievements", string(artwork) + achievements}, {"sprite", string(artwork)}} {
		if got := string(formatEncounter(record, artwork, c.mode)); got != c.want {
			t.Fatalf("%s:\n%s\nwant:\n%s", c.mode, got, c.want)
		}
	}
	for _, unlocks := range [][]trainer.Unlock{nil, {{Name: "First Contact", Description: "Recorded 1 encounters."}}, record.NewUnlocks()} {
		got := formatEncounter(record.WithUnlocks(unlocks), artwork, "achievements")
		if bytes.Count(got, []byte("🏆 ")) != len(unlocks) {
			t.Fatal("achievement notices")
		}
		if len(unlocks) == 0 && !bytes.Equal(got, artwork) {
			t.Fatal("empty achievements mode is not sprite only")
		}
	}
}

func TestEncounterFormattingDiscoveryCasesAndAppearance(t *testing.T) {
	for _, c := range []struct {
		firstSpecies, firstVariant, shiny, regular bool
		notice                                     string
		xp                                         int64
	}{{true, true, false, false, "NEW SPECIES DISCOVERED", 70}, {false, true, false, false, "NEW VARIANT DISCOVERED", 30}, {false, false, false, true, "REPEAT ENCOUNTER", 10}, {true, true, true, false, "NEW SPECIES DISCOVERED", 170}, {false, true, true, true, "NEW VARIANT DISCOVERED", 130}, {false, false, true, false, "REPEAT ENCOUNTER", 110}} {
		pool := selectedEncounterPool(t, 678, "standard", "female")
		choice, err := pool.Select(func(n int) (int, error) {
			if n == 4096 && !c.shiny {
				return 1, nil
			}
			return 0, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		record := trainer.Record{Choice: choice, FirstSpecies: c.firstSpecies, FirstVariant: c.firstVariant, RegularCollected: c.regular, XPAwarded: c.xp, Before: trainer.Progress{Level: 2}, After: trainer.Progress{Level: 2}, Completion: trainer.Completion{Species: 1, SpeciesTotal: 1017, Variants: 1, VariantsTotal: 2669}}
		output := string(formatEncounter(record, []byte("[SPRITE]\n"), "full"))
		if !strings.Contains(output, c.notice+"\n") || !strings.Contains(output, fmt.Sprintf("XP gained: %d\n", c.xp)) || !strings.Contains(output, "#678 Meowstic · Female") {
			t.Fatalf("case %v: %s", c, output)
		}
		if strings.Contains(output, "SHINY ENCOUNTER") != c.shiny || strings.Contains(output, "regular variant not yet collected") != (c.shiny && c.firstVariant && !c.regular) || strings.Contains(output, "LEVEL UP") {
			t.Fatal("incorrect shiny/level notice")
		}
		if c.firstSpecies && strings.Contains(output, "NEW VARIANT DISCOVERED") {
			t.Fatal("first species double notice")
		}
		if c.shiny && !strings.Contains(output, "Female · Shiny") {
			t.Fatal("missing appearance label")
		}
	}
}

func TestEncounterModesPersistIdenticalState(t *testing.T) {
	var expected []int64
	for _, mode := range []string{"full", "compact", "no-title", "achievements", "sprite"} {
		t.Run(mode, func(t *testing.T) {
			path := encounterPath(t)
			factory := fixedEncounterFactory(t, 6, "mega-x", "default", true)
			var out bytes.Buffer
			status, errout := invokeEncounter(t, path, []string{"--output", mode}, &out, factory)
			if status != 0 || errout != "" || out.Len() == 0 {
				t.Fatalf("%d %q", status, errout)
			}
			state := readEncounterState(t, path)
			if expected == nil {
				expected = state
			}
			if !reflect.DeepEqual(expected, state) || state[0] != 1 || state[1] != 170 || state[5] != 170 || state[4] != 3 {
				t.Fatalf("mode state %v", state)
			}
			// Suppressing notices never suppresses persisted achievements.
			if (mode == "sprite" || mode == "compact") && bytes.Contains(out.Bytes(), []byte("🏆")) {
				t.Fatal("hidden achievement printed")
			}
			if mode == "no-title" && (bytes.Contains(out.Bytes(), []byte("#006")) || bytes.Contains(out.Bytes(), []byte("Fire / Dragon"))) {
				t.Fatal("identity leaked")
			}
			out.Reset()
			status, errout = invokeEncounter(t, path, []string{"--output", "achievements"}, &out, factory)
			if status != 0 || errout != "" || bytes.Contains(out.Bytes(), []byte("🏆")) || bytes.Contains(out.Bytes(), []byte("SHINY ENCOUNTER")) {
				t.Fatal("repeat achievements mode")
			}
			state = readEncounterState(t, path)
			if state[0] != 2 || state[1] != 280 || state[4] != 3 {
				t.Fatalf("repeat state %v", state)
			}
		})
	}
}

type encounterFailWriter struct {
	err   error
	short bool
}

func (w encounterFailWriter) Write(p []byte) (int, error) {
	if w.short {
		return len(p) - 1, nil
	}
	return 0, w.err
}

func TestEncounterPostCommitOutputFailures(t *testing.T) {
	for _, c := range []struct {
		name   string
		writer io.Writer
		status int
	}{{"broken pipe", encounterFailWriter{err: syscall.EPIPE}, 0}, {"write failure", encounterFailWriter{err: errors.New("output unavailable")}, 1}, {"short write", encounterFailWriter{short: true}, 1}} {
		t.Run(c.name, func(t *testing.T) {
			path := encounterPath(t)
			status, errout := invokeEncounter(t, path, nil, c.writer, fixedEncounterFactory(t, 25, "standard", "default", false))
			if status != c.status || (c.status == 0 && errout != "") || (c.status == 1 && !strings.Contains(errout, "write output")) {
				t.Fatalf("%d %q", status, errout)
			}
			if got := readEncounterState(t, path); !reflect.DeepEqual(got, []int64{1, 70, 1, 1, 1, 70, 1, 1}) {
				t.Fatalf("output failure state %v", got)
			}
		})
	}
}

func TestEncounterBeforeCommitFailuresNoOutputOrWrites(t *testing.T) {
	for _, failure := range []string{"factory", "random", "prepare", "empty prepare", "storage"} {
		t.Run(failure, func(t *testing.T) {
			path := encounterPath(t)
			base := fixedEncounterFactory(t, 25, "standard", "default", false)
			if failure == "storage" {
				repo, err := storage.Open(context.Background(), path)
				if err != nil {
					t.Fatal(err)
				}
				err = repo.Write(context.Background(), func(tx *storage.Tx) error {
					_, err := tx.ExecContext(context.Background(), "CREATE TRIGGER fail_output_encounter BEFORE INSERT ON achievement_unlocks BEGIN SELECT RAISE(ABORT,'unlock failed'); END")
					return err
				})
				repo.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
			factory := func(repo encounterRepository) (trainer.EncounterService, error) {
				if failure == "factory" {
					return trainer.EncounterService{}, errors.New("factory failed")
				}
				service, err := base(repo)
				if failure == "random" {
					service.Index = func(int) (int, error) { return 0, errors.New("random failed") }
				}
				if failure == "prepare" {
					service.Prepare = func(trainer.Choice) ([]byte, error) { return nil, errors.New("prepare failed") }
				}
				if failure == "empty prepare" {
					service.Prepare = func(trainer.Choice) ([]byte, error) { return nil, nil }
				}
				return service, err
			}
			var out bytes.Buffer
			status, errout := invokeEncounter(t, path, nil, &out, factory)
			if status != 1 || out.Len() != 0 || errout == "" {
				t.Fatalf("failure streams %d %q %q", status, out.String(), errout)
			}
			if got := readEncounterState(t, path); !reflect.DeepEqual(got, make([]int64, 8)) {
				t.Fatalf("failed encounter wrote %v", got)
			}
		})
	}
}

func TestEncounterFirstRunInactiveAndCorruptState(t *testing.T) {
	factory := fixedEncounterFactory(t, 25, "standard", "default", false)
	path := filepath.Join(t.TempDir(), "absent", "trainers.sqlite3")
	var out bytes.Buffer
	status, errout := invokeEncounter(t, path, nil, &out, factory)
	if status != 1 || out.Len() != 0 || !strings.Contains(errout, "trainer create <name>") {
		t.Fatalf("first run %d %q", status, errout)
	}
	if _, err := os.Stat(filepath.Dir(path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("first run created state")
	}
	repo, err := storage.Initialize(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	repo.Close()
	status, errout = invokeEncounter(t, path, nil, &out, factory)
	if status != 1 || !strings.Contains(errout, "No trainer profiles found.") {
		t.Fatal("empty initialized guidance")
	}
	path = encounterPath(t)
	repo, err = storage.Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Write(context.Background(), func(tx *storage.Tx) error {
		_, err := tx.ExecContext(context.Background(), "UPDATE app_state SET active_trainer_id=NULL")
		return err
	})
	repo.Close()
	if err != nil {
		t.Fatal(err)
	}
	status, errout = invokeEncounter(t, path, nil, &out, factory)
	if status != 1 || out.Len() != 0 || !strings.Contains(errout, "Mayank") || !strings.Contains(errout, "trainer use <name>") {
		t.Fatalf("inactive guidance %d %q", status, errout)
	}
	if got := readEncounterState(t, path); !reflect.DeepEqual(got, make([]int64, 8)) {
		t.Fatal("inactive encounter wrote state")
	}
	corrupt := filepath.Join(t.TempDir(), "corrupt.sqlite3")
	original := []byte("corrupt trainer state")
	if err := os.WriteFile(corrupt, original, 0600); err != nil {
		t.Fatal(err)
	}
	status, _ = invokeEncounter(t, corrupt, nil, &out, factory)
	if status != 1 || out.Len() != 0 {
		t.Fatal("corrupt state accepted")
	}
	got, err := os.ReadFile(corrupt)
	if err != nil || !bytes.Equal(got, original) {
		t.Fatal("corrupt state changed")
	}
	var errbuf bytes.Buffer
	status = executeEncounter(nil, &out, &errbuf, func() (string, error) { return "", storage.ErrInvalidPath }, testEncounterOpen, factory)
	if status != 2 || out.Len() != 0 {
		t.Fatal("invalid path status")
	}
}
