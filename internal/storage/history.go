package storage

import (
	"context"
	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

// HistoryEntry retains recorded identities even if catalog artwork is removed.
type HistoryEntry struct {
	ID, EncounteredAtMS, XP    int64
	Key                        catalog.VariantKey
	Snapshot                   trainer.Snapshot
	FirstSpecies, FirstVariant bool
}

// RecentEncounters returns at most the latest 50 encounters for one trainer.
// The indexed timestamp order has an ID tie-breaker for stable navigation.
func (r *Repository) RecentEncounters(ctx context.Context, trainerID int64) ([]HistoryEntry, error) {
	rows, err := r.QueryContext(ctx, `SELECT id, encountered_at_ms, xp_awarded, species_id,
 form_id, gender_key, palette, species_name_snapshot, form_name_snapshot,
 type1_snapshot, COALESCE(type2_snapshot,''), regional_snapshot, transformation_snapshot,
 first_species, first_variant FROM encounters WHERE trainer_id=?
 ORDER BY encountered_at_ms DESC,id DESC LIMIT 50`, trainerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []HistoryEntry{}
	for rows.Next() {
		var h HistoryEntry
		if err = rows.Scan(&h.ID, &h.EncounteredAtMS, &h.XP, &h.Key.SpeciesID, &h.Key.FormID, &h.Key.Gender, &h.Key.Palette, &h.Snapshot.SpeciesName, &h.Snapshot.FormName, &h.Snapshot.Type1, &h.Snapshot.Type2, &h.Snapshot.Regional, &h.Snapshot.Transformation, &h.FirstSpecies, &h.FirstVariant); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}
