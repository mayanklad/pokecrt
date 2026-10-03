package trainer

import (
	"sort"

	"github.com/mayanklad/pokecrt/internal/catalog"
)

// TrainerRecords is one consistent persisted read for a single trainer.
// Raw evidence is converted into disclosure-safe views before presentation.
type TrainerRecords struct {
	Dex                               DexRecords
	XP, ShinyEncounters               int64
	FirstEncounterMS, LastEncounterMS *int64
	AchievementState                  AchievementState
	Unlocks                           []Unlock
}

type GenerationStatistics struct {
	Generation                          int
	Discovered, Eligible, EligibleTotal int64
}

type Statistics struct {
	UnclassifiedSpecies                                              int64
	Progress                                                         Progress
	Encounters, Species, Variants, ShinyEncounters, ShinyCollections int64
	FirstEncounterMS, LastEncounterMS                                *int64
	Completion                                                       Completion
	Generations                                                      []GenerationStatistics
}

// TrainerStatistics retains historical counts separately from current eligibility.
func TrainerStatistics(records TrainerRecords, species []catalog.Species, available func(catalog.VariantKey) bool) (Statistics, error) {
	progress, err := XPProgress(records.XP)
	if err != nil {
		return Statistics{}, err
	}
	dex := NewDex(species, records.Dex, available).Summary()
	out := Statistics{Progress: progress, Encounters: records.Dex.Encounters, Species: int64(len(records.Dex.Species)), Variants: int64(len(records.Dex.Variants)), ShinyEncounters: records.ShinyEncounters, ShinyCollections: dex.ShinyCollections, Completion: dex.Completion}
	out.UnclassifiedSpecies = out.Species
	generations := map[int]*GenerationStatistics{}
	for _, species := range species {
		if generations[species.Generation] == nil {
			generations[species.Generation] = &GenerationStatistics{Generation: species.Generation}
		}
		if _, seen := records.Dex.Species[species.ID]; seen {
			generations[species.Generation].Discovered++
			out.UnclassifiedSpecies--
		}
	}
	for _, g := range dex.Generations {
		generations[g.Generation].Eligible = g.Seen
		generations[g.Generation].EligibleTotal = g.Total
	}
	for _, g := range generations {
		out.Generations = append(out.Generations, *g)
	}
	sort.Slice(out.Generations, func(i, j int) bool { return out.Generations[i].Generation < out.Generations[j].Generation })
	if records.FirstEncounterMS != nil {
		value := *records.FirstEncounterMS
		out.FirstEncounterMS = &value
	}
	if records.LastEncounterMS != nil {
		value := *records.LastEncounterMS
		out.LastEncounterMS = &value
	}
	return out, nil
}

type AchievementView struct {
	ID, Name, Description string
	Current, Target       int64
	HasTarget             bool
	EarnedAtMS            int64
	TargetAtUnlock        int64
	HadTargetAtUnlock     bool
}
type AchievementViews struct{ Unlocked, Locked []AchievementView }

// Views consumes the same ordered registry as encounter evaluation. Ready goals
// without a persisted unlock stay locked; reading never awards an achievement.
func (t *AchievementTargets) Views(state AchievementState, unlocks []Unlock) AchievementViews {
	earned := map[string]Unlock{}
	for _, u := range unlocks {
		earned[u.ID] = u
	}
	goals := map[string]Goal{}
	for _, g := range t.Goals(state) {
		goals[g.ID] = g
	}
	out := AchievementViews{}
	for _, definition := range t.Definitions() {
		if u, ok := earned[definition.ID]; ok {
			out.Unlocked = append(out.Unlocked, AchievementView{ID: definition.ID, Name: definition.Name, Description: definition.Description, EarnedAtMS: u.UnlockedAtMS, TargetAtUnlock: u.Target, HadTargetAtUnlock: u.HasTarget})
			delete(earned, definition.ID)
		} else if g, ok := goals[definition.ID]; ok {
			out.Locked = append(out.Locked, AchievementView{ID: g.ID, Name: g.Name, Description: g.Description, Current: g.Current, Target: g.Target, HasTarget: g.HasTarget})
		}
	}
	// Retain unlocks from definitions absent in a later registry, without inventing
	// a name or leaking catalog identities from an old identifier.
	ids := []string{}
	for id := range earned {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		u := earned[id]
		out.Unlocked = append(out.Unlocked, AchievementView{ID: id, Name: "Retained achievement", Description: "Earned in an earlier version.", EarnedAtMS: u.UnlockedAtMS, TargetAtUnlock: u.Target, HadTargetAtUnlock: u.HasTarget})
	}
	return out
}
