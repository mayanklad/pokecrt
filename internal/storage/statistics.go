package storage

import (
	"context"
	"database/sql"
	"errors"

	"github.com/mayanklad/pokecrt/internal/trainer"
)

// TrainerRecords reads counts, discovery snapshots, goal evidence and unlocks in
// one read transaction. It never evaluates awards or initializes missing state.
func (r *Repository) TrainerRecords(ctx context.Context, id int64) (trainer.TrainerRecords, error) {
	out := trainer.TrainerRecords{}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return out, err
	}
	defer tx.Rollback()
	if err = tx.QueryRowContext(ctx, "SELECT xp_total FROM trainer_progress WHERE trainer_id=?", id).Scan(&out.XP); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = trainer.ErrNotFound
		}
		return out, err
	}
	out.Dex, err = dexRecords(ctx, tx, id)
	if err != nil {
		return out, err
	}
	var first, last sql.NullInt64
	if err = tx.QueryRowContext(ctx, `SELECT COUNT(CASE WHEN palette='shiny' THEN 1 END),MIN(encountered_at_ms),MAX(encountered_at_ms) FROM encounters WHERE trainer_id=?`, id).Scan(&out.ShinyEncounters, &first, &last); err != nil {
		return out, err
	}
	if first.Valid {
		value := first.Int64
		out.FirstEncounterMS = &value
	}
	if last.Valid {
		value := last.Int64
		out.LastEncounterMS = &value
	}
	record := trainer.Record{Encounters: out.Dex.Encounters, Species: int64(len(out.Dex.Species)), Variants: int64(len(out.Dex.Variants))}
	out.AchievementState, err = encounterAchievementState(ctx, tx, id, record)
	if err != nil {
		return out, err
	}
	for _, v := range out.Dex.Variants {
		out.AchievementState.SeenVariants = append(out.AchievementState.SeenVariants, v.Key)
	}
	rows, err := tx.QueryContext(ctx, "SELECT achievement_id,unlocked_at_ms,dataset_id,target_at_unlock FROM achievement_unlocks WHERE trainer_id=? ORDER BY achievement_id", id)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var u trainer.Unlock
		var target sql.NullInt64
		if err = rows.Scan(&u.ID, &u.UnlockedAtMS, &u.DatasetID, &target); err != nil {
			rows.Close()
			return out, err
		}
		u.HasTarget = target.Valid
		u.Target = target.Int64
		out.Unlocks = append(out.Unlocks, u)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	err = tx.Commit()
	return out, err
}
