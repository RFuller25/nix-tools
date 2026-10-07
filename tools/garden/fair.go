package main

import (
	"fmt"
	"math"
	"time"
)

// The fair. Once a week the village holds a show. It names a category, you
// enter one plant that you think suits it, and when the week is over the
// judges place it against a field of other growers. Places pay gold and earn
// a ribbon for the almanac. It is you against a bar that rises as you win, and
// nothing is lost by coming last.
//
// The judging is a hash of the garden's seed and the week, so it comes out the
// same however the garden was kept: a show missed while the program was closed
// is judged exactly as if it had been there.

const (
	fairRivals     = 5    // other growers in every class
	fairRivalMean  = 52.0 // how good the field is to begin with
	fairRivalSpan  = 16.0 // and how widely it varies
	fairBarPerWin  = 1.5  // how far the field's mean rises for each first place you have won
	fairBarMax     = 16.0
	fairHealthPart = 15.0 // points of a score that are about how well the plant is kept
	prizeFirst     = 50
	prizeSecond    = 30
	prizeThird     = 18
	prizeFourth    = 8
	prizeLast      = 5
)

// FairCategory is one class at the show.
type FairCategory struct {
	ID    string
	Name  string
	Blurb string
	Kind  Kind // for classes limited to one sort of plant; -1 for any
	Score func(sp *Species, gn Genome) float64
}

// vividness is how rich a colour is: saturated and neither washed out nor black.
func vividness(gn Genome) float64 {
	s := float64(gn.Sat) / 100
	l := float64(gn.Light) / 100
	return 100 * s * (1 - math.Min(1, math.Abs(l-0.5)*1.6))
}

func allRounder(sp *Species, gn Genome) float64 {
	balance := 100 - math.Abs(float64(gn.Shape)-50)*2
	return (float64(gn.Speed) + float64(gn.Yield) + vividness(gn) + balance) / 4
}

var fairCategories = []FairCategory{
	{ID: "colour", Name: "Finest colour", Blurb: "the richest, most unusual shade", Kind: -1, Score: func(sp *Species, gn Genome) float64 {
		_, gap := sp.NearestVariety(gn)
		return 0.6*vividness(gn) + 0.4*math.Min(100, gap*300)
	}},
	{ID: "tallest", Name: "Tallest", Blurb: "the one that stands highest for its kind", Kind: -1, Score: func(sp *Species, gn Genome) float64 { return float64(gn.Height) }},
	{ID: "compact", Name: "Neatest and most compact", Blurb: "small, slim and tidy", Kind: -1, Score: func(sp *Species, gn Genome) float64 {
		return 0.6*(100-float64(gn.Height)) + 0.4*(100-float64(gn.Shape))
	}},
	{ID: "quickest", Name: "Quickest grower", Blurb: "the one that wastes no time", Kind: -1, Score: func(sp *Species, gn Genome) float64 { return float64(gn.Speed) }},
	{ID: "abundant", Name: "Most abundant", Blurb: "heavy with seed", Kind: -1, Score: func(sp *Species, gn Genome) float64 { return float64(gn.Yield) }},
	{ID: "allround", Name: "Best all-rounder", Blurb: "quick, generous, well-coloured and well-formed together", Kind: -1, Score: allRounder},
	{ID: "flowers", Name: "Best flower", Blurb: "the best all-rounder among the flowers", Kind: KindFlower, Score: allRounder},
	{ID: "herbs", Name: "Best herb", Blurb: "the best all-rounder among the herbs", Kind: KindHerb, Score: allRounder},
	{ID: "edibles", Name: "Best for the kitchen", Blurb: "the best all-rounder among the vegetables and fruit", Kind: KindEdible, Score: allRounder},
}

// weekKey names a calendar week, Monday to Sunday.
func weekKey(t time.Time) int {
	y, w := t.ISOWeek()
	return y*100 + w
}

// FairFor is the class at the show in a given week.
func (g *Garden) FairFor(week int) FairCategory {
	return fairCategories[int(hashUnit(g.Seed, int64(week), 0xFA17)*float64(len(fairCategories)))%len(fairCategories)]
}

// Eligible reports whether a plant may be shown in a class.
func (c FairCategory) Eligible(sp *Species) bool { return c.Kind < 0 || sp.Kind == c.Kind }

// FairEntry is a plant entered in a week's show, as it stood when entered.
type FairEntry struct {
	Week    int     `json:"week"`
	Species string  `json:"species"`
	Name    string  `json:"name"`
	Genome  Genome  `json:"genome"`
	Health  float64 `json:"health"`
	Bed     int     `json:"bed"`
}

// Ribbon is a result, kept in the almanac.
type Ribbon struct {
	Week     int       `json:"week"`
	Category string    `json:"category"`
	Place    int       `json:"place"`
	Entrants int       `json:"entrants"`
	Species  string    `json:"species"`
	Name     string    `json:"name"`
	Genome   Genome    `json:"genome"`
	Score    float64   `json:"score"`
	Prize    int       `json:"prize"`
	Judged   time.Time `json:"judged"`
}

// FairState is everything the garden remembers about the show.
type FairState struct {
	Entry   *FairEntry `json:"entry,omitempty"`
	Ribbons []Ribbon   `json:"ribbons,omitempty"`
	Wins    int        `json:"wins,omitempty"`
}

// plantHealth is how well kept a plant is, 0 to 1.
func plantHealth(p *Plot) float64 {
	return 0.5*clamp01(p.Moisture/0.7) + 0.5*(1-clamp01(p.Weeds))
}

// FairScore is what the judges give a plant in a class, 0 to 100.
func FairScore(c FairCategory, sp *Species, gn Genome, health float64) float64 {
	return clamp(c.Score(sp, gn)*(1-fairHealthPart/100)+health*fairHealthPart, 0, 100)
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }

// CanEnter explains why a plant cannot be entered in this week's show, or
// returns nil.
func (g *Garden) CanEnter(bed int, now time.Time) error {
	if bed < 0 || bed >= len(g.Plots) {
		return fmt.Errorf("no such bed")
	}
	p := &g.Plots[bed]
	sp := p.Species()
	cat := g.FairFor(weekKey(now))
	switch {
	case sp == nil:
		return fmt.Errorf("nothing is growing in bed %d", bed+1)
	case p.Growth < 1:
		return fmt.Errorf("%s is not in flower yet; the judges only see plants in bloom", p.DisplayName())
	case !cat.Eligible(sp):
		return fmt.Errorf("this week's class (%s) is for %ss only", cat.Name, cat.Kind)
	}
	return nil
}

// Enter puts a plant in this week's show, replacing any earlier entry.
func (g *Garden) Enter(bed int, now time.Time) error {
	if err := g.CanEnter(bed, now); err != nil {
		return err
	}
	p := &g.Plots[bed]
	g.Fair.Entry = &FairEntry{
		Week: weekKey(now), Species: p.SpeciesID, Name: p.FullName(), Genome: p.Genes(),
		Health: plantHealth(p), Bed: bed,
	}
	g.Log(now, "Entered %s in the %s class.", p.DisplayName(), g.FairFor(weekKey(now)).Name)
	return nil
}

func ordinalPlace(n int) string {
	switch n {
	case 1:
		return "first"
	case 2:
		return "second"
	case 3:
		return "third"
	}
	return ordinal(n)
}

// settleFair judges an entry whose week is over.
func (g *Garden) settleFair(now time.Time) {
	e := g.Fair.Entry
	if e == nil || e.Week >= weekKey(now) {
		return
	}
	g.Fair.Entry = nil
	sp := SpeciesByID(e.Species)
	if sp == nil {
		return
	}
	cat := g.FairFor(e.Week)
	score := FairScore(cat, sp, e.Genome, e.Health)

	bar := math.Min(fairBarMax, float64(g.Fair.Wins)*fairBarPerWin)
	place := 1
	for i := 0; i < fairRivals; i++ {
		rival := fairRivalMean + bar + fairRivalSpan*(2*hashUnit(g.Seed, int64(e.Week), int64(i)+0x71)-1)
		if rival > score {
			place++
		}
	}
	prize := [...]int{0, prizeFirst, prizeSecond, prizeThird, prizeFourth}
	gold := prizeLast
	if place < len(prize) {
		gold = prize[place]
	}
	g.earn(gold)
	if place == 1 {
		g.Fair.Wins++
	}
	g.Fair.Ribbons = append(g.Fair.Ribbons, Ribbon{
		Week: e.Week, Category: cat.ID, Place: place, Entrants: fairRivals + 1, Species: e.Species,
		Name: e.Name, Genome: e.Genome, Score: score, Prize: gold, Judged: now,
	})
	g.Log(now, "The show judged %s in the %s class: %s of %d, with %.0f points. %s.",
		e.Name, cat.Name, ordinalPlace(place), fairRivals+1, score, goldLabel(gold))
}

// categoryByID finds a class by its identifier.
func categoryByID(id string) (FairCategory, bool) {
	for _, c := range fairCategories {
		if c.ID == id {
			return c, true
		}
	}
	return FairCategory{}, false
}
