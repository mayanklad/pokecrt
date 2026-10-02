package storage

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

var _ trainer.EncounterRepository = (*Repository)(nil)

// RecordEncounter commits discovery, XP and achievement unlocks atomically.
func (r *Repository) RecordEncounter(ctx context.Context, trainerID int64, choice trainer.Choice, whenMS int64) (record trainer.Record, err error) {
	if trainerID <= 0 {
		return record, trainer.ErrNotFound
	}
	if err := validateEncounterChoice(choice); err != nil {
		return record, err
	}
	r.goalsOnce.Do(func() {
		r.goals, r.goalsErr = trainer.NewAchievementTargets(catalog.All(), func(k catalog.VariantKey) bool { _, ok := sprite.Lookup(k); return ok })
	})
	if r.goalsErr != nil {
		return record, r.goalsErr
	}
	key, snapshot := choice.Key(), choice.Snapshot()
	err = r.Write(ctx, func(tx *Tx) error {
		var exists bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM trainers WHERE id = ?)", trainerID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return trainer.ErrNotFound
		}
		var speciesExists, variantExists bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM species_discoveries WHERE trainer_id = ? AND species_id = ?)", trainerID, key.SpeciesID).Scan(&speciesExists); err != nil {
			return err
		}
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM variant_discoveries WHERE trainer_id = ? AND species_id = ? AND form_id = ? AND gender_key = ? AND palette = ?)", trainerID, key.SpeciesID, key.FormID, key.Gender, key.Palette).Scan(&variantExists); err != nil {
			return err
		}
		var total int64
		if err := tx.QueryRowContext(ctx, "SELECT xp_total FROM trainer_progress WHERE trainer_id = ?", trainerID).Scan(&total); err != nil {
			return err
		}
		before, err := trainer.XPProgress(total)
		if err != nil {
			return err
		}
		award := trainer.AwardXP(!speciesExists, !variantExists, key.Palette == "shiny")
		after, err := trainer.AdvanceXP(total, award)
		if err != nil {
			return err
		}
		var type2 any
		if snapshot.Type2 != "" {
			type2 = snapshot.Type2
		}
		result, err := tx.ExecContext(ctx, `INSERT INTO encounters (
   trainer_id, species_id, form_id, gender_key, palette, encountered_at_ms,
   dataset_id, species_name_snapshot, form_name_snapshot, type1_snapshot,
   type2_snapshot, regional_snapshot, transformation_snapshot, first_species,
   first_variant, xp_awarded
  ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			trainerID, key.SpeciesID, key.FormID, key.Gender, key.Palette, whenMS,
			catalog.DatasetID, snapshot.SpeciesName, snapshot.FormName, snapshot.Type1,
			type2, snapshot.Regional, snapshot.Transformation, !speciesExists, !variantExists, award)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO species_discoveries
   (trainer_id,species_id,first_seen_ms,last_seen_ms,encounter_count)
   VALUES (?, ?, ?, ?, 1)
   ON CONFLICT(trainer_id,species_id) DO UPDATE SET
   first_seen_ms = MIN(first_seen_ms,excluded.first_seen_ms),
   last_seen_ms = MAX(last_seen_ms,excluded.last_seen_ms),
   encounter_count = encounter_count + 1`, trainerID, key.SpeciesID, whenMS, whenMS); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO variant_discoveries
   (trainer_id,species_id,form_id,gender_key,palette,first_seen_ms,last_seen_ms,encounter_count)
   VALUES (?, ?, ?, ?, ?, ?, ?, 1)
   ON CONFLICT(trainer_id,species_id,form_id,gender_key,palette) DO UPDATE SET
   first_seen_ms = MIN(first_seen_ms,excluded.first_seen_ms),
   last_seen_ms = MAX(last_seen_ms,excluded.last_seen_ms),
   encounter_count = encounter_count + 1`, trainerID, key.SpeciesID, key.FormID, key.Gender, key.Palette, whenMS, whenMS); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "UPDATE trainer_progress SET xp_total = ? WHERE trainer_id = ?", after.Total, trainerID); err != nil {
			return err
		}
		record = trainer.Record{XPAwarded: award, Before: before, After: after, ID: id, TrainerID: trainerID, EncounteredAtMS: whenMS, Choice: choice, FirstSpecies: !speciesExists, FirstVariant: !variantExists}
		// One snapshot of committed-result counts, still inside the write lock.
		if err := tx.QueryRowContext(ctx, `SELECT
   (SELECT count(*) FROM encounters WHERE trainer_id = ?),
   (SELECT count(*) FROM species_discoveries WHERE trainer_id = ?),
   (SELECT count(*) FROM variant_discoveries WHERE trainer_id = ?)`, trainerID, trainerID, trainerID).Scan(&record.Encounters, &record.Species, &record.Variants); err != nil {
			return err
		}
		state, err := encounterAchievementState(ctx, tx, trainerID, record)
		if err != nil {
			return err
		}
		var unlocks []trainer.Unlock
		for _, goal := range r.goals.Goals(state) {
			if !goal.Ready() {
				continue
			}
			var target any
			if goal.HasTarget {
				target = goal.Target
			}
			result, err := tx.ExecContext(ctx, `INSERT INTO achievement_unlocks (trainer_id,achievement_id,unlocked_at_ms,dataset_id,target_at_unlock) VALUES (?,?,?,?,?) ON CONFLICT(trainer_id,achievement_id) DO NOTHING`, trainerID, goal.ID, whenMS, catalog.DatasetID, target)
			if err != nil {
				return err
			}
			n, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if n == 1 {
				unlocks = append(unlocks, trainer.Unlock{ID: goal.ID, Name: goal.Name, Description: goal.Description, DatasetID: catalog.DatasetID, UnlockedAtMS: whenMS, Target: goal.Target, HasTarget: goal.HasTarget})
			}
		}
		record = record.WithUnlocks(unlocks)
		return nil
	})
	if err != nil {
		return trainer.Record{}, err
	}
	return record, nil
}

func validateEncounterChoice(choice trainer.Choice) error {
	if !choice.Valid() {
		return errors.New("invalid encounter choice")
	}
	key := choice.Key()
	regular := key
	regular.Palette = "regular"
	if _, ok := sprite.Lookup(regular); !ok {
		return errors.New("encounter choice lacks exact regular artwork")
	}
	if _, ok := sprite.Lookup(key); !ok {
		return errors.New("encounter choice artwork is unavailable")
	}
	species, ok := catalog.ByNumber(key.SpeciesID)
	if !ok {
		return errors.New("encounter species is unavailable")
	}
	for _, form := range species.Forms {
		if form.ID != key.FormID {
			continue
		}
		if !slices.Contains(form.Genders, key.Gender) {
			break
		}
		if len(form.Types) < 1 || len(form.Types) > 2 {
			break
		}
		snapshot := trainer.Snapshot{SpeciesName: species.Name, FormName: form.Name, Type1: form.Types[0], Regional: slices.Contains(form.Tags, "regional"), Transformation: slices.Contains(form.Tags, "mega") || slices.Contains(form.Tags, "gigantamax")}
		if len(form.Types) == 2 {
			snapshot.Type2 = form.Types[1]
		}
		if snapshot != choice.Snapshot() {
			return errors.New("encounter metadata does not match current inventory")
		}
		return nil
	}
	return fmt.Errorf("encounter form/gender is unavailable for species #%03d", key.SpeciesID)
}

func encounterAchievementState(ctx context.Context, tx *Tx, id int64, record trainer.Record) (trainer.AchievementState, error) {
	state := trainer.AchievementState{Encounters: record.Encounters, Species: record.Species, Variants: record.Variants, SeenSpecies: map[int]bool{}, EncounteredTypes: map[string]bool{}}
	rows, err := tx.QueryContext(ctx, "SELECT species_id FROM species_discoveries WHERE trainer_id = ?", id)
	if err != nil {
		return state, err
	}
	for rows.Next() {
		var species int
		if err := rows.Scan(&species); err != nil {
			rows.Close()
			return state, err
		}
		state.SeenSpecies[species] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return state, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT type1_snapshot FROM encounters WHERE trainer_id = ? UNION SELECT type2_snapshot FROM encounters WHERE trainer_id = ? AND type2_snapshot IS NOT NULL`, id, id)
	if err != nil {
		return state, err
	}
	for rows.Next() {
		var typ string
		if err := rows.Scan(&typ); err != nil {
			rows.Close()
			return state, err
		}
		state.EncounteredTypes[typ] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return state, err
	}
	err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM encounters WHERE trainer_id=? AND palette='shiny'), EXISTS(SELECT 1 FROM encounters WHERE trainer_id=? AND regional_snapshot=1), EXISTS(SELECT 1 FROM encounters WHERE trainer_id=? AND transformation_snapshot=1)`, id, id, id).Scan(&state.Shiny, &state.Regional, &state.Transformation)
	return state, err
}
