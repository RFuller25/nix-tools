package main

import (
	"fmt"
	"math"
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
	// Gifted lines came from a friend's code; From is who made them.
	Gifted bool   `json:"gifted,omitempty"`
	From   string `json:"from,omitempty"`
	// Tol is the definition: how far each gene may stray from Genome (the
	// ideal) before a plant is no longer this cultivar. Zero means the default.
	Tol Tolerance `json:"tol,omitzero"`
	// History is how the line came to be, one step a generation.
	History []Step `json:"history,omitempty"`
	// Grown is when a plant of the line first flowered here; a found line was
	// grown the day it was found, a gift is not until one of its seeds flowers.
	Grown time.Time `json:"grown,omitzero"`
}

func (c Cultivar) SpeciesRef() *Species { return SpeciesByID(c.Species) }

// describeGenome is a few words on what a genome looks like.
func describeGenome(g Genome) string {
	return g.ColourName() + ", " + g.HeightWord()
}

// cultivarFor finds the line a genome belongs to, if any: one whose definition
// admits it, the closest if several do.
func (g *Garden) cultivarFor(sp *Species, gn Genome) *Cultivar {
	var best *Cultivar
	bestD := math.MaxFloat64
	for i := range g.Cultivars {
		c := &g.Cultivars[i]
		if c.Species != sp.ID || !c.Admits(gn) {
			continue
		}
		if d := c.Genome.Distance(gn); d < bestD {
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
	if c != nil && !c.Admits(gn) {
		// It has drifted out of its definition: an off-type, not the variety.
		g.Log(when, "%s in bed %d has drifted out of ‘%s’: %s.", p.DisplayName(), idx+1, c.Name, c.strayWords(gn))
		p.Line, p.LineName, p.Streak = 0, "", 0
		c = nil
	}
	if c == nil {
		if !p.IsHybrid() {
			return
		}
		c = g.cultivarFor(sp, gn)
	}
	if c == nil && g.forgotten(sp, gn) {
		return // a line the gardener deleted; it is not written down again
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
			History: copyLineage(p.History), Grown: when,
		})
		c = &g.Cultivars[len(g.Cultivars)-1]
		g.Log(when, "A new hybrid flowered in bed %d: %s, %s. It is in the almanac as ‘%s’.",
			idx+1, describeGenome(gn), sp.Common, c.Name)
	}
	p.Line, p.LineName = c.ID, c.Name
	if c.Grown.IsZero() {
		c.Grown = when
		g.Log(when, "‘%s’ has flowered in your garden for the first time.", c.Name)
	}
	if p.Stable() && !c.Stable {
		c.Stable = true
		g.Log(when, "‘%s’ has bred true for %d generations: it is now a stable line.", c.Name, stableRuns)
	}
}

// nameSeedLine files a hybrid packet the gardener has named in the almanac
// straight away, without waiting for a plant of it to flower. Seed that is
// still the shop's named form is not new, so is left alone.
func (g *Garden) nameSeedLine(packet int, label string, now time.Time) {
	pk := &g.Shed[packet]
	label = strings.TrimSpace(label)
	sp := pk.Species()
	if label == "" || sp == nil || pk.Pure {
		return
	}
	if pk.Line != 0 && g.CultivarByID(pk.Line) != nil {
		g.RenameCultivar(pk.Line, label, now)
		return
	}
	gn := pk.Mean()
	variety, gap := sp.NearestVariety(gn)
	c := g.cultivarFor(sp, gn)
	if c == nil {
		if gap <= hybridGap {
			return
		}
		g.unforget(sp, gn)
		g.CultivarSeq++
		g.Cultivars = append(g.Cultivars, Cultivar{
			ID: g.CultivarSeq, Name: label, Species: sp.ID, Genome: gn, Gen: pk.Gen,
			Descent: pk.Descent, Found: now, Variety: variety, Named: true, Stable: pk.Stable(),
			History: extendLineage(pk.History, pk.pendingStep(sp, gn)),
		})
		c = &g.Cultivars[len(g.Cultivars)-1]
		g.Log(now, "Named a new hybrid seed ‘%s’: it is in the almanac.", label)
	} else if !c.Named {
		g.RenameCultivar(c.ID, label, now)
	}
	pk.Line = c.ID
	pk.Label = c.Name
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

// UniqueCultivarsOf lists the lines of one species with each genome once, the
// first found, for showing beside the species' named forms.
func (g *Garden) UniqueCultivarsOf(sp *Species) []Cultivar {
	var out []Cultivar
	seen := map[Genome]bool{}
	for _, c := range g.CultivarsOf(sp) {
		if seen[c.Genome] {
			continue
		}
		seen[c.Genome] = true
		out = append(out, c)
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
