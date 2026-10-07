package main

import (
	"fmt"
	"time"
)

// Pollination. A flower that is open, with a neighbour of its own kind also in
// flower, may take that neighbour's pollen. Whatever it is carrying when its
// seed is gathered becomes the other parent of that seed.
//
// Nature does it on its own: bees by day, moths for the night flowers, and
// less of either in the wet. A gardener can do it better, with a brush: pick a
// donor anywhere in the garden and the next seed is certainly that cross.
//
// Everything here is decided by hashes on quarter-hour boundaries, as
// self-seeding is, never by the animals drawn on screen, so a garden left
// running and one catching up on a fortnight closed make the same crosses.

// Pollination odds per quarter hour for an open flower with a mate beside it.
const (
	beeOdds     = 0.10 // by day, dry
	mothOdds    = 0.09 // after dark, for the night-scented flowers
	wetFactor   = 0.12 // rain washes pollen away and keeps the bees in
	winterOdds  = 0.35 // few insects in winter
	extraMates  = 0.5  // each further neighbour in flower adds this much of the odds
	maxPollenAt = 0.45 // never more likely than this per quarter hour
)

// flowerOpen reports whether a plant has a flower open to be pollinated right
// now: grown, not finished, awake for the season, and not shut for the night.
func flowerOpen(sp *Species, p *Plot, season Season, ph phase) bool {
	if sp == nil || p.Empty() || p.Growth < 1 || p.Spent || dormant(sp, season) {
		return false
	}
	if sp.ClosesAtNight() && ph.Dark() {
		return false
	}
	if sp.OpensAtNight() && !ph.Dark() && ph != phaseDusk {
		return false
	}
	return true
}

// pollinatorOdds is how likely an open flower is to be visited in a quarter
// hour, before counting its mates.
func pollinatorOdds(sp *Species, w Weather, season Season, ph phase) float64 {
	night := ph.Dark() || ph == phaseDusk
	var odds float64
	switch {
	case night && (sp.NightScented() || sp.OpensAtNight()):
		odds = mothOdds
	case night:
		return 0
	case attractsBees(sp):
		odds = beeOdds
	default:
		return 0
	}
	if w.Wet() {
		odds *= wetFactor
	}
	if season == Winter {
		odds *= winterOdds
	}
	return odds
}

// maybePollinate lets nature cross a flower with a neighbour of its own kind.
func (g *Garden) maybePollinate(p *Plot, idx int, sp *Species, w Weather, season Season, at time.Time, dt float64) {
	ph := phaseAt(at)
	if !flowerOpen(sp, p, season, ph) {
		return
	}
	base := pollinatorOdds(sp, w, season, ph)
	if base <= 0 {
		return
	}
	var mates []int
	for _, n := range g.Neighbours(idx) {
		o := &g.Plots[n]
		if o.SpeciesID == p.SpeciesID && flowerOpen(sp, o, season, ph) {
			mates = append(mates, n)
		}
	}
	if len(mates) == 0 {
		return
	}
	odds := base * (1 + extraMates*float64(len(mates)-1))
	if odds > maxPollenAt {
		odds = maxPollenAt
	}

	start := at.Unix() / quarterSeconds
	end := at.Add(time.Duration(dt*float64(time.Hour))).Unix() / quarterSeconds
	for q := start + 1; q <= end; q++ {
		if hashUnit(g.Seed, q, int64(idx)+0xB0B0) > odds {
			continue
		}
		donor := mates[int(hashUnit(g.Seed, q, int64(idx)+0xD0D0)*float64(len(mates)))%len(mates)]
		pollen := g.Plots[donor].Genes()
		p.Pollen = &pollen
		g.Crossed++
		return
	}
}

// HandPollinate carries pollen from one flower to another with a brush. Both
// must be of one species and in open flower, and the weather fit for it.
func (g *Garden) HandPollinate(target, donor int, now time.Time) error {
	if target < 0 || target >= len(g.Plots) || donor < 0 || donor >= len(g.Plots) {
		return fmt.Errorf("no such bed")
	}
	if target == donor {
		return fmt.Errorf("a flower cannot be its own pollen donor — choose another bed")
	}
	t, d := &g.Plots[target], &g.Plots[donor]
	sp := t.Species()
	if sp == nil {
		return fmt.Errorf("nothing is growing in bed %d", target+1)
	}
	if d.SpeciesID != t.SpeciesID {
		return fmt.Errorf("pollen only takes between plants of one species — %s will not set %s", d.DisplayName(), t.DisplayName())
	}
	season, ph, w := g.Season(now), phaseAt(now), g.Weather(now)
	if !flowerOpen(sp, t, season, ph) {
		return fmt.Errorf("%s has no open flower to pollinate right now", t.DisplayName())
	}
	if !flowerOpen(sp, d, season, ph) {
		return fmt.Errorf("%s has no open flower to take pollen from right now", d.DisplayName())
	}
	if w.Kind == Rain || w.Kind == Storm || w.Kind == Snow {
		return fmt.Errorf("the %s has the pollen washed out of the air — try again when it is dry", w.Name())
	}
	pollen := d.Genes()
	t.Pollen = &pollen
	g.Crossed++
	g.HandCrossed++
	g.Log(now, "Brushed pollen from %s (bed %d) onto %s (bed %d).", d.DisplayName(), donor+1, t.DisplayName(), target+1)
	return nil
}
