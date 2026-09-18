package main

import (
	"sort"
	"strings"
)

// Rarity groups species by how hard they are to come by in the seed shop.
type Rarity int

const (
	Common Rarity = iota
	Uncommon
	Rare
	Legendary
)

func (r Rarity) String() string {
	switch r {
	case Common:
		return "common"
	case Uncommon:
		return "uncommon"
	case Rare:
		return "rare"
	default:
		return "legendary"
	}
}

// Kind is the loose gardening category a species belongs to. It drives the
// almanac's grouping and the seed shop's filters.
type Kind int

const (
	KindFlower Kind = iota
	KindBulb
	KindVine
	KindHerb
	KindEdible
	KindTree
	KindSucculent
	KindFern
	KindGrass
	KindCarnivore
	KindAquatic
)

func (k Kind) String() string {
	switch k {
	case KindFlower:
		return "flower"
	case KindBulb:
		return "bulb"
	case KindVine:
		return "vine"
	case KindHerb:
		return "herb"
	case KindEdible:
		return "edible"
	case KindTree:
		return "tree & shrub"
	case KindSucculent:
		return "succulent"
	case KindFern:
		return "fern & moss"
	case KindGrass:
		return "grass"
	case KindCarnivore:
		return "carnivorous"
	default:
		return "aquatic"
	}
}

// Stages are the five drawn phases of every plant's life.
const StageCount = 5

const (
	StageSeed = iota
	StageSprout
	StageSeedling
	StageBud
	StageMature
)

var stageNames = [StageCount]string{"seed", "sprout", "seedling", "budding", "mature"}

// Palette colours a species' artwork. Values are 256-colour terminal codes.
type Palette struct {
	Stem   string
	Leaf   string
	Bloom  string
	Accent string
}

// Species is one real plant: its botany, its care notes and its five frames
// of artwork. Art frames are bottom-aligned when drawn.
type Species struct {
	ID     string
	Common string
	Latin  string
	Family string
	Kind   Kind
	Rarity Rarity
	Origin string
	Bloom  string
	Sun    string
	Water  string
	Height string
	Note   string // one-line gardener's tip
	Desc   string // real description, shown on the info card

	Seasons  []Season // seasons this species grows happiest in
	SeedCost int
	Unlock   int     // lifetime matured plants needed before the shop stocks it
	Hours    float64 // hours of well-tended growth from seed to mature

	Palette Palette
	Art     [StageCount][]string
	// Glyphs optionally overrides how individual runes in Art are coloured.
	Glyphs map[rune]glyphClass
}

// Stage returns the art frame for a stage index, clamped into range.
func (s *Species) Stage(i int) []string {
	if i < 0 {
		i = 0
	}
	if i >= StageCount {
		i = StageCount - 1
	}
	return s.Art[i]
}

// SeasonNames renders the growing seasons as readable text.
func (s *Species) SeasonNames() string {
	if len(s.Seasons) == 0 || len(s.Seasons) == 4 {
		return "all year"
	}
	parts := make([]string, 0, len(s.Seasons))
	for _, sea := range s.Seasons {
		parts = append(parts, sea.String())
	}
	return strings.Join(parts, ", ")
}

// LikesSeason reports whether the species is in its element right now.
func (s *Species) LikesSeason(season Season) bool {
	if len(s.Seasons) == 0 {
		return true
	}
	for _, sea := range s.Seasons {
		if sea == season {
			return true
		}
	}
	return false
}

var (
	speciesList []*Species
	speciesByID map[string]*Species
)

// register adds species to the catalogue. Each species file calls it from an
// init function so the catalogue assembles itself.
func register(list ...*Species) {
	speciesList = append(speciesList, list...)
}

func buildIndex() {
	sort.Slice(speciesList, func(i, j int) bool {
		if speciesList[i].Kind != speciesList[j].Kind {
			return speciesList[i].Kind < speciesList[j].Kind
		}
		return speciesList[i].Common < speciesList[j].Common
	})
	speciesByID = make(map[string]*Species, len(speciesList))
	for _, sp := range speciesList {
		speciesByID[sp.ID] = sp
	}
}

// AllSpecies returns the catalogue in display order.
func AllSpecies() []*Species {
	if speciesByID == nil {
		buildIndex()
	}
	return speciesList
}

// SpeciesByID looks a species up, returning nil when it is unknown.
func SpeciesByID(id string) *Species {
	if speciesByID == nil {
		buildIndex()
	}
	return speciesByID[id]
}
