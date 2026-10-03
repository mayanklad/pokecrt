package storage

import (
	"context"
	"database/sql"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

// readQueries is shared by an encounter write transaction and a read transaction.
type readQueries interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// DexRecords reads one trainer's discoveries and recorded form facts in a single
// read transaction. It performs no writes, migration, reward evaluation or lookup.
func (r *Repository) DexRecords(ctx context.Context, trainerID int64) (trainer.DexRecords, error) {
	out := trainer.DexRecords{Species: map[int]trainer.Discovery{}}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	out, err = dexRecords(ctx, tx, trainerID)
	if err != nil {
		return out, err
	}
	err = tx.Commit()
	return out, err
}

func dexRecords(ctx context.Context, tx readQueries, trainerID int64) (trainer.DexRecords, error) {
	out := trainer.DexRecords{Species: map[int]trainer.Discovery{}}
	var err error
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM encounters WHERE trainer_id=?", trainerID).Scan(&out.Encounters); err != nil {
		return out, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT species_id,encounter_count,first_seen_ms,last_seen_ms FROM species_discoveries WHERE trainer_id=?", trainerID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var id int
		var v trainer.Discovery
		if err = rows.Scan(&id, &v.Count, &v.FirstMS, &v.LastMS); err != nil {
			rows.Close()
			return out, err
		}
		out.Species[id] = v
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT v.species_id,v.form_id,v.gender_key,v.palette,v.encounter_count,v.first_seen_ms,v.last_seen_ms,e.form_name_snapshot,e.type1_snapshot,COALESCE(e.type2_snapshot,'')
 FROM variant_discoveries v JOIN encounters e ON e.id=(SELECT id FROM encounters WHERE trainer_id=v.trainer_id AND species_id=v.species_id AND form_id=v.form_id AND gender_key=v.gender_key AND palette=v.palette ORDER BY encountered_at_ms DESC,id DESC LIMIT 1)
 WHERE v.trainer_id=? ORDER BY v.species_id,v.form_id,v.gender_key,v.palette`, trainerID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var v trainer.VariantDiscovery
		var t1, t2 string
		if err = rows.Scan(&v.Key.SpeciesID, &v.Key.FormID, &v.Key.Gender, &v.Key.Palette, &v.Count, &v.FirstMS, &v.LastMS, &v.FormName, &t1, &t2); err != nil {
			rows.Close()
			return out, err
		}
		v.Types = []string{t1}
		if t2 != "" {
			v.Types = append(v.Types, t2)
		}
		out.Variants = append(out.Variants, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	return out, nil
}
