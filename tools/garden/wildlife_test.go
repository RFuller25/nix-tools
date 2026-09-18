package main

import (
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Creatures are drawn into a single cell of the artwork, so a wide glyph
// would tear the bed apart.
func TestCreatureGlyphsFitOneCell(t *testing.T) {
	for c, k := range creatures {
		if w := lipgloss.Width(k.glyph); w != 1 {
			t.Errorf("%s is %d columns wide", k.name, w)
		}
		if k.name == "" || k.note == "" || k.color == "" {
			t.Errorf("creature %d is missing its details", c)
		}
		if k.life <= 0 || k.speed <= 0 {
			t.Errorf("%s never moves or never leaves", k.name)
		}
	}
}

// Nothing visits an empty garden: what turns up depends on what is growing.
func TestNothingVisitsBareEarth(t *testing.T) {
	now := at(time.July, 14, 0)
	g := newTestGarden(now)
	w := newWildlife(1)

	for i := 0; i < 5000; i++ {
		w.advance(windTick.Seconds(), g, now, weatherFor(Sunny), nil)
	}
	if len(w.visitors) != 0 {
		t.Errorf("%d creatures visited a garden with nothing in it", len(w.visitors))
	}
}

func TestBeesComeToFlowers(t *testing.T) {
	now := at(time.July, 14, 0)
	g := newTestGarden(now)
	g.Plots[6] = Plot{SpeciesID: "lavender", Growth: 1, Matured: true, PH: 7, Richness: 0.5}

	w := newWildlife(2)
	seen := map[creature]bool{}
	for i := 0; i < 8000 && len(seen) == 0; i++ {
		w.advance(windTick.Seconds(), g, now, weatherFor(Sunny), func(c creature) { seen[c] = true })
	}
	if len(w.visitors) == 0 && len(seen) == 0 {
		t.Fatal("nothing came to a lavender in full flower on a sunny afternoon")
	}
}

func TestRainKeepsThePollinatorsAway(t *testing.T) {
	now := at(time.July, 14, 0)
	g := newTestGarden(now)
	g.Plots[6] = Plot{SpeciesID: "lavender", Growth: 1, Matured: true, PH: 7, Richness: 0.5}

	w := newWildlife(3)
	for i := 0; i < 4000; i++ {
		w.advance(windTick.Seconds(), g, now, weatherFor(Storm), nil)
	}
	for _, v := range w.visitors {
		if v.what == bee || v.what == butterfly {
			t.Errorf("a %s was out in a thunderstorm", v.what)
		}
	}
}

func TestMothsComeOutAtNightAndBeesDoNot(t *testing.T) {
	night := at(time.July, 23, 30)
	g := newTestGarden(night)
	g.Plots[6] = Plot{SpeciesID: "moonflower", Growth: 1, Matured: true, PH: 6.5, Richness: 0.5}

	w := newWildlife(4)
	sawMoth := false
	for i := 0; i < 8000 && !sawMoth; i++ {
		w.advance(windTick.Seconds(), g, night, weatherFor(Clear), func(c creature) {
			if c == moth {
				sawMoth = true
			}
			if c == bee || c == butterfly {
				t.Errorf("a %s was flying at half past eleven at night", c)
			}
		})
	}
	if !sawMoth {
		t.Error("no moth came to a moonflower on a clear night")
	}
}

func TestFinchesWantSeedHeads(t *testing.T) {
	now := at(time.October, 11, 0)
	g := newTestGarden(now)
	// A sunflower gone over, full of seed: exactly what a finch is after.
	g.Plots[6] = Plot{SpeciesID: "sunflower", Growth: 1, Matured: true, Pods: 3, PH: 6.5, Richness: 0.5}

	w := newWildlife(5)
	seen := false
	for i := 0; i < 20000 && !seen; i++ {
		w.advance(windTick.Seconds(), g, now, weatherFor(Cloudy), func(c creature) {
			if c == finch {
				seen = true
			}
		})
	}
	if !seen {
		t.Error("no finch came to a sunflower full of seed")
	}
}

func TestDragonfliesNeedAPond(t *testing.T) {
	now := at(time.July, 13, 0)
	dry := newTestGarden(now)
	dry.Plots[6] = Plot{SpeciesID: "cosmos", Growth: 1, Matured: true, PH: 6.5, Richness: 0.5}

	w := newWildlife(6)
	for i := 0; i < 6000; i++ {
		w.advance(windTick.Seconds(), dry, now, weatherFor(Sunny), func(c creature) {
			if c == dragonfly {
				t.Fatal("a dragonfly turned up in a garden with no water")
			}
		})
	}
}

func TestVisitorsCrossTheGardenAndLeave(t *testing.T) {
	now := at(time.July, 14, 0)
	g := newTestGarden(now)
	w := newWildlife(7)
	w.arrive(bee, 1)

	if len(w.visitors) != 1 {
		t.Fatal("the bee did not arrive")
	}
	for i := 0; i < 2000 && len(w.visitors) > 0; i++ {
		w.advance(0.05, g, now, weatherFor(Sunny), nil)
	}
	if len(w.visitors) != 0 {
		t.Error("the bee never left")
	}
}

func TestOverlayStaysInsideTheBed(t *testing.T) {
	w := newWildlife(8)
	for i := 0; i < 40; i++ {
		w.arrive(bee, i%3)
	}
	for step := 0; step < 400; step++ {
		w.advance(0.05, newTestGarden(at(time.July, 14, 0)), at(time.July, 14, 0), weatherFor(Sunny), nil)
		for bed := 0; bed < PlotCount; bed++ {
			for pos := range w.overlayFor(bed, cellInner, artHeight) {
				if pos[0] < 0 || pos[0] >= artHeight || pos[1] < 0 || pos[1] >= cellInner {
					t.Fatalf("a creature was drawn at %v, outside the bed", pos)
				}
			}
		}
	}
}

func TestFirstSightingIsRecordedOnce(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)

	if !g.sight("bee", "A bee found the garden.", now) {
		t.Fatal("the first bee was not recorded")
	}
	if g.sight("bee", "A bee found the garden.", now) {
		t.Error("the same creature was recorded twice")
	}
	if _, ok := g.Sightings["bee"]; !ok {
		t.Error("the sighting was not kept")
	}

	entries := 0
	for _, e := range g.Journal {
		if e.Text == "A bee found the garden." {
			entries++
		}
	}
	if entries != 1 {
		t.Errorf("the journal mentions the first bee %d times", entries)
	}
}

func TestSightingsSurviveASave(t *testing.T) {
	now := testStart()
	path := t.TempDir() + "/garden.json"
	g := newTestGarden(now)
	g.sight("fox", "A fox crossed the garden.", now)

	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := back.Sightings["fox"]; !ok {
		t.Error("the fox was forgotten over a save")
	}
}
