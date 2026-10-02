package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func checkVersion(version int) error {
	return supportedVersion(version, SchemaVersion)
}

func supportedVersion(version, target int) error {
	if version > target {
		return fmt.Errorf("trainer schema %d is newer than supported schema %d; upgrade PokéCRT", version, target)
	}
	if version < 0 {
		return fmt.Errorf("invalid trainer schema version %d", version)
	}
	return nil
}

func (r *Repository) migrate(ctx context.Context, files fs.FS, target int) error {
	var version int
	if err := r.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if err := supportedVersion(version, target); err != nil {
		return err
	}
	var integrity string
	if err := r.db.QueryRowContext(ctx, "PRAGMA quick_check").Scan(&integrity); err != nil {
		return err
	}
	if integrity != "ok" {
		return fmt.Errorf("trainer database is corrupt: %s; preserve it for recovery", integrity)
	}
	var mode string
	if err := r.db.QueryRowContext(ctx, "PRAGMA journal_mode = DELETE").Scan(&mode); err != nil {
		return err
	}
	if mode != "delete" {
		return fmt.Errorf("cannot enable DELETE journal mode: %s", mode)
	}
	var backup string
	err := r.Write(ctx, func(tx *Tx) error {
		// Another process may have migrated while this process waited for its
		// write lock. Always recheck before backup or executing migration SQL.
		if err := tx.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
			return err
		}
		if err := supportedVersion(version, target); err != nil {
			return err
		}
		if version == target {
			return nil
		}
		var tables int
		if err := tx.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_schema WHERE type = 'table' AND name NOT LIKE 'sqlite_%'").Scan(&tables); err != nil {
			return err
		}
		if version == 0 && tables != 0 {
			return errors.New("unversioned nonempty trainer database; preserve it and review its schema before migration")
		}
		if version != 0 || tables != 0 {
			var err error
			backup, err = r.backup(ctx, version, target)
			if err != nil {
				return err
			}
		}
		for next := version + 1; next <= target; next++ {
			name := fmt.Sprintf("migrations/%03d_", next)
			matches, err := fs.Glob(files, name+"*.sql")
			if err != nil {
				return err
			}
			if len(matches) != 1 {
				return fmt.Errorf("expected one migration for schema %d, found %d", next, len(matches))
			}
			script, err := fs.ReadFile(files, matches[0])
			if err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, string(script)); err != nil {
				return fmt.Errorf("migration %03d: %w", next, err)
			}
		}
		_, err := tx.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", target))
		return err
	})
	if err != nil && backup != "" {
		return fmt.Errorf("trainer migration failed; preserved backup: %s: %w", backup, err)
	}
	return err
}

// backup runs on a separate read-only connection while the caller holds the
// migration write lock. VACUUM INTO reads committed data without writing the
// source; it cannot run inside the migration connection's transaction.
func (r *Repository) backup(ctx context.Context, from, to int) (string, error) {
	f, err := os.CreateTemp(filepath.Dir(r.path), fmt.Sprintf("%s.backup-v%d-to-v%d-*", filepath.Base(r.path), from, to))
	if err != nil {
		return "", err
	}
	path := f.Name()
	if err = errors.Join(f.Chmod(0600), f.Close()); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.Remove(path)
		}
	}()
	db, err := connect(r.path, "ro")
	if err != nil {
		return "", err
	}
	defer db.Close()
	if _, err = db.ExecContext(ctx, "VACUUM INTO ?", path); err != nil {
		return "", fmt.Errorf("backup trainer database: %w", err)
	}
	f, err = os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return "", err
	}
	if err = errors.Join(f.Sync(), f.Close()); err != nil {
		return "", err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	if err = errors.Join(dir.Sync(), dir.Close()); err != nil {
		return "", err
	}
	complete = true
	return path, nil
}
