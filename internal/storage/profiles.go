package storage

import (
	"context"
	"database/sql"
	"errors"

	"github.com/mayanklad/pokecrt/internal/trainer"
)

var _ trainer.Profiles = (*Repository)(nil)

func (r *Repository) CreateProfile(ctx context.Context, name trainer.Name, createdAtMS int64) (profile trainer.Profile, err error) {
	if name.Key() == "" {
		return profile, trainer.ErrInvalidName
	}
	err = r.Write(ctx, func(tx *Tx) error {
		var exists bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM trainers WHERE normalized_name = ?)", name.Key()).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return trainer.ErrDuplicateName
		}
		var first bool
		if err := tx.QueryRowContext(ctx, "SELECT NOT EXISTS(SELECT 1 FROM trainers)").Scan(&first); err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, "INSERT INTO trainers (display_name, normalized_name, created_at_ms) VALUES (?, ?, ?)", name.Display(), name.Key(), createdAtMS)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO trainer_progress (trainer_id, xp_total) VALUES (?, 0)", id); err != nil {
			return err
		}
		if first {
			if _, err = tx.ExecContext(ctx, "UPDATE app_state SET active_trainer_id = ? WHERE id = 1", id); err != nil {
				return err
			}
		}
		profile = trainer.Profile{ID: id, Name: name.Display(), CreatedAtMS: createdAtMS, Active: first}
		return nil
	})
	if err != nil {
		return trainer.Profile{}, err
	}
	return profile, nil
}

func (r *Repository) UseProfile(ctx context.Context, name trainer.Name) (profile trainer.Profile, err error) {
	if name.Key() == "" {
		return profile, trainer.ErrInvalidName
	}
	err = r.Write(ctx, func(tx *Tx) error {
		err := tx.QueryRowContext(ctx, "SELECT id, display_name, created_at_ms FROM trainers WHERE normalized_name = ?", name.Key()).Scan(&profile.ID, &profile.Name, &profile.CreatedAtMS)
		if errors.Is(err, sql.ErrNoRows) {
			return trainer.ErrNotFound
		}
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE app_state SET active_trainer_id = ? WHERE id = 1", profile.ID); err != nil {
			return err
		}
		profile.Active = true
		return nil
	})
	if err != nil {
		return trainer.Profile{}, err
	}
	return profile, nil
}

const profileColumns = `t.id, t.display_name, t.created_at_ms,
    COALESCE(t.id = (SELECT active_trainer_id FROM app_state WHERE id = 1), 0)`

func (r *Repository) ListProfiles(ctx context.Context) ([]trainer.Profile, error) {
	rows, err := r.QueryContext(ctx, "SELECT "+profileColumns+" FROM trainers t ORDER BY t.normalized_name, t.id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	profiles := []trainer.Profile{}
	for rows.Next() {
		var p trainer.Profile
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAtMS, &p.Active); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (r *Repository) FindProfile(ctx context.Context, name trainer.Name) (trainer.Profile, error) {
	if name.Key() == "" {
		return trainer.Profile{}, trainer.ErrInvalidName
	}
	var p trainer.Profile
	err := r.QueryRowContext(ctx, "SELECT "+profileColumns+" FROM trainers t WHERE t.normalized_name = ?", name.Key()).Scan(&p.ID, &p.Name, &p.CreatedAtMS, &p.Active)
	if errors.Is(err, sql.ErrNoRows) {
		return trainer.Profile{}, trainer.ErrNotFound
	}
	return p, err
}

func (r *Repository) ActiveProfile(ctx context.Context) (trainer.Profile, error) {
	var p trainer.Profile
	err := r.QueryRowContext(ctx, "SELECT "+profileColumns+" FROM trainers t WHERE t.id = (SELECT active_trainer_id FROM app_state WHERE id = 1)").Scan(&p.ID, &p.Name, &p.CreatedAtMS, &p.Active)
	if errors.Is(err, sql.ErrNoRows) {
		return trainer.Profile{}, trainer.ErrNoActive
	}
	return p, err
}
