package trainer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"time"

	"github.com/mayanklad/pokecrt/internal/catalog"
	"github.com/mayanklad/pokecrt/internal/sprite"
)

// Snapshot records the metadata of the encountered form, independently of later
// catalog changes. Scalar fields keep selected values owned by the action.
type Snapshot struct {
	SpeciesName, FormName, Type1, Type2 string
	Regional, Transformation            bool
}

type Choice struct {
	key      catalog.VariantKey
	snapshot Snapshot
	valid    bool
}

func (c Choice) Key() catalog.VariantKey { return c.key }
func (c Choice) Snapshot() Snapshot      { return c.snapshot }
func (c Choice) Valid() bool             { return c.valid }

type genderChoice struct {
	regular Choice
	shiny   bool
}
type formChoices struct {
	id      string
	genders []genderChoice
}
type speciesChoices struct {
	id    int
	forms []formChoices
}

// Pool is derived from metadata plus exact regular artwork availability. Its
// hierarchy makes species, forms and genders separate uniform selections.
type Pool struct {
	species  []speciesChoices
	variants int
	forms    int
}

func BundledPool() (*Pool, error) {
	return NewPool(catalog.All(), func(key catalog.VariantKey) bool { _, ok := sprite.Lookup(key); return ok })
}

func NewPool(species []catalog.Species, available func(catalog.VariantKey) bool) (*Pool, error) {
	if available == nil {
		return nil, errors.New("encounter inventory is unavailable")
	}
	p := &Pool{}
	seenSpecies := map[int]bool{}
	for _, s := range species {
		if s.ID <= 0 || s.Name == "" || seenSpecies[s.ID] {
			return nil, fmt.Errorf("invalid or duplicate encounter species %d", s.ID)
		}
		seenSpecies[s.ID] = true
		group := speciesChoices{id: s.ID}
		seenForms := map[string]bool{}
		for _, f := range s.Forms {
			if f.ID == "" || f.Name == "" || seenForms[f.ID] || len(f.Types) < 1 || len(f.Types) > 2 {
				return nil, fmt.Errorf("invalid encounter form for species %d", s.ID)
			}
			seenForms[f.ID] = true
			form := formChoices{id: f.ID}
			genders := slices.Clone(f.Genders)
			if len(genders) == 0 {
				return nil, fmt.Errorf("encounter form %d/%s has no modeled gender", s.ID, f.ID)
			}
			sort.Strings(genders)
			for i, g := range genders {
				if (g != "default" && g != "male" && g != "female") || (i > 0 && genders[i-1] == g) {
					return nil, fmt.Errorf("invalid encounter gender for %d/%s", s.ID, f.ID)
				}
				key := catalog.VariantKey{SpeciesID: s.ID, FormID: f.ID, Gender: g, Palette: "regular"}
				if !available(key) {
					continue
				}
				snapshot := Snapshot{SpeciesName: s.Name, FormName: f.Name, Type1: f.Types[0], Regional: slices.Contains(f.Tags, "regional"), Transformation: slices.Contains(f.Tags, "mega") || slices.Contains(f.Tags, "gigantamax")}
				if len(f.Types) == 2 {
					snapshot.Type2 = f.Types[1]
				}
				shinyKey := key
				shinyKey.Palette = "shiny"
				shiny := available(shinyKey)
				form.genders = append(form.genders, genderChoice{regular: Choice{key: key, snapshot: snapshot, valid: true}, shiny: shiny})
				p.variants++
				if shiny {
					p.variants++
				}
			}
			if len(form.genders) > 0 {
				group.forms = append(group.forms, form)
				p.forms++
			}
		}
		sort.Slice(group.forms, func(i, j int) bool { return group.forms[i].id < group.forms[j].id })
		if len(group.forms) > 0 {
			p.species = append(p.species, group)
		}
	}
	sort.Slice(p.species, func(i, j int) bool { return p.species[i].id < p.species[j].id })
	return p, nil
}

func (p *Pool) SpeciesCount() int { return len(p.species) }
func (p *Pool) VariantCount() int { return p.variants }
func (p *Pool) FormCount() int    { return p.forms }

// Select uses one draw for each hierarchy level. A shiny roll is made only when
// the exact regular combination also has shiny artwork; zero is the 1/4096 hit.
func (p *Pool) Select(index func(int) (int, error)) (Choice, error) {
	if p == nil || len(p.species) == 0 {
		return Choice{}, errors.New("no eligible encounter artwork")
	}
	if index == nil {
		return Choice{}, errors.New("encounter random source is unavailable")
	}
	draw := func(n int) (int, error) {
		i, err := index(n)
		if err != nil {
			return 0, err
		}
		if i < 0 || i >= n {
			return 0, fmt.Errorf("encounter random index %d outside [0,%d)", i, n)
		}
		return i, nil
	}
	i, err := draw(len(p.species))
	if err != nil {
		return Choice{}, err
	}
	species := p.species[i]
	i, err = draw(len(species.forms))
	if err != nil {
		return Choice{}, err
	}
	form := species.forms[i]
	i, err = draw(len(form.genders))
	if err != nil {
		return Choice{}, err
	}
	gender := form.genders[i]
	choice := gender.regular
	if gender.shiny {
		roll, err := draw(4096)
		if err != nil {
			return Choice{}, err
		}
		if roll == 0 {
			choice.key.Palette = "shiny"
		}
	}
	return choice, nil
}

type Record struct {
	ID, TrainerID, EncounteredAtMS int64
	Choice                         Choice
	FirstSpecies, FirstVariant     bool
	RegularCollected               bool
	Completion                     Completion
	Encounters, Species, Variants  int64
	XPAwarded                      int64
	Before, After                  Progress
	unlocks                        []Unlock
}

func (r Record) NewUnlocks() []Unlock                { return slices.Clone(r.unlocks) }
func (r Record) WithUnlocks(unlocks []Unlock) Record { r.unlocks = slices.Clone(unlocks); return r }

type EncounterRepository interface {
	ActiveProfile(context.Context) (Profile, error)
	RecordEncounter(context.Context, int64, Choice, int64) (Record, error)
}

// EncounterService captures the active profile before selection/preparation.
// Preparation must decode/render in memory; storage is called only on success.
type EncounterService struct {
	Repository EncounterRepository
	Pool       *Pool
	Index      func(int) (int, error)
	Clock      func() time.Time
	Prepare    func(Choice) ([]byte, error)
}

type EncounterResult struct {
	Record  Record
	artwork []byte
}

// NewBundledEncounterService prepares inventory only for an explicit encounter
// action. Production draws use the shared crypto/rand-backed unbiased index.
func NewBundledEncounterService(repo EncounterRepository, prepare func(Choice) ([]byte, error)) (EncounterService, error) {
	pool, err := BundledPool()
	if err != nil {
		return EncounterService{}, err
	}
	return EncounterService{Repository: repo, Pool: pool, Index: catalog.CryptoIndex, Clock: time.Now, Prepare: prepare}, nil
}

func (r EncounterResult) Artwork() []byte { return bytes.Clone(r.artwork) }

func (s EncounterService) Encounter(ctx context.Context) (EncounterResult, error) {
	if err := ctx.Err(); err != nil {
		return EncounterResult{}, err
	}
	if s.Repository == nil || s.Clock == nil || s.Prepare == nil {
		return EncounterResult{}, errors.New("encounter service is incomplete")
	}
	profile, err := s.Repository.ActiveProfile(ctx)
	if err != nil {
		return EncounterResult{}, err
	}
	choice, err := s.Pool.Select(s.Index)
	if err != nil {
		return EncounterResult{}, err
	}
	artwork, err := s.Prepare(choice)
	if err != nil {
		return EncounterResult{}, fmt.Errorf("prepare encounter artwork: %w", err)
	}
	if len(artwork) == 0 {
		return EncounterResult{}, errors.New("prepare encounter artwork: no rendered bytes")
	}
	artwork = bytes.Clone(artwork)
	if err = ctx.Err(); err != nil {
		return EncounterResult{}, err
	}
	when := s.Clock().UTC().UnixMilli()
	record, err := s.Repository.RecordEncounter(ctx, profile.ID, choice, when)
	if err != nil {
		return EncounterResult{}, err
	}
	return EncounterResult{Record: record, artwork: artwork}, nil
}
