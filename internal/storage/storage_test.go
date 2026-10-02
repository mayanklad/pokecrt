package storage

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

func TestResolvePath(t *testing.T) {
	for _, tc := range []struct {
		name, override, xdg, home, want string
		fail                            bool
	}{
		{"override", "/private/state", "/xdg", "/home/user", "/private/state/trainers.sqlite3", false},
		{"relative override", "state", "/xdg", "/home/user", "", true},
		{"xdg", "", "/xdg", "/home/user", "/xdg/pokecrt/trainers.sqlite3", false},
		{"relative xdg", "", "relative", "/home/user", "/home/user/.local/share/pokecrt/trainers.sqlite3", false},
		{"home", "", "", "/home/user", "/home/user/.local/share/pokecrt/trainers.sqlite3", false},
		{"bad home", "", "", "relative", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolvePath(func(key string) string {
				if key == "POKECRT_DATA_DIR" {
					return tc.override
				}
				return tc.xdg
			}, func() (string, error) { return tc.home, nil })
			if (err != nil) != tc.fail || got != tc.want {
				t.Fatalf("got %q, %v; want %q, failure %v", got, err, tc.want, tc.fail)
			}
		})
	}
	_, err := resolvePath(func(string) string { return "" }, func() (string, error) { return "", errors.New("home unavailable") })
	if err == nil {
		t.Fatal("missing home error")
	}
}

func fresh(t *testing.T) (*Repository, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state", databaseName)
	r, err := Initialize(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = r.Close() })
	return r, path
}

func integer(t *testing.T, r *Repository, query string) int {
	t.Helper()
	var n int
	if err := r.QueryRowContext(context.Background(), query).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestMissingStateDoesNotCreatePaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent", databaseName)
	for _, open := range []func(context.Context, string) (*Repository, error){Open, ReadOnly} {
		_, err := open(context.Background(), path)
		if !errors.Is(err, ErrNoState) {
			t.Fatalf("got %v", err)
		}
		if _, err := os.Stat(filepath.Dir(path)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("directory was created: %v", err)
		}
	}
	if _, err := Initialize(context.Background(), "relative.sqlite3"); err == nil {
		t.Fatal("relative path accepted")
	}
}

func TestPathEscapingAndCanceledInitialization(t *testing.T) {
	path := filepath.Join(t.TempDir(), "space # ? % 雪", databaseName)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Initialize(ctx, path); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled initialization: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("canceled initialization created state")
	}
	r, err := Initialize(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if integer(t, r, "PRAGMA user_version") != 1 {
		t.Fatal("escaped path did not initialize")
	}
}

func TestInitialSchemaAndConnectionPolicy(t *testing.T) {
	r, path := fresh(t)
	for query, want := range map[string]int{"PRAGMA user_version": 1, "PRAGMA foreign_keys": 1, "PRAGMA busy_timeout": 5000, "PRAGMA synchronous": 2, "SELECT count(*) FROM app_state WHERE id = 1 AND active_trainer_id IS NULL": 1, "SELECT count(*) FROM trainers": 0} {
		if got := integer(t, r, query); got != want {
			t.Fatalf("%s: %d != %d", query, got, want)
		}
	}
	for _, file := range []struct {
		path string
		mode os.FileMode
	}{{filepath.Dir(path), 0700}, {path, 0600}} {
		info, err := os.Stat(file.path)
		if err != nil || info.Mode().Perm() != file.mode {
			t.Fatalf("permissions %s: %v, %v", file.path, info, err)
		}
	}
	var mode string
	if err := r.QueryRowContext(context.Background(), "PRAGMA journal_mode").Scan(&mode); err != nil || mode != "delete" {
		t.Fatalf("journal mode %q: %v", mode, err)
	}
	// Force a replacement connection: URI pragmas must apply to each one.
	r.db.SetMaxIdleConns(0)
	if integer(t, r, "PRAGMA foreign_keys") != 1 || integer(t, r, "PRAGMA busy_timeout") != 5000 {
		t.Fatal("replacement connection not configured")
	}
	ro, err := ReadOnly(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	err = ro.Write(context.Background(), func(tx *Tx) error {
		_, err := tx.ExecContext(context.Background(), "INSERT INTO trainers VALUES (1, 'A', 'a', 1)")
		return err
	})
	if err == nil {
		t.Fatal("read-only connection accepted write")
	}
	if backups, _ := filepath.Glob(path + ".backup-*"); len(backups) != 0 {
		t.Fatal("fresh init created backup")
	}
}

func insertTrainer(ctx context.Context, tx *Tx, id int) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO trainers VALUES (?, ?, ?, ?)", id, "Trainer", id, 1)
	return err
}

func TestWriteRollbackCancellationPanicAndConstraints(t *testing.T) {
	r, _ := fresh(t)
	ctx := context.Background()
	injected := errors.New("injected failure")
	err := r.Write(ctx, func(tx *Tx) error {
		if err := insertTrainer(ctx, tx, 1); err != nil {
			return err
		}
		return injected
	})
	if !errors.Is(err, injected) || integer(t, r, "SELECT count(*) FROM trainers") != 0 {
		t.Fatalf("rollback failed: %v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	err = r.Write(canceled, func(tx *Tx) error {
		if err := insertTrainer(canceled, tx, 1); err != nil {
			return err
		}
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) || integer(t, r, "SELECT count(*) FROM trainers") != 0 {
		t.Fatalf("cancellation committed: %v", err)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("panic swallowed")
			}
		}()
		_ = r.Write(ctx, func(tx *Tx) error {
			if err := insertTrainer(ctx, tx, 1); err != nil {
				t.Fatal(err)
			}
			panic("failure")
		})
	}()
	if integer(t, r, "SELECT count(*) FROM trainers") != 0 {
		t.Fatal("panic committed")
	}
	if err := r.Write(ctx, func(tx *Tx) error {
		if err := insertTrainer(ctx, tx, 1); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO trainer_progress VALUES (1, 0)")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		"INSERT INTO trainer_progress VALUES (2, 0)",
		"UPDATE trainer_progress SET xp_total = -1",
		"DELETE FROM trainers WHERE id = 1",
		"INSERT INTO trainers VALUES (2, 'Other', '1', 2)",
		"INSERT INTO variant_discoveries VALUES (1, 25, 'standard', 'default', 'regular', 1, 1, 1)",
	} {
		if err := r.Write(ctx, func(tx *Tx) error { _, err := tx.ExecContext(ctx, query); return err }); err == nil {
			t.Fatalf("constraint accepted: %s", query)
		}
	}
}

func TestRejectedFilesPreserved(t *testing.T) {
	for _, tc := range []string{"newer", "corrupt", "unversioned"} {
		t.Run(tc, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), databaseName)
			if tc == "corrupt" {
				if err := os.WriteFile(path, []byte("not a database"), 0600); err != nil {
					t.Fatal(err)
				}
			} else {
				db, err := connect(path, "rwc")
				if err != nil {
					t.Fatal(err)
				}
				query := "CREATE TABLE legacy (id INTEGER)"
				if tc == "newer" {
					query += "; PRAGMA user_version = 99"
				}
				if _, err = db.Exec(query); err != nil {
					t.Fatal(err)
				}
				_ = db.Close()
			}
			before, _ := os.ReadFile(path)
			_, err := Open(context.Background(), path)
			if err == nil {
				t.Fatal("rejected database opened")
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(before, after) {
				t.Fatal("rejected database changed")
			}
		})
	}
	_, path := fresh(t)
	link := filepath.Join(filepath.Dir(path), "link.sqlite3")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(context.Background(), link); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestUpgradeBackupAndFailedMigration(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "rollback"}[fail], func(t *testing.T) {
			r, path := fresh(t)
			ctx := context.Background()
			if err := r.Write(ctx, func(tx *Tx) error { return insertTrainer(ctx, tx, 1) }); err != nil {
				t.Fatal(err)
			}
			script := "CREATE TABLE upgrade_marker (id INTEGER);"
			if fail {
				script += " INSERT INTO nonexistent VALUES (1);"
			}
			files := fstest.MapFS{"migrations/002_test.sql": {Data: []byte(script)}}
			err := r.migrate(ctx, files, 2)
			if (err != nil) != fail {
				t.Fatalf("migration error: %v", err)
			}
			backups, _ := filepath.Glob(path + ".backup-v1-to-v2-*")
			if len(backups) != 1 {
				t.Fatalf("backups: %v", backups)
			}
			if fail && !strings.Contains(err.Error(), backups[0]) {
				t.Fatalf("missing backup guidance: %v", err)
			}
			backup, err := ReadOnly(ctx, backups[0])
			if err != nil {
				t.Fatal(err)
			}
			if integer(t, backup, "SELECT count(*) FROM trainers") != 1 || integer(t, backup, "PRAGMA user_version") != 1 {
				t.Fatal("backup lost original data")
			}
			_ = backup.Close()
			info, _ := os.Stat(backups[0])
			if info.Mode().Perm() != 0600 {
				t.Fatal("backup permissions")
			}
			if fail {
				if integer(t, r, "PRAGMA user_version") != 1 || integer(t, r, "SELECT count(*) FROM sqlite_schema WHERE name = 'upgrade_marker'") != 0 {
					t.Fatal("migration was not atomic")
				}
				before, _ := os.ReadFile(backups[0])
				if err := r.migrate(ctx, fstest.MapFS{"migrations/002_test.sql": {Data: []byte("CREATE TABLE upgrade_marker (id INTEGER);")}}, 2); err != nil {
					t.Fatal(err)
				}
				after, _ := os.ReadFile(backups[0])
				if !bytes.Equal(before, after) {
					t.Fatal("existing backup overwritten")
				}
				all, _ := filepath.Glob(path + ".backup-v1-to-v2-*")
				if len(all) != 2 {
					t.Fatal("retry did not preserve both backups")
				}
			} else if integer(t, r, "PRAGMA user_version") != 2 {
				t.Fatal("schema version not upgraded")
			}
		})
	}
}

func TestConcurrentWritesAndMigrationRecheck(t *testing.T) {
	r, path := fresh(t)
	ctx := context.Background()
	other, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if err := r.Write(ctx, func(tx *Tx) error {
		if err := insertTrainer(ctx, tx, 1); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, "INSERT INTO trainer_progress VALUES (1, 0)")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, repo := range []*Repository{r, other} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				if err := repo.Write(ctx, func(tx *Tx) error {
					var xp int
					if err := tx.QueryRowContext(ctx, "SELECT xp_total FROM trainer_progress WHERE trainer_id = 1").Scan(&xp); err != nil {
						return err
					}
					_, err := tx.ExecContext(ctx, "UPDATE trainer_progress SET xp_total = ? WHERE trainer_id = 1", xp+1)
					return err
				}); err != nil {
					errs <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if integer(t, r, "SELECT xp_total FROM trainer_progress") != 40 {
		t.Fatal("lost concurrent updates")
	}
	files := fstest.MapFS{"migrations/002_test.sql": {Data: []byte("CREATE TABLE upgrade_once (id INTEGER);")}}
	errs = make(chan error, 2)
	for _, repo := range []*Repository{r, other} {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- repo.migrate(ctx, files, 2) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	backups, _ := filepath.Glob(path + ".backup-v1-to-v2-*")
	if len(backups) != 1 {
		t.Fatalf("migration ran more than once: %v", backups)
	}
}
