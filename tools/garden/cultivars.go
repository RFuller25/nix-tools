package main

import (
	"fmt"
	"strings"
	"time"
)

// Cultivars. A plant that comes into flower unlike every named form is a new
// hybrid, and the garden writes it down. A line that has bred true for
// stableRuns generations is stable. Cultivars can be renamed in the almanac.

// lineMatch is how alike two genomes must be to count as the same line.
const lineMatch = 0.08

// Cultivar is one hybrid line the garden has discovered.
type Cultivar struct {
	ID      int       `json:"id"`
	Name    string    `json:"name"`
	Species string    `json:"species"`
	Genome  Genome    `json:"genome"`
	Gen     int       `json:"gen"`
	Descent string    `json:"descent,omitempty"`
	Found   time.Time `json:"found"`
	Stable  bool      `json:"stable,omitempty"`
	Named   bool      `json:"named,omitempty"`
	Variety int       `json:"variety,omitempty"`
}

func (c Cultivar) SpeciesRef() *Species { return SpeciesByID(c.Species) }

// describeGenome is a few words on what a genome looks like.
func describeGenome(g Genome) string {
	return g.ColourName() + ", " + g.HeightWord()
}

// cultivarFor finds the discovered line a genome belongs to, if any.
func (g *Garden) cultivarFor(sp *Species, gn Genome) *Cultivar {
	var best *Cultivar
	bestD := lineMatch
	for i := range g.Cultivars {
		c := &g.Cultivars[i]
		if c.Species != sp.ID {
			continue
		}
		if d := c.Genome.Distance(gn); d <= bestD {
			best, bestD = c, d
		}
	}
	return best
}

// CultivarByID finds a cultivar by its number.
func (g *Garden) CultivarByID(id int) *Cultivar {
	for i := range g.Cultivars {
		if g.Cultivars[i].ID == id {
			return &g.Cultivars[i]
		}
	}
	return nil
}

// discover files a plant that has just come into flower: as a new hybrid if it
// is unlike every named form and every line already found, or into the line it
// belongs to.
func (g *Garden) discover(p *Plot, sp *Species, idx int, at, now time.Time) {
	gn := p.Genes()
	when := at
	if when.After(now) {
		when = now
	}

	c := g.CultivarByID(p.Line)
	if c == nil {
		if !p.IsHybrid() {
			return
		}
		c = g.cultivarFor(sp, gn)
	}
	if c == nil {
		n := 1
		for _, o := range g.Cultivars {
			if o.Species == sp.ID {
				n++
			}
		}
		g.CultivarSeq++
		variety, _ := sp.NearestVariety(gn)
		g.Cultivars = append(g.Cultivars, Cultivar{
			ID: g.CultivarSeq, Name: fmt.Sprintf("%s hybrid No. %d", sp.Common, n), Species: sp.ID,
			Genome: gn, Gen: p.Gen, Descent: p.Descent, Found: when, Variety: variety,
		})
		c = &g.Cultivars[len(g.Cultivars)-1]
		g.Log(when, "A new hybrid flowered in bed %d: %s, %s. It is in the almanac as ‘%s’.",
			idx+1, describeGenome(gn), sp.Common, c.Name)
	}
	p.Line, p.LineName = c.ID, c.Name
	if p.Stable() && !c.Stable {
		c.Stable = true
		g.Log(when, "‘%s’ has bred true for %d generations: it is now a stable line.", c.Name, stableRuns)
	}
}

// RenameCultivar gives a discovered line a name of the gardener's choosing,
// carrying it to the plants and seed that belong to it.
func (g *Garden) RenameCultivar(id int, name string, now time.Time) bool {
	c := g.CultivarByID(id)
	name = strings.TrimSpace(name)
	if c == nil || name == "" {
		return false
	}
	old := c.Name
	c.Name, c.Named = name, true
	for i := range g.Plots {
		if g.Plots[i].Line == id {
			g.Plots[i].LineName = name
		}
	}
	for i := range g.Shed {
		if g.Shed[i].Line == id {
			g.Shed[i].Label = name
		}
	}
	g.Log(now, "Named the line ‘%s’ ‘%s’.", old, name)
	return true
}

// CultivarsOf lists the discovered lines of one species.
func (g *Garden) CultivarsOf(sp *Species) []Cultivar {
	var out []Cultivar
	for _, c := range g.Cultivars {
		if c.Species == sp.ID {
			out = append(out, c)
		}
	}
	return out
}

// StableLines counts how many discovered lines have settled.
func (g *Garden) StableLines() int {
	n := 0
	for _, c := range g.Cultivars {
		if c.Stable {
			n++
		}
	}
	return n
}
