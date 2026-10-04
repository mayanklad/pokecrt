package tui

import (
	"context"
	"time"

	"github.com/mayanklad/pokecrt/internal/storage"
	"github.com/mayanklad/pokecrt/internal/trainer"
)

type profileActions struct {
	create func(context.Context, string) (trainer.Profile, error)
	use    func(context.Context, string) (trainer.Profile, error)
}

func createProfile(ctx context.Context, raw string) (trainer.Profile, error) {
	name, err := trainer.ParseName(raw)
	if err != nil {
		return trainer.Profile{}, err
	}
	path, err := storage.ResolvePath()
	if err != nil {
		return trainer.Profile{}, err
	}
	if err = ctx.Err(); err != nil {
		return trainer.Profile{}, err
	}
	repo, err := storage.Initialize(ctx, path)
	if err != nil {
		return trainer.Profile{}, err
	}
	defer repo.Close()
	return repo.CreateProfile(ctx, name, time.Now().UTC().UnixMilli())
}
func useProfile(ctx context.Context, raw string) (trainer.Profile, error) {
	name, err := trainer.ParseName(raw)
	if err != nil {
		return trainer.Profile{}, err
	}
	path, err := storage.ResolvePath()
	if err != nil {
		return trainer.Profile{}, err
	}
	repo, err := storage.Open(ctx, path)
	if err != nil {
		return trainer.Profile{}, err
	}
	defer repo.Close()
	return repo.UseProfile(ctx, name)
}
