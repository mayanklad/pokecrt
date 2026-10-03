package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

// BenchmarkTrainerHistory uses a real first encounter and repeat-only history.
// Bulk fixture rows retain legitimate snapshots and aggregate/XP invariants.
// Encounter measurements append repeats, so measured histories start at N rows.
func BenchmarkTrainerHistory(b *testing.B) {
	for _, n := range []int{1000, 10000, 100000} {
		for _, operation := range []string{"Encounter", "TrainerView", "DexView"} {
			b.Run(fmt.Sprintf("%s/Rows%d", operation, n), func(b *testing.B) {
				ctx := context.Background()
				r, err := Initialize(ctx, filepath.Join(b.TempDir(), "trainers.sqlite3"))
				if err != nil {
					b.Fatal(err)
				}
				defer r.Close()
				name, _ := trainer.ParseName("Benchmark")
				p, err := r.CreateProfile(ctx, name, 0)
				if err != nil {
					b.Fatal(err)
				}
				s, _ := catalog.ByNumber(1)
				s.Forms = s.Forms[:1]
				pool, err := trainer.NewPool([]catalog.Species{s}, func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
				if err != nil {
					b.Fatal(err)
				}
				choice, err := pool.Select(func(n int) (int, error) {
					if n == 4096 {
						return 1, nil
					}
					return 0, nil
				})
				if err != nil {
					b.Fatal(err)
				}
				if _, err = r.RecordEncounter(ctx, p.ID, choice, 1); err != nil {
					b.Fatal(err)
				}
				err = r.Write(ctx, func(tx *Tx) error {
					_, err := tx.ExecContext(ctx, `WITH RECURSIVE numbers(n) AS (SELECT 2 UNION ALL SELECT n+1 FROM numbers WHERE n<?)
 INSERT INTO encounters(trainer_id,species_id,form_id,gender_key,palette,encountered_at_ms,dataset_id,species_name_snapshot,form_name_snapshot,type1_snapshot,type2_snapshot,regional_snapshot,transformation_snapshot,first_species,first_variant,xp_awarded)
 SELECT e.trainer_id,e.species_id,e.form_id,e.gender_key,e.palette,n,e.dataset_id,e.species_name_snapshot,e.form_name_snapshot,e.type1_snapshot,e.type2_snapshot,e.regional_snapshot,e.transformation_snapshot,0,0,10 FROM encounters e,numbers WHERE e.id=1`, n)
					if err != nil {
						return err
					}
					for _, table := range []string{"species_discoveries", "variant_discoveries"} {
						if _, err = tx.ExecContext(ctx, "UPDATE "+table+" SET encounter_count=?,last_seen_ms=? WHERE trainer_id=?", n, n, p.ID); err != nil {
							return err
						}
					}
					_, err = tx.ExecContext(ctx, "UPDATE trainer_progress SET xp_total=? WHERE trainer_id=?", int64(n)*10+60, p.ID)
					return err
				})
				if err != nil {
					b.Fatal(err)
				}
				// Bring unlocks up to the fixture's legitimate cumulative history before timing.
				if _, err = r.RecordEncounter(ctx, p.ID, choice, int64(n+1)); err != nil {
					b.Fatal(err)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					switch operation {
					case "Encounter":
						_, err = r.RecordEncounter(ctx, p.ID, choice, int64(n+2+i))
					case "TrainerView":
						_, err = r.TrainerRecords(ctx, p.ID)
					case "DexView":
						_, err = r.DexRecords(ctx, p.ID)
					}
					if err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}
