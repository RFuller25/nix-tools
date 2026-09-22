package main

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func TestCompostYieldScalesWithThePlant(t *testing.T) {
	seedling := &Plot{Growth: 0.2}
	grown := &Plot{Growth: 1}
	spent := &Plot{Growth: 1, Spent: true}

	basil := SpeciesByID("basil")         // annual
	maple := SpeciesByID("japanesemaple") // tree

	if compostYield(basil, seedling) >= compostYield(basil, grown) {
		t.Error("a seedling should be worth less to the heap than a grown plant")
	}
	if compostYield(basil, grown) >= compostYield(maple, grown) {
		t.Error("a tree should be worth more than a basil plant")
	}
	if compostYield(basil, spent) <= compostYield(basil, grown) {
		t.Error("a plant gone to seed is all dry matter and should be worth most")
	}
	for _, sp := range AllSpecies() {
		if y := compostYield(sp, grown); y <= 0 || y > 0.5 {
			t.Errorf("%s returns %.2f richness, outside a sensible range", sp.ID, y)
		}
	}
}

func TestLiftingEnrichesTheBed(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Plots[0] = Plot{SpeciesID: "sunflower", Growth: 1, Matured: true, PH: 7.8, Richness: 0.2}

	gain, ok := g.Uproot(0, now)
	if !ok {
		t.Fatal("lifting a plant failed")
	}
	if gain <= 0 {
		t.Errorf("lifting returned %.2f richness", gain)
	}
	if got := g.Plots[0].Richness; got <= 0.2 {
		t.Errorf("the bed is at %.2f richness, no better than before", got)
	}
	if g.Plots[0].PH >= 7.8 {
		t.Error("compost did not buffer the bed's pH towards neutral")
	}
	if g.Composted != 1 {
		t.Errorf("composted count = %d", g.Composted)
	}

	// Richness cannot run past a full bed.
	g.Plots[1] = Plot{SpeciesID: "japanesemaple", Growth: 1, Matured: true, PH: 6.5, Richness: 0.95}
	if _, ok := g.Uproot(1, now); !ok {
		t.Fatal("lifting the tree failed")
	}
	if got := g.Plots[1].Richness; got > 1 {
		t.Errorf("richness reached %.2f", got)
	}

	// An empty bed has nothing to give.
	if gain, ok := g.Uproot(2, now); ok || gain != 0 {
		t.Error("an empty bed yielded compost")
	}
}

func TestCompostAnimationRunsAndEnds(t *testing.T) {
	now := testStart()
	g := newTestGarden(now)
	g.Plots[0] = Plot{SpeciesID: "cosmos", Growth: 1, Matured: true, PH: 6.5, Richness: 0.3}

	m := newModel(g, "", now)
	m.width, m.height = 90, 30
	gain, ok := m.lift(0)
	if !ok || gain <= 0 {
		t.Fatalf("lift returned %.2f, %v", gain, ok)
	}

	fx, running := m.compost[0]
	if !running {
		t.Fatal("lifting did not start the animation")
	}
	if fx.Species.ID != "cosmos" {
		t.Errorf("the animation is showing %s", fx.Species.ID)
	}
	if fx.done(now) {
		t.Error("the animation was over before it began")
	}
	if !fx.done(now.Add(compostDuration)) {
		t.Error("the animation never finishes")
	}

	// Every frame has to fill the bed exactly, or the grid tears.
	for _, at := range []float64{0, 0.1, 0.3, 0.45, 0.6, 0.75, 0.9, 0.99} {
		when := now.Add(time.Duration(at * float64(compostDuration)))
		frame := fx.frame(cellInner, artHeight, when)
		if len(frame) != artHeight {
			t.Fatalf("at %.0f%% the frame is %d lines, want %d", at*100, len(frame), artHeight)
		}
		for _, line := range frame {
			if w := lipgloss.Width(line); w != cellInner {
				t.Fatalf("at %.0f%% a frame line is %d wide, want %d: %q", at*100, w, cellInner, line)
			}
		}
	}

	// The bed keeps drawing while the plant collapses, then goes back to
	// being bare earth.
	during := m.renderCell(0)
	if !strings.Contains(during, "composting") {
		t.Error("the bed does not say what is happening to it")
	}
	m.now = now.Add(2 * compostDuration)
	m.Update(windTickMsg(m.now))
	if _, still := m.compost[0]; still {
		t.Error("the finished animation was not cleared away")
	}
}

func TestSoilLineShowsRichness(t *testing.T) {
	hungry := soilLine(13, 0, true, phaseNoon, 0.05)
	fed := soilLine(13, 0, true, phaseNoon, 0.9)
	if hungry == fed {
		t.Error("hungry ground and well-composted ground look the same")
	}
	for _, r := range []float64{0, 0.3, 0.6, 1} {
		if w := lipgloss.Width(soilLine(13, 0, true, phaseNoon, r)); w != 13 {
			t.Errorf("at richness %.1f the soil line is %d wide", r, w)
		}
	}

	glyph, color := richnessGlyph(0.9)
	if glyph == "" || color == "" {
		t.Error("rich soil has no look of its own")
	}
}
