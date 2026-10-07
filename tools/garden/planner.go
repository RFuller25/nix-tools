package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// The planner. Plan mode lays ghosts of the seed in your shed over the beds so
// you can see how a layout would get on before sowing anything; a template is
// a layout saved to be stamped down again, anywhere in the garden.

// ghost is a planned planting.
type ghost struct {
	Species string
	Variety int
	Packet  int64 // the shed packet it will be sown from, or 0 to buy from the shop
	Genome  Genome
}

func (gh ghost) species() *Species { return SpeciesByID(gh.Species) }

// TemplateCell is one planting in a template, relative to its top-left bed.
type TemplateCell struct {
	DX, DY  int
	Species string
	Variety int
	Pond    bool
	// Want, when set, picks the seed from the shed that is highest (or lowest)
	// in a trait, so a template can ask for "the tallest foxglove at the back".
	Want *TraitWant `json:"want,omitempty"`
}

// TraitWant asks for the high or the low end of a trait.
type TraitWant struct {
	Trait Trait `json:"trait"`
	High  bool  `json:"high"`
}

// Template is a saved layout.
type Template struct {
	ID      string         `json:"id"`
	Name    string         `json:"name"`
	W       int            `json:"w"`
	H       int            `json:"h"`
	Cells   []TemplateCell `json:"cells"`
	Note    string         `json:"note,omitempty"`
	Builtin bool           `json:"-"`
}

// builtinTemplates are layouts gardeners have really used.
func builtinTemplates() []Template {
	return []Template{
		{
			ID: "three-sisters", Name: "Three Sisters", Builtin: true, W: 3, H: 1,
			Note: "Corn for the beans to climb, beans to feed the ground, and squash to shade it: the Haudenosaunee garden, in a row.",
			Cells: []TemplateCell{
				{DX: 0, DY: 0, Species: "sweetcorn"}, {DX: 1, DY: 0, Species: "pea"}, {DX: 2, DY: 0, Species: "pumpkin"},
			},
		},
		{
			ID: "tomato-guild", Name: "Tomato guild", Builtin: true, W: 3, H: 3,
			Note: "A tomato with basil beside it and marigolds around to guard it from the nematodes.",
			Cells: []TemplateCell{
				{DX: 1, DY: 0, Species: "marigold"},
				{DX: 0, DY: 1, Species: "basil"}, {DX: 1, DY: 1, Species: "tomato"}, {DX: 2, DY: 1, Species: "basil"},
				{DX: 1, DY: 2, Species: "marigold"},
			},
		},
		{
			ID: "cottage-border", Name: "Cottage border", Builtin: true, W: 5, H: 2,
			Note: "Tall things at the back, scented herbs and self-seeders in front, as a cottage border is planted.",
			Cells: []TemplateCell{
				{DX: 0, DY: 0, Species: "delphinium", Want: &TraitWant{TraitHeight, true}}, {DX: 1, DY: 0, Species: "foxglove", Want: &TraitWant{TraitHeight, true}},
				{DX: 2, DY: 0, Species: "rose"}, {DX: 3, DY: 0, Species: "foxglove", Want: &TraitWant{TraitHeight, true}},
				{DX: 4, DY: 0, Species: "delphinium", Want: &TraitWant{TraitHeight, true}},
				{DX: 0, DY: 1, Species: "lavender"}, {DX: 1, DY: 1, Species: "cosmos"}, {DX: 2, DY: 1, Species: "chives"},
				{DX: 3, DY: 1, Species: "cosmos"}, {DX: 4, DY: 1, Species: "lavender"},
			},
		},
		{
			ID: "rose-and-allium", Name: "Roses with alliums", Builtin: true, W: 3, H: 1,
			Note: "An onion relative at a rose's feet keeps the aphids off it.",
			Cells: []TemplateCell{
				{DX: 0, DY: 0, Species: "chives"}, {DX: 1, DY: 0, Species: "rose"}, {DX: 2, DY: 0, Species: "allium"},
			},
		},
		{
			ID: "pond-edge", Name: "Pond edge", Builtin: true, W: 3, H: 1,
			Note: "Water lilies and a lotus in a row of ponds; dig the ponds first (d).",
			Cells: []TemplateCell{
				{DX: 0, DY: 0, Species: "waterlily", Pond: true}, {DX: 1, DY: 0, Species: "lotus", Pond: true}, {DX: 2, DY: 0, Species: "waterlily", Pond: true},
			},
		},
	}
}

// AllTemplates is the built-in layouts followed by the ones you have saved.
func (g *Garden) AllTemplates() []Template {
	out := builtinTemplates()
	return append(out, g.Templates...)
}

// rect is a block of beds, by top-left and bottom-right corner.
type rect struct{ c0, r0, c1, r1 int }

// rectOf is the block of beds between two corners.
func rectOf(a, b int) rect {
	ac, ar, bc, br := a%plotCols, a/plotCols, b%plotCols, b/plotCols
	return rect{min(ac, bc), min(ar, br), max(ac, bc), max(ar, br)}
}

func (r rect) contains(idx int) bool {
	c, row := idx%plotCols, idx/plotCols
	return c >= r.c0 && c <= r.c1 && row >= r.r0 && row <= r.r1
}

// SaveTemplate records the plantings in a block of beds as a template.
func (g *Garden) SaveTemplate(name string, r rect, now time.Time) (Template, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Template{}, fmt.Errorf("a layout needs a name")
	}
	t := Template{Name: name, W: r.c1 - r.c0 + 1, H: r.r1 - r.r0 + 1}
	for row := r.r0; row <= r.r1; row++ {
		for c := r.c0; c <= r.c1; c++ {
			idx := row*plotCols + c
			if idx >= len(g.Plots) || g.Plots[idx].Empty() {
				continue
			}
			p := &g.Plots[idx]
			t.Cells = append(t.Cells, TemplateCell{DX: c - r.c0, DY: row - r.r0, Species: p.SpeciesID, Variety: p.Variety, Pond: p.Pond})
		}
	}
	if len(t.Cells) == 0 {
		return Template{}, fmt.Errorf("there is nothing planted in those beds to save")
	}
	g.TemplateSeq++
	t.ID = fmt.Sprintf("saved-%d", g.TemplateSeq)
	g.Templates = append(g.Templates, t)
	g.Log(now, "Saved the layout ‘%s’ (%d beds).", name, len(t.Cells))
	return t, nil
}

// DeleteTemplate removes a saved layout by ID; built-in ones cannot go.
func (g *Garden) DeleteTemplate(id string) bool {
	for i, t := range g.Templates {
		if t.ID == id {
			g.Templates = append(g.Templates[:i], g.Templates[i+1:]...)
			return true
		}
	}
	return false
}

// seedSource picks where a planting of a species will come from: the shed
// packet that best fits what the cell wants, or the shop. used tracks seeds
// already promised in this plan.
func (g *Garden) seedSource(sp *Species, variety int, want *TraitWant, used map[int64]int) ghost {
	best, bestScore := -1, -1e9
	for i, pk := range g.Shed {
		if pk.SpeciesID != sp.ID || pk.Count-used[pk.ID] < 1 {
			continue
		}
		mean := pk.Mean()
		score := 0.0
		switch {
		case want != nil:
			v := float64(mean.value(want.Trait))
			if !want.High {
				v = -v
			}
			score = v
		default:
			score = -float64(mean.Distance(sp.VarietyGenome(variety))) * 100
		}
		if score > bestScore {
			best, bestScore = i, score
		}
	}
	if best >= 0 {
		pk := g.Shed[best]
		used[pk.ID]++
		return ghost{Species: sp.ID, Variety: pk.Variety, Packet: pk.ID, Genome: pk.Mean()}
	}
	return ghost{Species: sp.ID, Variety: variety, Genome: sp.VarietyGenome(variety)}
}

// StampPlan lays a template over the garden with its top-left at anchor, as
// ghosts, and says what it had to leave out.
func (g *Garden) StampPlan(t Template, anchor int) (map[int]ghost, []string) {
	ghosts := map[int]ghost{}
	var skipped []string
	used := map[int64]int{}
	ac, ar := anchor%plotCols, anchor/plotCols
	for _, cell := range t.Cells {
		sp := SpeciesByID(cell.Species)
		if sp == nil {
			continue
		}
		col, row := ac+cell.DX, ar+cell.DY
		idx := row*plotCols + col
		switch {
		case col >= plotCols || idx >= len(g.Plots) || idx < 0:
			skipped = append(skipped, fmt.Sprintf("%s: off the edge of the garden", sp.Common))
		case !g.Plots[idx].Empty():
			skipped = append(skipped, fmt.Sprintf("bed %d: already holds %s", idx+1, g.Plots[idx].DisplayName()))
		case sp.Kind == KindAquatic && !g.Plots[idx].Pond:
			skipped = append(skipped, fmt.Sprintf("bed %d: %s needs a pond", idx+1, sp.Common))
		case sp.Kind != KindAquatic && g.Plots[idx].Pond:
			skipped = append(skipped, fmt.Sprintf("bed %d: a pond", idx+1))
		default:
			ghosts[idx] = g.seedSource(sp, cell.Variety, cell.Want, used)
		}
	}
	return ghosts, skipped
}

// PlanCost is the gold it would take to buy the seed a plan lacks.
func (g *Garden) PlanCost(ghosts map[int]ghost) (cost, fromShed, locked int) {
	for _, gh := range ghosts {
		sp := gh.species()
		switch {
		case gh.Packet != 0:
			fromShed++
		case sp != nil && !g.Unlocked(sp):
			locked++
		case sp != nil:
			cost += sp.seedPrice()
		}
	}
	return cost, fromShed, locked
}

// withGhosts is a copy of the garden with the planned plants standing in
// their beds, grown, so their neighbourhoods can be scored before anything is
// sown. It shares nothing with the garden it came from.
func (g *Garden) withGhosts(ghosts map[int]ghost) *Garden {
	c := *g
	c.Plots = append([]Plot(nil), g.Plots...)
	for idx, gh := range ghosts {
		if idx < 0 || idx >= len(c.Plots) {
			continue
		}
		c.Plots[idx] = Plot{
			SpeciesID: gh.Species, Variety: gh.Variety, Genome: gh.Genome,
			Growth: 1, Matured: true, Moisture: 0.7, PH: c.Plots[idx].PH, Richness: c.Plots[idx].Richness, Pond: c.Plots[idx].Pond,
		}
	}
	return &c
}

// CommitGhosts sows a plan: shed seed where there is some, bought seed
// otherwise. It reports what it did and anything it could not.
func (g *Garden) CommitGhosts(ghosts map[int]ghost, now time.Time) (sown, bought int, spent int, problems []string) {
	beds := make([]int, 0, len(ghosts))
	for idx := range ghosts {
		beds = append(beds, idx)
	}
	sort.Ints(beds)
	for _, idx := range beds {
		gh := ghosts[idx]
		sp := gh.species()
		if sp == nil {
			continue
		}
		if gh.Packet != 0 {
			if pi := g.packetIndex(gh.Packet); pi >= 0 {
				if err := g.SowPacket(idx, pi, now); err != nil {
					problems = append(problems, fmt.Sprintf("bed %d: %v", idx+1, err))
					continue
				}
				sown++
				continue
			}
		}
		before := g.Gold
		if err := g.Plant(idx, sp, gh.Variety, now); err != nil {
			problems = append(problems, fmt.Sprintf("bed %d: %v", idx+1, err))
			continue
		}
		spent += before - g.Gold
		bought++
		sown++
	}
	return sown, bought, spent, problems
}

// packetIndex finds a packet in the shed by its ID, or -1.
func (g *Garden) packetIndex(id int64) int {
	for i := range g.Shed {
		if g.Shed[i].ID == id {
			return i
		}
	}
	return -1
}
