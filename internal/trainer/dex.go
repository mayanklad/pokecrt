package trainer

import (
	"fmt"
	"slices"
	"sort"

	"github.com/mayanklad/pokecrt/internal/catalog"
)

// Discovery records contain persisted facts; they are never presentation models.
type Discovery struct{ Count, FirstMS, LastMS int64 }
type VariantDiscovery struct {
	Key catalog.VariantKey
	Discovery
	FormName string
	Types    []string
}
type DexRecords struct {
	Species    map[int]Discovery
	Variants   []VariantDiscovery
	Encounters int64
}
type DexFilter struct {
	Selection    catalog.Selection
	Seen, Unseen bool
}

func (f DexFilter) Metadata() bool {
	q := f.Selection
	return len(q.Types)+len(q.TypesAny)+len(q.Colors)+len(q.Stages) > 0 || q.Legendary || q.Mythical || q.Baby
}

type DexRow struct {
	Number, Generation int
	Name               string
	Seen, Eligible     bool
}
type DexGeneration struct {
	Generation  int
	Seen, Total int64
}
type DexSummary struct {
	Completion                   Completion
	Encounters, ShinyCollections int64
	Generations                  []DexGeneration
}
type DexForm struct {
	UnknownGenders int
	Name           string
	Types          []string
	Count          int64
	Genders        []DexGender
}
type DexGender struct {
	Name           string
	Regular, Shiny bool
}
type DexNode struct {
	Number   int
	Name     string
	Children []int
}

// DexEntry contains only revealed facts. ArtworkKey exists only for collected artwork.
type DexEntry struct {
	Number                    int
	Name                      string
	Seen                      bool
	Generation, Stage         int
	Color                     string
	Baby, Legendary, Mythical bool
	Discovery                 Discovery
	Forms                     []DexForm
	UnknownForms              int
	Evolution                 []DexNode
	SelectedName              string
	SelectedTypes             []string
	Selected                  Discovery
	Notice                    string
	ArtworkKey                *catalog.VariantKey
}
type Dex struct {
	species   []catalog.Species
	records   DexRecords
	available func(catalog.VariantKey) bool
	eligible  map[int]bool
}

func NewDex(species []catalog.Species, records DexRecords, available func(catalog.VariantKey) bool) *Dex {
	species = slices.Clone(species)
	for i := range species {
		species[i].Aliases = slices.Clone(species[i].Aliases)
		species[i].EvolvesTo = slices.Clone(species[i].EvolvesTo)
		species[i].Forms = slices.Clone(species[i].Forms)
		for j := range species[i].Forms {
			f := &species[i].Forms[j]
			f.Types = slices.Clone(f.Types)
			f.Genders = slices.Clone(f.Genders)
			f.Tags = slices.Clone(f.Tags)
			f.SourceAliases = slices.Clone(f.SourceAliases)
		}
	}
	copied := DexRecords{Species: map[int]Discovery{}, Encounters: records.Encounters, Variants: slices.Clone(records.Variants)}
	for id, v := range records.Species {
		copied.Species[id] = v
	}
	for i := range copied.Variants {
		copied.Variants[i].Types = slices.Clone(copied.Variants[i].Types)
	}
	d := &Dex{species: species, records: copied, available: available, eligible: map[int]bool{}}
	for _, s := range species {
		for _, f := range s.Forms {
			for _, g := range f.Genders {
				if available(catalog.VariantKey{SpeciesID: s.ID, FormID: f.ID, Gender: g, Palette: "regular"}) {
					d.eligible[s.ID] = true
				}
			}
		}
	}
	return d
}
func (d *Dex) Summary() DexSummary {
	out := DexSummary{Encounters: d.records.Encounters}
	seen := map[int]bool{}
	collected := map[catalog.VariantKey]bool{}
	eligibleKeys := map[catalog.VariantKey]bool{}
	for id := range d.records.Species {
		seen[id] = true
	}
	for _, v := range d.records.Variants {
		if !collected[v.Key] && v.Key.Palette == "shiny" {
			out.ShinyCollections++
		}
		collected[v.Key] = true
	}
	for _, s := range d.species {
		if d.eligible[s.ID] {
			out.Completion.SpeciesTotal++
			if seen[s.ID] {
				out.Completion.Species++
			}
		}
		for _, f := range s.Forms {
			for _, g := range f.Genders {
				k := catalog.VariantKey{SpeciesID: s.ID, FormID: f.ID, Gender: g, Palette: "regular"}
				if !d.available(k) {
					continue
				}
				eligibleKeys[k] = true
				k.Palette = "shiny"
				if d.available(k) {
					eligibleKeys[k] = true
				}
			}
		}
	}
	for k := range eligibleKeys {
		out.Completion.VariantsTotal++
		if collected[k] {
			out.Completion.Variants++
		}
	}
	gens := map[int]*DexGeneration{}
	for _, s := range d.species {
		if !d.eligible[s.ID] {
			continue
		}
		if gens[s.Generation] == nil {
			gens[s.Generation] = &DexGeneration{Generation: s.Generation}
		}
		g := gens[s.Generation]
		g.Total++
		if seen[s.ID] {
			g.Seen++
		}
	}
	for _, g := range gens {
		out.Generations = append(out.Generations, *g)
	}
	sort.Slice(out.Generations, func(i, j int) bool { return out.Generations[i].Generation < out.Generations[j].Generation })
	return out
}
func (d *Dex) List(f DexFilter) []DexRow {
	out := []DexRow{}
	q := f.Selection
	for _, s := range d.species {
		_, seen := d.records.Species[s.ID]
		if f.Seen && !seen || f.Unseen && seen || len(q.Generations) > 0 && !slices.Contains(q.Generations, s.Generation) {
			continue
		}
		if f.Metadata() {
			if !seen || len(q.Colors) > 0 && !slices.Contains(q.Colors, s.Color) || len(q.Stages) > 0 && !slices.Contains(q.Stages, s.Stage) || q.Legendary && !s.Legendary || q.Mythical && !s.Mythical || q.Baby && !s.Baby {
				continue
			}
			if len(q.Types)+len(q.TypesAny) > 0 {
				match := false
				for _, v := range d.records.Variants {
					if v.Key.SpeciesID != s.ID {
						continue
					}
					all := true
					for _, t := range q.Types {
						if !slices.Contains(v.Types, t) {
							all = false
						}
					}
					any := len(q.TypesAny) == 0
					for _, t := range q.TypesAny {
						if slices.Contains(v.Types, t) {
							any = true
						}
					}
					if all && any {
						match = true
					}
				}
				if !match {
					continue
				}
			}
		}
		name := "?????"
		if seen {
			name = s.Name
		}
		out = append(out, DexRow{Number: s.ID, Generation: s.Generation, Name: name, Seen: seen, Eligible: d.eligible[s.ID]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out
}
func (d *Dex) Entry(number int, q catalog.Selection) (DexEntry, error) {
	out := DexEntry{Number: number}
	var s catalog.Species
	for _, candidate := range d.species {
		if candidate.ID == number {
			s = candidate
			break
		}
	}
	if s.ID == 0 {
		return out, fmt.Errorf("unknown National number %d", number)
	}
	discovery, seen := d.records.Species[number]
	if !seen {
		out.Notice = "This Pokémon has not been discovered."
		return out, nil
	}
	out.Seen = true
	out.Name = s.Name
	out.Generation = s.Generation
	out.Color = s.Color
	out.Stage = s.Stage
	out.Baby = s.Baby
	out.Legendary = s.Legendary
	out.Mythical = s.Mythical
	out.Discovery = discovery
	known := map[string][]VariantDiscovery{}
	for _, v := range d.records.Variants {
		if v.Key.SpeciesID == number {
			known[v.Key.FormID] = append(known[v.Key.FormID], v)
		}
	}
	ids := []string{}
	for id := range known {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		vs := known[id]
		f := DexForm{Name: vs[0].FormName, Types: slices.Clone(vs[0].Types)}
		genders := map[string]*DexGender{}
		for _, v := range vs {
			f.Count += v.Count
			g := genders[v.Key.Gender]
			if g == nil {
				g = &DexGender{Name: v.Key.Gender}
				genders[v.Key.Gender] = g
			}
			if v.Key.Palette == "shiny" {
				g.Shiny = true
			} else {
				g.Regular = true
			}
		}
		gs := []string{}
		for g := range genders {
			gs = append(gs, g)
		}
		sort.Strings(gs)
		for _, g := range gs {
			f.Genders = append(f.Genders, *genders[g])
		}
		for _, catalogForm := range s.Forms {
			if catalogForm.ID == id {
				for _, gender := range catalogForm.Genders {
					if genders[gender] == nil && d.available(catalog.VariantKey{SpeciesID: number, FormID: id, Gender: gender, Palette: "regular"}) {
						f.UnknownGenders++
					}
				}
			}
		}
		out.Forms = append(out.Forms, f)
	}
	for _, f := range s.Forms {
		if len(known[f.ID]) > 0 {
			continue
		}
		supported := false
		for _, g := range f.Genders {
			if d.available(catalog.VariantKey{SpeciesID: number, FormID: f.ID, Gender: g, Palette: "regular"}) {
				supported = true
			}
		}
		if supported {
			out.UnknownForms++
		}
	}
	out.Evolution = d.evolution(number)
	formID := q.Form
	if formID == "" {
		formID = "standard"
	}
	gender := q.Gender
	palette := "regular"
	if q.Shiny {
		palette = "shiny"
	}
	var form catalog.Form
	for _, f := range s.Forms {
		if f.ID == formID {
			form = f
			break
		}
	}
	if gender == "" {
		gender = form.DefaultGender
		if gender == "" {
			for _, v := range known[formID] {
				gender = v.Key.Gender
				break
			}
		}
	}
	key := catalog.VariantKey{SpeciesID: number, FormID: formID, Gender: gender, Palette: palette}
	for _, v := range known[formID] {
		if v.Key == key {
			out.Selected = v.Discovery
			out.SelectedName = v.FormName
			if gender != "default" {
				out.SelectedName += " · " + gender
			}
			out.SelectedName += " · " + palette
			out.SelectedTypes = slices.Clone(v.Types)
			if d.available(key) {
				out.ArtworkKey = &key
			} else {
				out.Notice = "Collected artwork is unavailable in this dataset."
			}
			return out, nil
		}
	}
	if form.ID == "" || q.Gender != "" && (len(form.Genders) < 2 || !slices.Contains(form.Genders, gender)) || !d.available(key) {
		return DexEntry{}, fmt.Errorf("unsupported appearance selection")
	}
	if len(known[formID]) == 0 {
		out.Notice = "LOCKED FORM - this form has not been discovered."
	} else {
		out.Notice = "LOCKED VARIANT - this exact appearance has not been collected; no fallback artwork is shown."
	}
	return out, nil
}
func (d *Dex) evolution(number int) []DexNode {
	family := map[int]bool{number: true}
	changed := true
	for changed {
		changed = false
		for _, s := range d.species {
			for _, child := range s.EvolvesTo {
				if family[s.ID] || family[child] {
					if !family[s.ID] || !family[child] {
						changed = true
					}
					family[s.ID] = true
					family[child] = true
				}
			}
			if s.EvolvesFrom > 0 && (family[s.ID] || family[s.EvolvesFrom]) {
				if !family[s.ID] || !family[s.EvolvesFrom] {
					changed = true
				}
				family[s.ID] = true
				family[s.EvolvesFrom] = true
			}
		}
	}
	out := []DexNode{}
	for _, s := range d.species {
		if !family[s.ID] {
			continue
		}
		name := "?????"
		if _, ok := d.records.Species[s.ID]; ok {
			name = s.Name
		}
		out = append(out, DexNode{Number: s.ID, Name: name, Children: slices.Clone(s.EvolvesTo)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out
}
