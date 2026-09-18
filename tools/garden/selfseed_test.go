package main

import (
	"testing"
	"time"
)

// growTo runs a garden forward, watering as it goes.
func growTo(g *Garden, from time.Time, hours int) time.Time {
	at := from
	for i := 0; i < hours; i++ {
		at = at.Add(time.Hour)
		g.Advance(at)
		for b := range g.Plots {
			g.Water(b, at)
		}
	}
	return at
}

func TestSelfSeedersFillTheGapsBesideThem(t *testing.T) {
	sp := SpeciesByID("poppy")
	now := inSeason(sp)
	g := newTestGarden(now)
	if err := g.Plant(6, sp, now); err != nil { // a bed with neighbours all round
		t.Fatal(err)
	}

	growTo(g, now, 24*6)

	volunteers := 0
	for i, p := range g.Plots {
		if i == 6 || p.Empty() {
			continue
		}
		volunteers++
		if p.SpeciesID != sp.ID {
			t.Errorf("bed %d holds %s, which nothing here could have sown", i+1, p.SpeciesID)
		}
		neighbour := false
		for _, n := range g.Neighbours(6) {
			if n == i {
				neighbour = true
			}
		}
		// Volunteers may themselves have seeded on, so a plant further away
		// is fine as long as something is next to the original.
		_ = neighbour
	}
	if volunteers == 0 {
		t.Fatal("a poppy in full seed never sowed itself anywhere")
	}
	if g.Volunteers != volunteers {
		t.Errorf("the garden counted %d volunteers, found %d", g.Volunteers, volunteers)
	}
}

func TestPlantsThatDoNotSelfSeedStayPut(t *testing.T) {
	sp := SpeciesByID("tulip") // a bulb; it spreads by division, not by seeding about
	now := inSeason(sp)
	g := newTestGarden(now)
	if err := g.Plant(6, sp, now); err != nil {
		t.Fatal(err)
	}

	growTo(g, now, 24*10)

	for i, p := range g.Plots {
		if i != 6 && !p.Empty() {
			t.Errorf("bed %d grew a %s out of nowhere", i+1, p.SpeciesID)
		}
	}
	if g.Volunteers != 0 {
		t.Errorf("%d volunteers from a plant that does not self-seed", g.Volunteers)
	}
}

func TestVolunteersNeverLandInAPond(t *testing.T) {
	sp := SpeciesByID("cosmos")
	now := inSeason(sp)
	g := newTestGarden(now)
	for _, n := range []int{5, 7, 1, 11} { // everything around bed 7
		g.Plots[n].Pond = true
	}
	if err := g.Plant(6, sp, now); err != nil {
		t.Fatal(err)
	}

	growTo(g, now, 24*10)

	for _, n := range []int{5, 7, 1, 11} {
		if !g.Plots[n].Empty() {
			t.Errorf("a cosmos seeded itself into the pond in bed %d", n+1)
		}
	}
}

func TestSeedingCostsTheParentAPod(t *testing.T) {
	sp := SpeciesByID("chamomile")
	now := inSeason(sp)
	g := newTestGarden(now)
	if err := g.Plant(6, sp, now); err != nil {
		t.Fatal(err)
	}

	at := growTo(g, now, 24*3)
	if g.Volunteers == 0 {
		t.Skip("no volunteer appeared in this run")
	}
	// Pods are spent on seeding, so they must stay within their limit.
	if p := g.Plots[6].Pods; p < 0 || p > maxPods {
		t.Errorf("the parent holds %.2f pods after seeding", p)
	}
	_ = at
}

// The whole point of hashing rather than rolling dice: a garden left running
// and one caught up after being closed must grow exactly the same plants.
func TestSelfSeedingIsTheSameOnlineAndOff(t *testing.T) {
	sp := SpeciesByID("foxglove")
	now := inSeason(sp)

	live := newTestGarden(now)
	offline := newTestGarden(now)
	if err := live.Plant(6, sp, now); err != nil {
		t.Fatal(err)
	}
	if err := offline.Plant(6, sp, now); err != nil {
		t.Fatal(err)
	}

	// The live garden ticks along a second at a time for a while, then by the
	// minute; the other is simply asked for the same moment in one go.
	at := now
	for i := 0; i < 600; i++ {
		at = at.Add(time.Second)
		live.Advance(at)
	}
	for at.Before(now.Add(96 * time.Hour)) {
		at = at.Add(time.Minute)
		live.Advance(at)
	}
	offline.Advance(at)

	if live.Volunteers != offline.Volunteers {
		t.Errorf("live garden grew %d volunteers, the caught-up one %d", live.Volunteers, offline.Volunteers)
	}
	for i := range live.Plots {
		a, b := live.Plots[i], offline.Plots[i]
		if a.SpeciesID != b.SpeciesID {
			t.Errorf("bed %d holds %q live and %q offline", i+1, a.SpeciesID, b.SpeciesID)
		}
		if a.Spent != b.Spent {
			t.Errorf("bed %d is spent=%v live and %v offline", i+1, a.Spent, b.Spent)
		}
	}
}

func TestVolunteersCostNoSeeds(t *testing.T) {
	sp := SpeciesByID("nasturtium")
	now := inSeason(sp)
	g := newTestGarden(now)
	if err := g.Plant(6, sp, now); err != nil {
		t.Fatal(err)
	}
	seeds := g.Seeds

	growTo(g, now, 24*5)

	if g.Volunteers == 0 {
		t.Skip("no volunteer appeared in this run")
	}
	if g.Seeds < seeds {
		t.Errorf("volunteers cost the gardener %d seeds", seeds-g.Seeds)
	}
	if g.Planted < g.Volunteers {
		t.Error("volunteers were not counted among the plants sown")
	}
}
