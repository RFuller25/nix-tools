package main

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// The shop sells cultivars too: any line in your almanac, whether you bred it
// or took it in from a friend's code, can be bought back as seed. It costs
// several times what an ordinary seed of its species does, because the shop is
// selling a line somebody spent generations on; and it comes exactly true to
// the cultivar's ideal, like seed of any named form. The price is always well
// above what the shed pays for the same seed, so buying and selling is no way
// to make gold.

// Price is what the shop asks for one seed of the cultivar.
func (c Cultivar) Price() int {
	sp := c.SpeciesRef()
	if sp == nil {
		return 0
	}
	p := float64(sp.seedPrice() * cultivarCost)
	if c.Stable {
		p *= 1 + cultivarStable
	}
	return int(math.Round(p))
}

// cultivarPacket is n seeds of a cultivar as the shop sells them.
func (g *Garden) cultivarPacket(c *Cultivar, n int) Packet {
	streak := 0
	if c.Stable {
		streak = stableRuns
	}
	return Packet{
		SpeciesID: c.Species, A: c.Genome, B: c.Genome, Count: n, Pure: true, Gen: c.Gen,
		Variety: c.Variety, Label: c.Name, Line: c.ID, Streak: streak, Kind: stepCultivar,
		From: "the shop", Descent: "bought as the cultivar ‘" + c.Name + "’",
		History: copyLineage(c.History),
	}
}

// buyableCultivar checks a cultivar can be bought and what it costs.
func (g *Garden) buyableCultivar(id int) (*Cultivar, *Species, error) {
	c := g.CultivarByID(id)
	if c == nil {
		return nil, nil, fmt.Errorf("no such cultivar")
	}
	sp := c.SpeciesRef()
	if sp == nil {
		return nil, nil, fmt.Errorf("%s is a plant this garden does not know", c.Name)
	}
	if !g.Unlocked(sp) {
		return nil, nil, fmt.Errorf("%s needs %d matured plants to unlock", sp.Common, sp.Unlock)
	}
	return c, sp, nil
}

// BuyCultivar puts n seeds of one of your cultivars into the shed.
func (g *Garden) BuyCultivar(id, n int, now time.Time) error {
	if n < 1 {
		return fmt.Errorf("nothing to buy")
	}
	c, _, err := g.buyableCultivar(id)
	if err != nil {
		return err
	}
	price := c.Price() * n
	if !g.spend(price) {
		return fmt.Errorf("%d × ‘%s’ costs %s (you have %s)", n, c.Name, goldLabel(price), goldLabel(g.Gold))
	}
	g.AddPacket(g.cultivarPacket(c, n))
	g.Log(now, "Bought %d × ‘%s’ from the shop for %s.", n, c.Name, goldLabel(price))
	return nil
}

// PlantCultivar buys one seed of a cultivar and sows it at once.
func (g *Garden) PlantCultivar(idx, id int, now time.Time) error {
	c, sp, err := g.buyableCultivar(id)
	if err != nil {
		return err
	}
	if err := g.checkSowable(idx, sp); err != nil {
		return err
	}
	if err := g.BuyCultivar(id, 1, now); err != nil {
		return err
	}
	for i := range g.Shed {
		if pk := g.Shed[i]; pk.Pure && pk.Line == c.ID && pk.SpeciesID == c.Species {
			return g.SowPacket(idx, i, now)
		}
	}
	return fmt.Errorf("the seed was lost on the way to the bed")
}

// shopForm is one thing the shop sells under a species: a named form, or one
// of your cultivars.
type shopForm struct {
	Name     string
	Note     string
	Genome   Genome
	Variety  int // the named form it is drawn on
	Price    int
	Cultivar *Cultivar
}

// varietyForms lists everything the shop sells of a species: its named forms,
// then your cultivars of it.
func (g *Garden) varietyForms(sp *Species) []shopForm {
	var out []shopForm
	for i, v := range sp.Varieties() {
		out = append(out, shopForm{Name: v.Name, Note: v.Note, Genome: sp.VarietyGenome(i), Variety: i, Price: sp.seedPrice()})
	}
	for _, c := range g.CultivarsOf(sp) {
		c := c
		note := "your own line"
		if c.Gifted {
			note = "from a friend" + fromText(c.From)
		}
		out = append(out, shopForm{
			Name: c.Name, Note: note, Genome: c.Genome, Variety: c.Variety, Price: c.Price(), Cultivar: &c,
		})
	}
	return out
}

// speciesWithCultivars lists the gnome-made crosses that have a cultivar, which
// are not in the catalogue but can still be bought.
func (g *Garden) crossSpeciesWithCultivars() []*Species {
	seen := map[string]bool{}
	var out []*Species
	for _, c := range g.Cultivars {
		sp := c.SpeciesRef()
		if sp == nil || !sp.IsCross() || seen[sp.ID] {
			continue
		}
		seen[sp.ID] = true
		out = append(out, sp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Common < out[j].Common })
	return out
}
