package main

import (
	"fmt"
	"strings"
)

// Neighbours, by their genes. Alongside the planting rules, what a plant is
// like matters to what grows beside it: a giant shades a sun-lover, a bushy
// plant crowds a smaller one, and a heavy cropper takes more than its share of
// the water. These work on the genes of the actual plants, so the same species
// can help or hinder depending on how it has been bred.

const (
	shadeHeightCM  = 80  // a neighbour has to be at least this tall to shade anything
	shadeRatio     = 2.0 // and at least this many times as tall as the plant it shades
	shadeDelta     = -0.06
	shelterDelta   = 0.05 // a shade-lover beside a tall plant
	crowdShape     = 75   // a neighbour fuller than this crowds a smaller plant
	crowdDelta     = -0.04
	thirstyYield   = 75 // a neighbour that crops harder than this takes the water
	thirstyDelta   = -0.03
	crowdSizeRatio = 0.6 // it only crowds plants it is bigger than this fraction of
)

// sunLover reports a plant that wants open sun.
func sunLover(sp *Species) bool {
	s := strings.ToLower(sp.Sun)
	return strings.HasPrefix(s, "full sun") || strings.HasPrefix(s, "sun") || strings.HasPrefix(s, "head in sun")
}

// shadeLover reports a plant that wants shade.
func shadeLover(sp *Species) bool {
	s := strings.ToLower(sp.Sun)
	return strings.HasPrefix(s, "shade") || strings.HasPrefix(s, "light shade") || strings.HasPrefix(s, "dappled")
}

// geneEffects is what the beds beside idx are doing to it because of how they
// have been bred.
func (g *Garden) geneEffects(idx int, sp *Species) []companionEffect {
	self := &g.Plots[idx]
	if self.Empty() {
		return nil
	}
	selfCM := sp.HeightCM(self.Genes())
	var out []companionEffect
	for _, n := range g.Neighbours(idx) {
		o := &g.Plots[n]
		osp := o.Species()
		if osp == nil {
			continue
		}
		og := o.Genes()
		oCM := osp.HeightCM(og)
		add := func(d float64, note string) {
			out = append(out, companionEffect{Bed: n, Other: osp, Delta: d, Note: note})
		}
		if oCM >= shadeHeightCM && oCM >= selfCM*shadeRatio {
			switch {
			case sunLover(sp):
				add(shadeDelta, fmt.Sprintf("a %s %s in the next bed (%s) shades it", og.HeightWord(), osp.Common, osp.HeightText(og)))
			case shadeLover(sp):
				add(shelterDelta, fmt.Sprintf("the %s beside it (%s) gives it the shade it likes", osp.Common, osp.HeightText(og)))
			}
		}
		if og.Shape >= crowdShape && oCM >= selfCM*crowdSizeRatio && o.Growth > 0.5 {
			add(crowdDelta, fmt.Sprintf("a %s %s crowds it", og.ShapeWord(), osp.Common))
		}
		if og.Yield >= thirstyYield && o.Growth > 0.5 {
			add(thirstyDelta, fmt.Sprintf("a %s %s takes the water", og.YieldWord(), osp.Common))
		}
	}
	return out
}

// geneRuleDocs are the gene-based neighbour rules in words, for the cards and
// the guide; the numbers are the ones the simulation uses.
func geneRuleDocs() []effect {
	return []effect{
		{Glyph: "⚘", Label: "shade", Text: fmt.Sprintf("%+.0f%% to a sun-loving neighbour when it is %.0f cm or more and %.0fx its height; %+.0f%% to a shade-lover", shadeDelta*100, float64(shadeHeightCM), shadeRatio, shelterDelta*100)},
		{Glyph: "⚘", Label: "crowding", Text: fmt.Sprintf("%+.0f%% to a neighbour when it is full and bushy (shape over %d) and about its size", crowdDelta*100, crowdShape)},
		{Glyph: "⚘", Label: "thirst", Text: fmt.Sprintf("%+.0f%% to a neighbour when it is a heavy cropper (yield over %d)", thirstyDelta*100, thirstyYield)},
	}
}

// Synergy is how a bed is getting on with the beds around it: what they do for
// it, what it does for them, and the two together.
type Synergy struct {
	Take, Give float64
}

// Net is the overall effect of this bed's neighbourhood.
func (s Synergy) Net() float64 { return s.Take + s.Give }

// Synergy works out what a planted bed receives from its neighbours and
// gives to them.
func (g *Garden) Synergy(idx int) Synergy {
	sp := g.Plots[idx].Species()
	if sp == nil {
		return Synergy{}
	}
	var s Synergy
	s.Take = g.companionFactor(idx, sp) - 1
	for _, n := range g.Neighbours(idx) {
		nsp := g.Plots[n].Species()
		if nsp == nil {
			continue
		}
		for _, e := range g.companionEffects(n, nsp) {
			if e.Bed == idx {
				s.Give += e.Delta
			}
		}
	}
	return s
}

// SynergyWord says the net effect in a word.
func SynergyWord(net float64) string {
	switch {
	case net >= 0.15:
		return "thriving together"
	case net >= 0.04:
		return "good company"
	case net > -0.04:
		return "neutral"
	case net > -0.12:
		return "a little cramped"
	}
	return "working against each other"
}
