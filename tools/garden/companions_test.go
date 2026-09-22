package main

import (
	"testing"
	"time"
)

// Every rule must name plants that actually exist, or it silently does
// nothing forever.
func TestCompanionRulesMatchRealPlants(t *testing.T) {
	for i, c := range companions {
		if c.note == "" {
			t.Errorf("rule %d has no explanation", i)
		}
		if c.delta == 0 {
			t.Errorf("rule %d has no effect", i)
		}
		matchedWho, matchedFrom := false, false
		for _, sp := range AllSpecies() {
			if c.who(sp) {
				matchedWho = true
			}
			if c.from(sp) {
				matchedFrom = true
			}
		}
		if !matchedWho {
			t.Errorf("rule %d (%q) applies to no species in the catalogue", i, c.note)
		}
		if !matchedFrom {
			t.Errorf("rule %d (%q) is caused by no species in the catalogue", i, c.note)
		}
	}
}

func TestMarigoldsLookAfterTomatoes(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	if err := g.Plant(6, SpeciesByID("tomato"), 0, now); err != nil {
		t.Fatal(err)
	}

	plain := g.companionFactor(6, SpeciesByID("tomato"))
	if plain != 1 {
		t.Errorf("a tomato with nothing beside it has factor %.3f, want 1", plain)
	}

	if err := g.Plant(5, SpeciesByID("marigold"), 0, now); err != nil {
		t.Fatal(err)
	}
	helped := g.companionFactor(6, SpeciesByID("tomato"))
	if helped <= plain {
		t.Errorf("a marigold next door gave factor %.3f, no better than %.3f", helped, plain)
	}

	effects := g.companionEffects(6, SpeciesByID("tomato"))
	if len(effects) != 1 || effects[0].Other.ID != "marigold" {
		t.Fatalf("effects on the tomato = %+v", effects)
	}
	if effects[0].Bed != 5 {
		t.Errorf("the effect is credited to bed %d, want 6", effects[0].Bed+1)
	}
}

func TestMintCrowdsItsNeighbours(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	if err := g.Plant(6, SpeciesByID("basil"), 0, now); err != nil {
		t.Fatal(err)
	}
	if err := g.Plant(7, SpeciesByID("mint"), 0, now); err != nil {
		t.Fatal(err)
	}

	if f := g.companionFactor(6, SpeciesByID("basil")); f >= 1 {
		t.Errorf("basil beside mint has factor %.3f, want less than 1", f)
	}
	aside := companionAside(g, 6, SpeciesByID("basil"))
	if aside == "" {
		t.Error("planting beside mint said nothing")
	}
}

func TestLegumesFeedWhateverIsBesideThem(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	if err := g.Plant(6, SpeciesByID("sweetcorn"), 0, now); err != nil {
		t.Fatal(err)
	}
	if err := g.Plant(1, SpeciesByID("broadbean"), 0, now); err != nil {
		t.Fatal(err)
	}

	if f := g.companionFactor(6, SpeciesByID("sweetcorn")); f <= 1 {
		t.Errorf("corn beside beans has factor %.3f", f)
	}
	// And the beans get their climbing frame in return.
	if f := g.companionFactor(1, SpeciesByID("broadbean")); f <= 1 {
		t.Errorf("beans beside corn have factor %.3f", f)
	}
}

func TestCompanionEffectsAreCapped(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	sp := SpeciesByID("tomato")
	if err := g.Plant(6, sp, 0, now); err != nil {
		t.Fatal(err)
	}
	// Surround it with every good neighbour going.
	for i, id := range []string{"marigold", "basil", "pea", "nasturtium"} {
		bed := g.Neighbours(6)[i]
		if err := g.Plant(bed, SpeciesByID(id), 0, now); err != nil {
			t.Fatal(err)
		}
	}
	if f := g.companionFactor(6, sp); f > 1.3001 {
		t.Errorf("a perfectly companioned bed reached %.3f, above the cap", f)
	}

	// And the other way: nothing should ever stall a plant completely.
	g2 := newTestGarden(now)
	if err := g2.Plant(6, SpeciesByID("basil"), 0, now); err != nil {
		t.Fatal(err)
	}
	for _, bed := range g2.Neighbours(6) {
		g2.Plots[bed] = Plot{SpeciesID: "mint", Growth: 1, PH: 6.5, Richness: 0.5}
	}
	if f := g2.companionFactor(6, SpeciesByID("basil")); f < 0.8 {
		t.Errorf("a badly companioned bed fell to %.3f", f)
	}
}

func TestCompanionsOnlyCountNeighbours(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	if err := g.Plant(0, SpeciesByID("tomato"), 0, now); err != nil {
		t.Fatal(err)
	}
	// A marigold on the far side of the garden helps nobody.
	if err := g.Plant(14, SpeciesByID("marigold"), 0, now); err != nil {
		t.Fatal(err)
	}
	if f := g.companionFactor(0, SpeciesByID("tomato")); f != 1 {
		t.Errorf("a distant marigold gave factor %.3f", f)
	}
}

// Good company should show in the growing, not just on the info card.
func TestCompanionsChangeHowFastThingsGrow(t *testing.T) {
	now := inSeason(SpeciesByID("tomato"))

	alone := newTestGarden(now)
	paired := newTestGarden(now)
	if err := alone.Plant(6, SpeciesByID("tomato"), 0, now); err != nil {
		t.Fatal(err)
	}
	if err := paired.Plant(6, SpeciesByID("tomato"), 0, now); err != nil {
		t.Fatal(err)
	}
	if err := paired.Plant(5, SpeciesByID("marigold"), 0, now); err != nil {
		t.Fatal(err)
	}

	at := now
	for i := 0; i < 8; i++ {
		at = at.Add(time.Hour)
		alone.Advance(at)
		paired.Advance(at)
		alone.Water(6, at)
		paired.Water(6, at)
		paired.Water(5, at)
	}

	if paired.Plots[6].Growth <= alone.Plots[6].Growth {
		t.Errorf("the companioned tomato reached %.4f, the lonely one %.4f",
			paired.Plots[6].Growth, alone.Plots[6].Growth)
	}
}
