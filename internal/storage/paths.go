package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const databaseName = "trainers.sqlite3"

var ErrInvalidPath = errors.New("invalid trainer data path")

// ResolvePath returns the trainer database path without touching the filesystem.
// Public commands must not call it; only stateful commands need this path.
func ResolvePath() (string, error) {
	return resolvePath(os.Getenv, os.UserHomeDir)
}

func resolvePath(getenv func(string) string, home func() (string, error)) (string, error) {
	if dir := getenv("POKECRT_DATA_DIR"); dir != "" {
		if !filepath.IsAbs(dir) {
			return "", fmt.Errorf("%w: POKECRT_DATA_DIR must be an absolute path", ErrInvalidPath)
		}
		return filepath.Join(dir, databaseName), nil
	}
	if dir := getenv("XDG_DATA_HOME"); filepath.IsAbs(dir) {
		return filepath.Join(dir, "pokecrt", databaseName), nil
	}
	dir, err := home()
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(dir) {
		return "", errors.New("cannot resolve an absolute user home directory")
	}
	return filepath.Join(dir, ".local", "share", "pokecrt", databaseName), nil
}
