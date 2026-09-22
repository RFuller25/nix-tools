package main

import (
	"os"
	"testing"
	"time"
)

func TestGardenIsAFixedWidthWorld(t *testing.T) {
	g := newTestGarden(testStart())
	if len(g.Plots) != PlotCount {
		t.Fatalf("a new garden has %d beds, want %d", len(g.Plots), PlotCount)
	}
	if g.Rows() != PlotCount/plotCols {
		t.Errorf("a new garden has %d rows, want %d", g.Rows(), PlotCount/plotCols)
	}
}

func TestNeighboursAreTheBedsAlongside(t *testing.T) {
	g := newTestGarden(testStart())

	same := func(got []int, want ...int) bool {
		if len(got) != len(want) {
			return false
		}
		seen := map[int]bool{}
		for _, v := range got {
			seen[v] = true
		}
		for _, v := range want {
			if !seen[v] {
				return false
			}
		}
		return true
	}

	// Top-left corner: right and below only.
	if n := g.Neighbours(0); !same(n, 1, 5) {
		t.Errorf("neighbours of bed 1 = %v, want 2 and 6", n)
	}
	// Middle of the top row.
	if n := g.Neighbours(2); !same(n, 1, 3, 7) {
		t.Errorf("neighbours of bed 3 = %v", n)
	}
	// Middle of the garden: all four.
	if n := g.Neighbours(6); !same(n, 5, 7, 1, 11) {
		t.Errorf("neighbours of bed 7 = %v", n)
	}
	// The right-hand edge must not wrap onto the next row.
	for _, n := range g.Neighbours(4) {
		if n == 5 {
			t.Error("the end of a row is neighbours with the start of the next")
		}
	}
	// Nothing may point off the bottom of the garden.
	for _, n := range g.Neighbours(len(g.Plots) - 1) {
		if n >= len(g.Plots) || n < 0 {
			t.Errorf("neighbour %d is outside the garden", n)
		}
	}
}

func TestBuyingBeds(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Seeds = 0

	if err := g.BuyBed(now); err == nil {
		t.Error("buying a bed with no seeds should fail")
	}

	cost := g.BedCost()
	g.Seeds = cost
	if err := g.BuyBed(now); err != nil {
		t.Fatalf("buying a bed: %v", err)
	}
	if len(g.Plots) != PlotCount+1 {
		t.Errorf("the garden has %d beds, want %d", len(g.Plots), PlotCount+1)
	}
	if g.Seeds != 0 {
		t.Errorf("%d seeds left, want the bed to have cost all of them", g.Seeds)
	}
	if g.BedCost() <= cost {
		t.Error("each new bed should cost more than the last")
	}
	if p := g.Plots[len(g.Plots)-1]; p.PH == 0 || p.Richness == 0 {
		t.Error("new ground was broken without any soil in it")
	}

	// The garden cannot grow without limit.
	g.Seeds = 100000
	for len(g.Plots) < maxPlots {
		if err := g.BuyBed(now); err != nil {
			t.Fatalf("buying bed %d: %v", len(g.Plots)+1, err)
		}
	}
	if err := g.BuyBed(now); err == nil {
		t.Errorf("the garden grew past its %d-bed limit", maxPlots)
	}
}

func TestPondsHoldWaterAndTakeOnlyWaterPlants(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)

	if err := g.DigPond(0, now); err != nil {
		t.Fatalf("digging a pond: %v", err)
	}
	if !g.Plots[0].Pond || g.Plots[0].Moisture != 1 {
		t.Error("a freshly dug pond is not full of water")
	}

	if err := g.Plant(0, SpeciesByID("sunflower"), now); err == nil {
		t.Error("a sunflower was planted in a pond")
	}
	if err := g.Plant(0, SpeciesByID("waterlily"), now); err != nil {
		t.Errorf("planting a water lily in a pond: %v", err)
	}

	// A dry bed is no place for a water lily.
	if err := g.Plant(1, SpeciesByID("waterlily"), now); err == nil {
		t.Error("a water lily was planted on dry land")
	}

	// Ponds never dry out and never grow weeds, whatever the weather.
	g.Plots[0].Weeds = 0.9
	g.Advance(now.Add(72 * time.Hour))
	if m := g.Plots[0].Moisture; m < 0.99 {
		t.Errorf("the pond dried to %.2f", m)
	}
	if w := g.Plots[0].Weeds; w != 0 {
		t.Errorf("the pond grew %.2f of weeds", w)
	}
	if g.Water(0, now) {
		t.Error("a pond asked to be watered")
	}
}

func TestPondCanBeFilledInAgain(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	if err := g.DigPond(2, now); err != nil {
		t.Fatal(err)
	}
	if err := g.DigPond(2, now); err != nil {
		t.Fatalf("filling a pond in: %v", err)
	}
	if g.Plots[2].Pond {
		t.Error("the pond is still there")
	}

	// A planted bed has to be cleared first.
	if err := g.Plant(3, SpeciesByID("basil"), now); err != nil {
		t.Fatal(err)
	}
	if err := g.DigPond(3, now); err == nil {
		t.Error("a pond was dug underneath a growing plant")
	}
}

func TestSoilSuitsSomePlantsBetterThanOthers(t *testing.T) {
	acid := &Plot{PH: 5.2, Richness: 0.5}
	chalk := &Plot{PH: 7.6, Richness: 0.5}

	hydrangea := SpeciesByID("hydrangea") // wants acid
	lavender := SpeciesByID("lavender")   // wants lime
	sunflower := SpeciesByID("sunflower") // does not mind

	if soilFactor(hydrangea, acid) <= soilFactor(hydrangea, chalk) {
		t.Error("hydrangea should do better in acid soil")
	}
	if soilFactor(lavender, chalk) <= soilFactor(lavender, acid) {
		t.Error("lavender should do better in chalk")
	}
	if a, b := soilFactor(sunflower, acid), soilFactor(sunflower, chalk); a != b {
		t.Errorf("an easy-going plant reacted to pH: %.3f vs %.3f", a, b)
	}

	// Rich ground beats hungry ground, and neither is ever a death sentence.
	rich := &Plot{PH: 6.5, Richness: 1}
	poor := &Plot{PH: 6.5, Richness: 0}
	if soilFactor(sunflower, rich) <= soilFactor(sunflower, poor) {
		t.Error("compost should help")
	}
	if soilFactor(hydrangea, &Plot{PH: 8, Richness: 0}) < 0.7 {
		t.Error("the worst possible ground should still allow slow growth")
	}
}

// Bigleaf hydrangea reads the soil: blue in acid, pink in lime.
func TestHydrangeaFlowersWithTheSoil(t *testing.T) {
	sp := SpeciesByID("hydrangea")
	blue := sp.PaletteIn(&Plot{PH: 5.4})
	pink := sp.PaletteIn(&Plot{PH: 7.4})
	mauve := sp.PaletteIn(&Plot{PH: 6.4})

	if blue.Bloom == pink.Bloom || blue.Bloom == mauve.Bloom || pink.Bloom == mauve.Bloom {
		t.Errorf("hydrangea flowered the same colour in different soil: %v %v %v", blue.Bloom, mauve.Bloom, pink.Bloom)
	}
	if HydrangeaColour(5.4) != "blue" || HydrangeaColour(7.4) != "pink" || HydrangeaColour(6.4) != "mauve" {
		t.Error("the colours are named wrongly")
	}

	// Everything else keeps its own colours whatever the bed.
	rose := SpeciesByID("rose")
	if rose.PaletteIn(&Plot{PH: 5}) != rose.Palette {
		t.Error("a rose changed colour with the soil")
	}
}

func TestCompostingEnrichesTheBed(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Plots[0].PH = 8
	g.Plots[0].Richness = 0.1
	if err := g.Plant(0, SpeciesByID("basil"), now); err != nil {
		t.Fatal(err)
	}

	g.Uproot(0, now) //nolint:errcheck // the yield is tested elsewhere
	if g.Plots[0].Richness <= 0.1 {
		t.Errorf("composting left the bed at %.2f richness", g.Plots[0].Richness)
	}
	if g.Plots[0].PH >= 8 {
		t.Errorf("compost did not buffer the pH: %.2f", g.Plots[0].PH)
	}
	if !g.Plots[0].Empty() {
		t.Error("the bed is not empty after lifting the plant")
	}
}

func TestGrowingPlantsDrawOnTheSoil(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Plots[0].Richness = 1
	if err := g.Plant(0, SpeciesByID("pumpkin"), now); err != nil {
		t.Fatal(err)
	}

	g.Advance(now.Add(10 * time.Hour))
	if g.Plots[0].Richness >= 1 {
		t.Error("a growing plant took nothing from the soil")
	}
	if g.Plots[0].Richness < 0 {
		t.Error("richness went negative")
	}
}

func TestSoilSurvivesASaveAndLoad(t *testing.T) {
	now := testStart()
	path := t.TempDir() + "/garden.json"
	g := newTestGarden(now)
	g.DigPond(1, now)
	g.Plots[0].PH, g.Plots[0].Richness = 5.1, 0.9

	if err := Save(path, g); err != nil {
		t.Fatal(err)
	}
	back, err := Load(path, now)
	if err != nil {
		t.Fatal(err)
	}
	if back.Plots[0].PH != 5.1 || back.Plots[0].Richness != 0.9 {
		t.Errorf("soil came back as %+v", back.Plots[0])
	}
	if !back.Plots[1].Pond {
		t.Error("the pond was not saved")
	}
}

func TestOldSavesGetSoil(t *testing.T) {
	now := testStart()
	path := t.TempDir() + "/garden.json"
	// A save from before beds had any soil in them.
	body := `{"version":1,"seed":99,"plots":[{"species_id":"basil","growth":0.5}],"seeds":4}`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	g, err := Load(path, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Plots) != PlotCount {
		t.Errorf("the garden came back with %d beds", len(g.Plots))
	}
	for i, p := range g.Plots {
		if p.PH < 4 || p.PH > 9 {
			t.Errorf("bed %d came back with pH %.2f", i+1, p.PH)
		}
		if p.Richness <= 0 {
			t.Errorf("bed %d came back with no richness", i+1)
		}
	}
}
