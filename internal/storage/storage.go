// Package storage provides local SQLite state. Opening is explicit: importing
// this package never resolves paths or creates trainer data.
package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const SchemaVersion = 1

var ErrNoState = errors.New("trainer storage does not exist; create a trainer first")

//go:embed migrations/*.sql
var migrations embed.FS

type Repository struct {
	db   *sql.DB
	path string
}

// Open opens existing writable storage and applies supported migrations.
// It never creates a missing database or directory.
func Open(ctx context.Context, path string) (*Repository, error) {
	return open(ctx, path, false, false)
}

// Initialize is reserved for explicit first-profile setup. It creates protected
// storage if missing, but never replaces an existing database.
func Initialize(ctx context.Context, path string) (*Repository, error) {
	return open(ctx, path, true, false)
}

// ReadOnly opens current-schema storage without initialization or upgrades.
func ReadOnly(ctx context.Context, path string) (*Repository, error) {
	return open(ctx, path, false, true)
}

func open(ctx context.Context, path string, initialize, readOnly bool) (*Repository, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("trainer database path must be absolute")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if !initialize {
			return nil, ErrNoState
		}
		dir := filepath.Dir(path)
		if err = os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
		if err = os.Chmod(dir, 0700); err != nil {
			return nil, err
		}
		f, createErr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if createErr != nil && !errors.Is(createErr, os.ErrExist) {
			return nil, createErr
		}
		if f != nil {
			err = errors.Join(f.Chmod(0600), f.Close())
			if err != nil {
				return nil, err
			}
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("trainer database must be a regular file, not a symlink or directory")
	}
	mode := "rw"
	if readOnly {
		mode = "ro"
	}
	db, err := connect(path, mode)
	if err != nil {
		return nil, err
	}
	r := &Repository{db: db, path: path}
	if readOnly {
		var version int
		err = db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version)
		if err == nil {
			err = checkVersion(version)
		}
		if err == nil && version != SchemaVersion {
			err = errors.New("trainer storage needs migration; open it for writing first")
		}
	} else {
		err = r.migrate(ctx, migrations, SchemaVersion)
	}
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open trainer storage: %w", err)
	}
	if !readOnly {
		if err = os.Chmod(filepath.Dir(path), 0700); err != nil {
			_ = db.Close()
			return nil, err
		}
		if err = os.Chmod(path, 0600); err != nil {
			_ = db.Close()
			return nil, err
		}
	}
	return r, nil
}

func connect(path, mode string) (*sql.DB, error) {
	u := url.URL{Scheme: "file", Path: path}
	q := url.Values{"mode": {mode}, "_pragma": {"busy_timeout(5000)", "foreign_keys(1)", "synchronous(FULL)"}}
	// Configure writable journal mode only after checking schema compatibility.
	// Read-only opens and rejected newer databases must not change it.
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, nil
}

func (r *Repository) Close() error { return r.db.Close() }

func (r *Repository) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, query, args...)
}

func (r *Repository) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return r.db.QueryRowContext(ctx, query, args...)
}

// Tx is valid only inside a Write callback. Use its connection for all reads and
// writes in the operation; do not call repository methods from the callback.
type Tx struct{ conn *sql.Conn }

func (t *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return t.conn.ExecContext(ctx, query, args...)
}
func (t *Tx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return t.conn.QueryContext(ctx, query, args...)
}
func (t *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return t.conn.QueryRowContext(ctx, query, args...)
}

// Write serializes the operation with BEGIN IMMEDIATE on one dedicated
// connection. It rolls back failures, cancellation and panics. Commit errors
// are returned without retrying an operation whose outcome may be uncertain.
func (r *Repository) Write(ctx context.Context, fn func(*Tx) error) (err error) {
	conn, err := r.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		// Cancellation can leave the outcome of BEGIN uncertain. Discard this
		// connection rather than risk pooling an unfinished transaction.
		_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		return err
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, rollbackErr := conn.ExecContext(cleanup, "ROLLBACK")
		if rollbackErr != nil {
			// A failed rollback must never return an open transaction to the pool.
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
			err = errors.Join(err, fmt.Errorf("rollback trainer transaction: %w", rollbackErr))
		}
	}()
	if err = fn(&Tx{conn: conn}); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if _, err = conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit trainer transaction failed; outcome may be uncertain (do not retry automatically): %w", err)
	}
	committed = true
	return nil
}
