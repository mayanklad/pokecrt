package tui

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var errSettingsVersion = errors.New("unsupported settings version; file left unchanged")

func appearancePath() (string, error) {
	if dir := os.Getenv("POKECRT_CONFIG_DIR"); dir != "" {
		if !filepath.IsAbs(dir) {
			return "", errors.New("POKECRT_CONFIG_DIR must be an absolute path")
		}
		return filepath.Join(dir, "appearance.conf"), nil
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(dir) {
		return filepath.Join(dir, "pokecrt", "appearance.conf"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(home) {
		return "", errors.New("cannot resolve an absolute configuration directory")
	}
	return filepath.Join(home, ".config", "pokecrt", "appearance.conf"), nil
}
func loadAppearance(ctx context.Context) (Appearance, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path, err := appearancePath()
	if err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("settings must be a regular file")
	}
	if info.Size() > 256 {
		return "", errors.New("settings file exceeds 256 bytes")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	data, err := io.ReadAll(io.LimitReader(f, 257))
	err = errors.Join(err, f.Close())
	if err != nil {
		return "", err
	}
	if len(data) > 256 {
		return "", errors.New("settings file exceeds 256 bytes")
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "version=") || !strings.HasPrefix(lines[1], "appearance=") {
		return "", errors.New("invalid appearance settings format")
	}
	if lines[0] != "version=1" {
		return "", errSettingsVersion
	}
	return ParseAppearance(strings.TrimPrefix(lines[1], "appearance="))
}
func saveAppearance(ctx context.Context, a Appearance) error {
	if _, err := ParseAppearance(string(a)); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := appearancePath()
	if err != nil {
		return err
	}
	if _, e := loadAppearance(ctx); e != nil {
		return e
	}
	if info, e := os.Lstat(path); e == nil && !info.Mode().IsRegular() {
		return errors.New("settings must be a regular file")
	} else if e != nil && !errors.Is(e, os.ErrNotExist) {
		return e
	}
	data := []byte("version=1\nappearance=" + string(a) + "\n")
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".settings-*")
	if err != nil {
		return err
	}
	temp := f.Name()
	defer os.Remove(temp)
	if err = f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	err = errors.Join(err, f.Close())
	if err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if err = os.Rename(temp, path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}
